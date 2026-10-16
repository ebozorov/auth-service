package web

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"auth-service/internal"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes serves the UI at the root. Register it last: it answers every GET path
// not taken by other routes. Paths that are not files get index.html, so the React router
// handles them (e.g. /users opened directly).
func RegisterRoutes(router fiber.Router) error {
	files := FS()
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return fmt.Errorf("web: %w (run `make web-build`)", err)
	}

	router.Get("/*", func(c *fiber.Ctx) error {
		// An unknown API path stays a JSON 404, not the UI.
		if c.Path() == internal.APIPrefix || strings.HasPrefix(c.Path(), internal.APIPrefix+"/") {
			return c.Next()
		}

		name := c.Params("*")
		if data, err := fs.ReadFile(files, name); name != "" && err == nil {
			// Files in assets/ have a content hash in their name, so they never change.
			if strings.HasPrefix(name, "assets/") {
				c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
			} else {
				c.Set(fiber.HeaderCacheControl, "no-cache")
			}
			c.Type(path.Ext(name))
			return c.Send(data)
		}

		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Type("html")
		return c.Send(index)
	})
	return nil
}
