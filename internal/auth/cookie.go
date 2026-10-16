package auth

import (
	"time"

	"auth-service/internal"

	"github.com/gofiber/fiber/v2"
)

// Browser cookies (docs/DECISION.md #4, #12). Both are HttpOnly, so JavaScript cannot read them,
// and SameSite=Lax, so they are sent when the user opens the link from the email.
const (
	deviceSecretCookie = "device_secret"
	deviceSecretPath   = internal.APIPrefix + "/auth/sign-in"

	refreshTokenCookie = "refresh_token"
	refreshTokenPath   = internal.APIPrefix + "/auth"
)

func setCookie(c *fiber.Ctx, name, value, path string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  expires,
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func clearCookie(c *fiber.Ctx, name, path string) {
	setCookie(c, name, "", path, time.Unix(0, 0))
}
