package auth

import (
	"errors"

	"auth-service/internal"
	"auth-service/internal/mailer"
	"auth-service/internal/session"
	"auth-service/internal/user"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Config holds the static sign-in settings (docs/DECISION.md #10).
// Limits and lifetimes are read from settings on every request (#25).
type Config struct {
	PublicURL   string
	RedirectURL string
}

// Handler serves /auth.
type Handler struct {
	cfg      Config
	settings *internal.DynamicConfig
	db       *gorm.DB
	tokens   *TokenManager
	users    *user.Service
	sessions *session.Service
	mailer   *mailer.Mailer
	limiter  *ipLimiter
}

func NewHandler(cfg Config, settings *internal.DynamicConfig, db *gorm.DB, tokens *TokenManager, users *user.Service, sessions *session.Service, m *mailer.Mailer) *Handler {
	return &Handler{
		cfg:      cfg,
		settings: settings,
		db:       db,
		tokens:   tokens,
		users:    users,
		sessions: sessions,
		mailer:   m,
		limiter:  newIPLimiter(settings),
	}
}

// Refresh rotates the refresh token: the old one stops working.
// Browsers send it in a cookie, apps and curl in the body (docs/DECISION.md #12).
func (h *Handler) Refresh(c *fiber.Ctx) error {
	token := c.Cookies(refreshTokenCookie)
	fromCookie := token != ""
	if !fromCookie {
		var body refreshBody
		_ = c.BodyParser(&body)
		token = body.RefreshToken
	}
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	sess, refresh, err := h.sessions.Rotate(token, c.IP())
	if errors.Is(err, session.ErrInvalidToken) {
		if fromCookie {
			clearCookie(c, refreshTokenCookie, refreshTokenPath)
		}
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return err
	}

	if fromCookie {
		setCookie(c, refreshTokenCookie, refresh, refreshTokenPath, sess.ExpiresAt)
		refresh = ""
	}
	return h.sendTokens(c, sess, refresh)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	userID, sessionID := internal.UserID(c), internal.SessionID(c)
	err := h.sessions.End(sessionID, userID, session.EndedBy(session.ReasonLogout, userID, sessionID))
	if err != nil && !errors.Is(err, session.ErrNotFound) {
		return err
	}
	clearCookie(c, refreshTokenCookie, refreshTokenPath)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	u, err := h.users.Get(internal.UserID(c))
	if errors.Is(err, user.ErrNotFound) {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return err
	}
	return c.JSON(u)
}

func (h *Handler) sendTokens(c *fiber.Ctx, sess *session.Session, refreshToken string) error {
	access, err := h.tokens.IssueAccess(sess.UserID, sess.ID)
	if err != nil {
		return err
	}
	return c.JSON(tokenResponse{
		AccessToken:  access,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.tokens.AccessTTL().Seconds()),
	})
}
