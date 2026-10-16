package session

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts /sessions. requireAuth comes from the auth package.
func RegisterRoutes(router fiber.Router, h *Handler, requireAuth fiber.Handler) {
	g := router.Group("/sessions", requireAuth)
	g.Get("/", h.List)
	g.Delete("/", h.TerminateOthers)
	g.Delete("/:id", h.Terminate)
}
