package user

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts /users. requireAuth comes from the auth package;
// Authorize then checks the caller's role against the method.
func RegisterRoutes(router fiber.Router, h *Handler, requireAuth fiber.Handler) {
	g := router.Group("/users", requireAuth, h.Authorize)
	g.Get("/", h.List)
	g.Post("/", h.Create)
	g.Patch("/:id", h.Update)
	g.Get("/:id/sessions", h.ListSessions)
	g.Delete("/:id/sessions/:sid", h.TerminateSession)
}
