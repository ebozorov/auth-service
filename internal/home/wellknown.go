package home

import "github.com/gofiber/fiber/v2"

// Public keys may be cached by other services; they refetch on an unknown kid.
const wellKnownCacheControl = "public, max-age=300"

// JWKS publishes the public signing key so other services can verify access tokens.
func (h *Handler) JWKS(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, wellKnownCacheControl)
	return c.JSON(h.tokens.JWKS())
}

// OpenIDConfiguration is a minimal discovery document pointing to the JWKS.
func (h *Handler) OpenIDConfiguration(c *fiber.Ctx) error {
	issuer := h.tokens.Issuer()
	c.Set(fiber.HeaderCacheControl, wellKnownCacheControl)
	return c.JSON(fiber.Map{
		"issuer":                                issuer,
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}
