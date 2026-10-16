package auth

import (
	"errors"
	"strconv"
	"strings"

	"auth-api/internal"
	"auth-api/internal/session"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth checks the Bearer access token and that its session is still active,
// so a terminated device loses access right away.
func (h *Handler) RequireAuth(c *fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}

	claims, err := h.tokens.ParseAccess(token)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	sess, err := h.sessions.FindActive(claims.SessionID, userID)
	if errors.Is(err, session.ErrNotFound) {
		return fiber.NewError(fiber.StatusUnauthorized, "session terminated")
	}
	if err != nil {
		return err
	}

	h.sessions.Touch(sess, c.IP())
	internal.SetIdentity(c, userID, sess.ID)
	return c.Next()
}
