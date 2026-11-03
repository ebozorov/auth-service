package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// captureLog points slog's default at a buffer for the test and returns the parsed lines.
func captureLog(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	return func() []map[string]any {
		var recs []map[string]any
		for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
			var rec map[string]any
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatalf("not JSON: %q", line)
			}
			recs = append(recs, rec)
		}
		return recs
	}
}

func testApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Use(requestid.New(), logRequest, recover.New())
	app.Get("/link", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/bad", func(*fiber.Ctx) error { return fiber.NewError(fiber.StatusBadRequest, "invalid email") })
	app.Get("/broken", func(*fiber.Ctx) error { return errors.New("db is down") })
	app.Get("/panic", func(*fiber.Ctx) error { panic("boom") })
	return app
}

func TestLogRequest(t *testing.T) {
	for _, tc := range []struct {
		path   string
		status int
		level  string
		err    string
	}{
		{"/link", 200, "INFO", ""},
		{"/bad", 400, "INFO", "invalid email"},
		{"/broken", 500, "ERROR", "db is down"},
		{"/panic", 500, "ERROR", "boom"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			lines := captureLog(t)
			resp, err := testApp().Test(httptest.NewRequest("GET", tc.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.status {
				t.Fatalf("status %d, want %d", resp.StatusCode, tc.status)
			}

			recs := lines()
			if len(recs) != 1 {
				t.Fatalf("%d log lines, want 1: %v", len(recs), recs)
			}
			rec := recs[0]
			if rec["msg"] != "request" || rec["level"] != tc.level || rec["status"] != float64(tc.status) {
				t.Errorf("record %v", rec)
			}
			if rec["request_id"] == nil || rec["request_id"] == "" {
				t.Error("no request_id")
			}
			if got, _ := rec["error"].(string); !strings.Contains(got, tc.err) || (tc.err == "" && got != "") {
				t.Errorf("error %q, want %q", got, tc.err)
			}
		})
	}
}

// The sign-in link carries its token in the query string; it must never reach the log.
func TestLogRequestSkipsQuery(t *testing.T) {
	lines := captureLog(t)
	if _, err := testApp().Test(httptest.NewRequest("GET", "/link?token=secret-token", nil)); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(lines())
	if strings.Contains(string(raw), "secret-token") {
		t.Errorf("token in the log: %s", raw)
	}
	if !strings.Contains(string(raw), `"path":"/link"`) {
		t.Errorf("path missing: %s", raw)
	}
}
