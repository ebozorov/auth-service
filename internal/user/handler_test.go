package user

import (
	"net/http/httptest"
	"strings"
	"testing"

	"auth-service/internal"

	"github.com/gofiber/fiber/v2"
)

func TestAllowed(t *testing.T) {
	for _, tc := range []struct {
		role, method string
		want         bool
	}{
		{RoleAdmin, fiber.MethodGet, true},
		{RoleAdmin, fiber.MethodPost, true},
		{RoleAdmin, fiber.MethodPatch, true},
		{RoleAdmin, fiber.MethodDelete, true},
		{RoleUser, fiber.MethodGet, true},
		{RoleUser, fiber.MethodHead, true},
		{RoleUser, fiber.MethodPost, false},
		{RoleUser, fiber.MethodPatch, false},
		{RoleUser, fiber.MethodPut, false},
		{RoleUser, fiber.MethodDelete, false},
		{"", fiber.MethodGet, false},
		{"Admin", fiber.MethodGet, false},
		{"auditor", fiber.MethodGet, false},
		{"auditor", fiber.MethodPost, false},
	} {
		if got := Allowed(tc.role, tc.method); got != tc.want {
			t.Errorf("%q %s: got %v, want %v", tc.role, tc.method, got, tc.want)
		}
	}
}

// These cover the checks made before the database is touched;
// Authorize and the rest are left to the integration tests.

func newTestApp() *fiber.App {
	h := NewHandler(nil, nil)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		internal.SetIdentity(c, 1, 1)
		return c.Next()
	})
	app.Post("/users", h.Create)
	app.Patch("/users/:id", h.Update)
	app.Get("/users/:id/sessions", h.ListSessions)
	app.Delete("/users/:id/sessions/:sid", h.TerminateSession)
	return app
}

func TestCreateValidates(t *testing.T) {
	app := newTestApp()
	for name, tc := range map[string]struct {
		body, msg string
	}{
		"broken json":   {`{"email":`, "invalid request body"},
		"no email":      {`{"name":"Bob"}`, "invalid email"},
		"invalid email": {`{"email":"bob","name":"Bob"}`, "invalid email"},
		"no name":       {`{"email":"bob@example.com"}`, "name is required"},
		"blank name":    {`{"email":"bob@example.com","name":"   "}`, "name is required"},
		"unknown role":  {`{"email":"bob@example.com","name":"Bob","role":"root"}`, "role must be admin or user"},
	} {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(tc.body))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 100)
		n, _ := resp.Body.Read(b)
		if resp.StatusCode != fiber.StatusBadRequest || string(b[:n]) != tc.msg {
			t.Errorf("%s: got %d %q, want 400 %q", name, resp.StatusCode, b[:n], tc.msg)
		}
	}
}

// A path id that is not a number is a missing user or session, not a bad request.
func TestNonNumericIDIsNotFound(t *testing.T) {
	app := newTestApp()
	for _, tc := range []struct{ method, path string }{
		{"PATCH", "/users/abc"},
		{"GET", "/users/abc/sessions"},
		{"DELETE", "/users/abc/sessions/1"},
	} {
		resp, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("%s %s: got %d, want 404", tc.method, tc.path, resp.StatusCode)
		}
	}
}
