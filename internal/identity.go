package internal

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

const (
	localUserID    = "userID"
	localSessionID = "sessionID"
)

// SetIdentity stores the authenticated user and session on the request.
func SetIdentity(c *fiber.Ctx, userID, sessionID int64) {
	c.Locals(localUserID, userID)
	c.Locals(localSessionID, sessionID)
}

// UserID returns the authenticated user. Only valid behind the auth middleware.
func UserID(c *fiber.Ctx) int64 { return c.Locals(localUserID).(int64) }

// SessionID returns the current session. Only valid behind the auth middleware.
func SessionID(c *fiber.Ctx) int64 { return c.Locals(localSessionID).(int64) }

// ParamID reads a numeric path parameter, e.g. /sessions/:id.
func ParamID(c *fiber.Ctx, name string) (int64, error) {
	return strconv.ParseInt(c.Params(name), 10, 64)
}
