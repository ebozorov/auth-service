package internal

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestIdentity(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		SetIdentity(c, 41784020480034629, 7)
		if UserID(c) != 41784020480034629 || SessionID(c) != 7 {
			t.Errorf("got user %d, session %d", UserID(c), SessionID(c))
		}
		return nil
	})
	if _, err := app.Test(httptest.NewRequest("GET", "/", nil)); err != nil {
		t.Fatal(err)
	}
}

func TestParamID(t *testing.T) {
	for path, want := range map[string]struct {
		id int64
		ok bool
	}{
		"/41784020480034629":    {41784020480034629, true},
		"/abc":                  {0, false},
		"/1.5":                  {0, false},
		"/99999999999999999999": {0, false},
	} {
		app := fiber.New()
		app.Get("/:id", func(c *fiber.Ctx) error {
			id, err := ParamID(c, "id")
			if (err == nil) != want.ok || (want.ok && id != want.id) {
				t.Errorf("%s: got %d, %v", path, id, err)
			}
			return nil
		})
		if _, err := app.Test(httptest.NewRequest("GET", path, nil)); err != nil {
			t.Fatal(err)
		}
	}
}
