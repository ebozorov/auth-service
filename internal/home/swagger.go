package home

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// registerSwaggerUI serves Swagger UI from dir at /swagger-ui/ (dev/test only).
func registerSwaggerUI(router fiber.Router, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return fmt.Errorf("swagger ui: %w (run `make swagger-ui`)", err)
	}

	// Assets are referenced relatively, so the UI must be served from /swagger-ui/.
	router.Get("/swagger-ui", func(c *fiber.Ctx) error {
		if c.Path() == "/swagger-ui" {
			return c.Redirect("/swagger-ui/", fiber.StatusMovedPermanently)
		}
		return c.Next()
	})
	// no-cache: the browser revalidates every time (304 if unchanged), so an edited
	// swagger-initializer.js is never served stale from the heuristic cache.
	router.Static("/swagger-ui", dir, fiber.Static{
		ModifyResponse: func(c *fiber.Ctx) error {
			c.Set(fiber.HeaderCacheControl, "no-cache")
			return nil
		},
	})
	return nil
}
