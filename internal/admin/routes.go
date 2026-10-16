package admin

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts /admin. requireAuth comes from the auth package.
func RegisterRoutes(router fiber.Router, h *Handler, requireAuth fiber.Handler) {
	g := router.Group("/admin", requireAuth, h.RequireAdmin)
	g.Get("/users", h.ListUsers)
	g.Post("/users", h.CreateUser)
	g.Patch("/users/:id", h.UpdateUser)
	g.Get("/users/:id/sessions", h.ListSessions)
	g.Delete("/users/:id/sessions/:sid", h.TerminateSession)
}
