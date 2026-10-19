package auth

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth-service/internal"
	"auth-service/internal/session"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// These cover the paths that answer before the database is touched;
// the rest is left to the integration tests.

func newTestApp(t *testing.T, yaml string) (*fiber.App, *Handler) {
	t.Helper()
	settings := newSettings(t, yaml)
	h := NewHandler(Config{PublicURL: "http://localhost:5000", RedirectURL: "/"}, settings, nil, newTokens(t, testKey()), nil, nil, nil)
	app := fiber.New()
	RegisterRoutes(app, h)
	return app, h
}

func do(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func jsonRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return req
}

func TestSignInRejectsBadInput(t *testing.T) {
	app, _ := newTestApp(t, "")
	for name, body := range map[string]string{
		"broken json":   `{"email":`,
		"no email":      `{}`,
		"invalid email": `{"email":"not-an-email"}`,
		"display name":  `{"email":"Bob <bob@example.com>x"}`,
	} {
		resp, _ := do(t, app, jsonRequest("POST", "/auth/sign-in", body))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, resp.StatusCode)
		}
	}
}

// Every attempt counts toward the IP limit, valid or not.
func TestSignInIPLimit(t *testing.T) {
	app, _ := newTestApp(t, "sign_in_ip_limit: 2\n")
	for i := range 2 {
		resp, _ := do(t, app, jsonRequest("POST", "/auth/sign-in", `{"email":"x"}`))
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("attempt %d: got %d", i+1, resp.StatusCode)
		}
	}
	resp, _ := do(t, app, jsonRequest("POST", "/auth/sign-in", `{"email":"x"}`))
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", resp.StatusCode)
	}
}

func TestLinkAppRejectsMissingToken(t *testing.T) {
	app, h := newTestApp(t, "sign_in_ip_limit: 2\n")
	for name, body := range map[string]string{
		"no token":  `{"device_secret":"s"}`,
		"no secret": `{"token":"t"}`,
	} {
		resp, _ := do(t, app, jsonRequest("POST", "/auth/sign-in/link", body))
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", name, resp.StatusCode)
		}
	}
	// A failed link counts toward the IP limit, so tokens cannot be guessed.
	if !h.limiter.Exceeded("0.0.0.0") {
		t.Fatal("failed links were not counted")
	}
	resp, _ := do(t, app, jsonRequest("POST", "/auth/sign-in/link", `{}`))
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", resp.StatusCode)
	}

	app, _ = newTestApp(t, "")
	resp, _ = do(t, app, jsonRequest("POST", "/auth/sign-in/link", `{"token":`))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("broken json: got %d, want 400", resp.StatusCode)
	}
}

func TestLinkBrowserRedirectsOnError(t *testing.T) {
	app, _ := newTestApp(t, "sign_in_ip_limit: 1\n")

	// No device_secret cookie: the link was opened on another device.
	resp, _ := do(t, app, httptest.NewRequest("GET", "/auth/sign-in/link?token=abc", nil))
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/?error=invalid_link" {
		t.Fatalf("got %d to %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, _ = do(t, app, httptest.NewRequest("GET", "/auth/sign-in/link?token=abc", nil))
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/?error=too_many_requests" {
		t.Fatalf("got %d to %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestRefreshRequiresToken(t *testing.T) {
	app, _ := newTestApp(t, "")
	for name, req := range map[string]*http.Request{
		"no body":     httptest.NewRequest("POST", "/auth/refresh", nil),
		"empty token": jsonRequest("POST", "/auth/refresh", `{"refresh_token":""}`),
	} {
		resp, _ := do(t, app, req)
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, resp.StatusCode)
		}
	}
}

func TestRequireAuthRejects(t *testing.T) {
	app, h := newTestApp(t, "")
	notNumeric := sign(t, h.tokens, Claims{
		SessionID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "alice",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	foreign := must(newTokens(t, otherKey()).IssueAccess(1, 7))

	for name, header := range map[string]string{
		"no header":       "",
		"basic auth":      "Basic dXNlcjpwYXNz",
		"empty bearer":    "Bearer ",
		"lowercase":       "bearer " + must(h.tokens.IssueAccess(1, 7)),
		"garbage":         "Bearer abc",
		"other key":       "Bearer " + foreign,
		"non-numeric sub": "Bearer " + notNumeric,
	} {
		for _, path := range []string{"/auth/me", "/auth/logout"} {
			method := "GET"
			if path == "/auth/logout" {
				method = "POST"
			}
			req := httptest.NewRequest(method, path, nil)
			if header != "" {
				req.Header.Set(fiber.HeaderAuthorization, header)
			}
			resp, _ := do(t, app, req)
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf("%s %s: got %d, want 401", name, path, resp.StatusCode)
			}
		}
	}
}

func TestRedirectURL(t *testing.T) {
	for _, tc := range []struct {
		base, code, want string
	}{
		{"/", "", "/"},
		{"/", "invalid_link", "/?error=invalid_link"},
		{"https://app.example.com/home?tab=1", "invalid_link", "https://app.example.com/home?error=invalid_link&tab=1"},
		{"/?error=old", "too_many_requests", "/?error=too_many_requests"},
		// Unparsable: given back as is rather than a broken URL.
		{"http://[::1", "invalid_link", "http://[::1"},
	} {
		h := &Handler{cfg: Config{RedirectURL: tc.base}}
		if got := h.redirectURL(tc.code); got != tc.want {
			t.Errorf("%q + %q: got %q, want %q", tc.base, tc.code, got, tc.want)
		}
	}
}

func TestUseLinkNeedsTokenAndSecret(t *testing.T) {
	h := &Handler{}
	for _, tc := range [][2]string{{"", ""}, {"t", ""}, {"", "s"}} {
		if _, _, err := h.useLink(tc[0], tc[1], "1.1.1.1"); !errors.Is(err, session.ErrInvalidLink) {
			t.Errorf("token %q, secret %q: got %v", tc[0], tc[1], err)
		}
	}
}

func TestCookies(t *testing.T) {
	expires := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	app := fiber.New()
	app.Get("/set", func(c *fiber.Ctx) error {
		setCookie(c, refreshTokenCookie, "secret", refreshTokenPath, expires)
		return nil
	})
	app.Get("/clear", func(c *fiber.Ctx) error {
		clearCookie(c, refreshTokenCookie, refreshTokenPath)
		return nil
	})

	resp, _ := do(t, app, httptest.NewRequest("GET", "/set", nil))
	c := onlyCookie(t, resp)
	if c.Name != "refresh_token" || c.Value != "secret" || c.Path != internal.APIPrefix+"/auth" {
		t.Errorf("got %s=%s path %s", c.Name, c.Value, c.Path)
	}
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("secure %v, httponly %v, samesite %v", c.Secure, c.HttpOnly, c.SameSite)
	}
	if !c.Expires.Equal(expires) {
		t.Errorf("expires %v, want %v", c.Expires, expires)
	}

	resp, _ = do(t, app, httptest.NewRequest("GET", "/clear", nil))
	c = onlyCookie(t, resp)
	if c.Value != "" || c.Path != internal.APIPrefix+"/auth" || !c.Expires.Before(time.Now()) {
		t.Errorf("not cleared: value %q, path %s, expires %v", c.Value, c.Path, c.Expires)
	}
}

func onlyCookie(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies", len(cookies))
	}
	return cookies[0]
}
