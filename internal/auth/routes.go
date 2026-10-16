package auth

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts /auth (docs/ARCHITECTURE.md, "Endpointlar").
func RegisterRoutes(router fiber.Router, h *Handler) {
	g := router.Group("/auth")
	g.Post("/sign-in", h.SignIn)
	g.Get("/sign-in/link", h.LinkBrowser)
	g.Post("/sign-in/link", h.LinkApp)
	g.Post("/refresh", h.Refresh)
	g.Post("/logout", h.RequireAuth, h.Logout)
	g.Get("/me", h.RequireAuth, h.Me)
}
