package home

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"auth-service/internal"
	"auth-service/internal/auth"

	"github.com/gofiber/fiber/v2"
)

func TestLoadOpenAPI(t *testing.T) {
	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"openapi":`), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"missing": filepath.Join(dir, "missing.json"),
		"invalid": invalid,
	} {
		if _, err := LoadOpenAPI(path); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

// The document shipped in the repo must load, or the server would not start.
func TestRepoOpenAPIIsValid(t *testing.T) {
	b, err := LoadOpenAPI("../../docs/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		OpenAPI string         `json:"openapi"`
		Paths   map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.OpenAPI == "" || len(doc.Paths) == 0 {
		t.Fatalf("openapi %q, %d paths", doc.OpenAPI, len(doc.Paths))
	}
}

func TestSwaggerUINeedsFiles(t *testing.T) {
	if err := registerSwaggerUI(fiber.New(), t.TempDir()); err == nil {
		t.Fatal("want error without index.html")
	}
}

func TestWellKnown(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "private.pem")
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := internal.LoadDynamic(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := auth.NewTokenManager(keyPath, "https://auth.example.com", settings)
	if err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	h := NewHandler(nil, tokens, []byte(`{}`), Options{})
	if err := RegisterRoutes(app, app.Group(internal.APIPrefix), h); err != nil {
		t.Fatal(err)
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	if err != nil {
		t.Fatal(err)
	}
	var oidc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&oidc); err != nil {
		t.Fatal(err)
	}
	if oidc["issuer"] != "https://auth.example.com" || oidc["jwks_uri"] != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("got %v", oidc)
	}
	if resp.Header.Get(fiber.HeaderCacheControl) != wellKnownCacheControl {
		t.Errorf("cache-control %q", resp.Header.Get(fiber.HeaderCacheControl))
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/.well-known/jwks.json", nil))
	if err != nil {
		t.Fatal(err)
	}
	var jwks auth.JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].Kid == "" {
		t.Errorf("got %+v", jwks)
	}
	if resp.Header.Get(fiber.HeaderCacheControl) != wellKnownCacheControl {
		t.Errorf("cache-control %q", resp.Header.Get(fiber.HeaderCacheControl))
	}

	// Swagger UI is off, so it is not mounted.
	resp, err = app.Test(httptest.NewRequest("GET", "/swagger-ui/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("swagger ui: got %d, want 404", resp.StatusCode)
	}
}

func TestOpenAPIServed(t *testing.T) {
	app := fiber.New()
	h := NewHandler(nil, nil, []byte(`{"openapi":"3.1.0"}`), Options{})
	app.Get("/openapi.json", h.OpenAPI)
	resp, err := app.Test(httptest.NewRequest("GET", "/openapi.json", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get(fiber.HeaderContentType) != fiber.MIMEApplicationJSON {
		t.Errorf("content-type %q", resp.Header.Get(fiber.HeaderContentType))
	}
}
