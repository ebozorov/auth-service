package web

import (
	"io"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
)

const indexHTML = `<!doctype html><div id="root"></div>`

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	files := fstest.MapFS{
		"index.html":           {Data: []byte(indexHTML)},
		"favicon.svg":          {Data: []byte("<svg/>")},
		"assets/index-abc.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc.css": {Data: []byte("body{}")},
	}
	app := fiber.New()
	app.Get("/api/v1/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	if err := registerRoutes(app, files); err != nil {
		t.Fatal(err)
	}
	return app
}

func get(t *testing.T, app *fiber.App, path string) (status int, body, contentType, cache string) {
	t.Helper()
	res, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(b), res.Header.Get(fiber.HeaderContentType), res.Header.Get(fiber.HeaderCacheControl)
}

// Paths that are not files get index.html, so the React router handles them.
func TestIndexForAppPaths(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{"/", "/users", "/users/42", "/sign-in?next=/users", "/assets"} {
		status, body, ctype, cache := get(t, app, path)
		if status != 200 || body != indexHTML {
			t.Errorf("%s: got %d %q, want index.html", path, status, body)
		}
		if ctype != "text/html" {
			t.Errorf("%s: Content-Type %q", path, ctype)
		}
		if cache != "no-cache" {
			t.Errorf("%s: Cache-Control %q, want no-cache", path, cache)
		}
	}
}

func TestStaticFiles(t *testing.T) {
	app := newTestApp(t)
	tests := []struct {
		path, body, ctype, cache string
	}{
		{"/assets/index-abc.js", "console.log(1)", "text/javascript", "public, max-age=31536000, immutable"},
		{"/assets/index-abc.css", "body{}", "text/css", "public, max-age=31536000, immutable"},
		// Outside assets/ the name has no hash, so the browser must check it each time.
		{"/favicon.svg", "<svg/>", "image/svg+xml", "no-cache"},
		{"/index.html", indexHTML, "text/html", "no-cache"},
	}
	for _, tt := range tests {
		status, body, ctype, cache := get(t, app, tt.path)
		if status != 200 || body != tt.body {
			t.Errorf("%s: got %d %q, want %q", tt.path, status, body, tt.body)
		}
		if ctype != tt.ctype {
			t.Errorf("%s: Content-Type %q, want %q", tt.path, ctype, tt.ctype)
		}
		if cache != tt.cache {
			t.Errorf("%s: Cache-Control %q, want %q", tt.path, cache, tt.cache)
		}
	}
}

// An unknown API path stays a 404 (the app's error handler makes it JSON), not the UI.
func TestAPIPathsAreNotUI(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{"/api/v1", "/api/v1/", "/api/v1/unknown", "/api/v1/users/42/x"} {
		status, body, _, _ := get(t, app, path)
		if status != fiber.StatusNotFound || body == indexHTML {
			t.Errorf("%s: got %d %q, want 404", path, status, body)
		}
	}
	// Routes registered before the UI still answer.
	if status, body, _, _ := get(t, app, "/api/v1/health"); status != 200 || body != "ok" {
		t.Errorf("/api/v1/health: got %d %q", status, body)
	}
	// Only the /api/v1 prefix itself is reserved, not paths that merely start with it.
	if status, body, _, _ := get(t, app, "/api/v1x"); status != 200 || body != indexHTML {
		t.Errorf("/api/v1x: got %d %q, want index.html", status, body)
	}
}

// A file from an old build gets a 404, so the browser does not run index.html as a script.
func TestMissingAsset(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{"/assets/index-old.js", "/assets/sub/x.css", "/assets/../index.html"} {
		status, body, _, _ := get(t, app, path)
		if status != fiber.StatusNotFound || body == indexHTML {
			t.Errorf("%s: got %d %q, want 404", path, status, body)
		}
	}
}

func TestMissingIndex(t *testing.T) {
	if err := registerRoutes(fiber.New(), fstest.MapFS{}); err == nil {
		t.Fatal("want error without index.html")
	}
}

// The embedded build must contain index.html, or the server would not start.
func TestEmbeddedBuild(t *testing.T) {
	if err := RegisterRoutes(fiber.New()); err != nil {
		t.Fatal(err)
	}
}
