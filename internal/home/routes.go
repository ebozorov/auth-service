package home

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts info and the OpenAPI document under api, and /.well-known/* and
// Swagger UI at the root: discovery must sit next to the issuer URL.
func RegisterRoutes(root, api fiber.Router, h *Handler) error {
	api.Get("/", h.Info)
	api.Get("/openapi.json", h.OpenAPI)

	wk := root.Group("/.well-known")
	wk.Get("/jwks.json", h.JWKS)
	wk.Get("/openid-configuration", h.OpenIDConfiguration)

	if h.opts.SwaggerUI {
		return registerSwaggerUI(root, h.opts.SwaggerUIDir)
	}
	return nil
}
