package auth

import (
	"errors"
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"time"

	"auth-api/internal"
	"auth-api/internal/session"
	"auth-api/internal/user"

	"github.com/gofiber/fiber/v2"
)

// SignIn takes an email and sends a magic link to it.
// The answer is the same whether the email is known or not (docs/DECISION.md #7).
func (h *Handler) SignIn(c *fiber.Ctx) error {
	ip := c.IP()
	if h.limiter.Exceeded(ip) {
		return fiber.ErrTooManyRequests
	}
	h.limiter.Add(ip)

	var body signInBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	email := user.NormalizeEmail(body.Email)
	if _, err := mail.ParseAddress(email); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}

	// The device secret ties the link to this device (docs/DECISION.md #4).
	secret, secretHash := internal.NewToken()
	device := session.DetectDevice(body.Device, c.Get(fiber.HeaderUserAgent))
	if err := h.startSignIn(email, secretHash, device, ip); err != nil {
		return err
	}

	setCookie(c, deviceSecretCookie, secret, deviceSecretPath, time.Now().Add(h.cfg.RequestTTL))
	return c.Status(fiber.StatusAccepted).JSON(signInResponse{
		DeviceSecret: secret,
		ExpiresIn:    int(h.cfg.RequestTTL.Seconds()),
	})
}

// startSignIn stores the request and emails the link. Unknown or disabled users,
// and users over their limit, are skipped silently.
func (h *Handler) startSignIn(email, secretHash string, device session.Device, ip string) error {
	u, err := h.users.GetByEmail(email)
	if errors.Is(err, user.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !u.Enabled {
		return nil
	}

	var recent int64
	err = h.db.Model(&signinRequest{}).
		Where("user_id = ? AND requested_at > ?", u.ID, time.Now().Add(-h.cfg.UserWindow)).
		Count(&recent).Error
	if err != nil {
		return err
	}
	if recent >= int64(h.cfg.UserLimit) {
		return nil
	}

	token, tokenHash := internal.NewToken()
	now := time.Now()
	req := signinRequest{
		ID:          internal.NewID(),
		UserID:      u.ID,
		TokenHash:   tokenHash,
		SecretHash:  secretHash,
		Device:      device,
		RequestedIP: ip,
		RequestedAt: now,
		ExpiresAt:   now.Add(h.cfg.RequestTTL),
	}
	if err := h.db.Create(&req).Error; err != nil {
		return err
	}

	// Sent in the background, so the response time does not show whether the email exists.
	go h.sendLink(u.Email, token)
	return nil
}

func (h *Handler) sendLink(email, token string) {
	link := h.cfg.PublicURL + internal.APIPrefix + "/auth/sign-in/link?token=" + url.QueryEscape(token)
	body := fmt.Sprintf(
		"Tizimga kirish uchun linkni oching:\n\n%s\n\n"+
			"Link %d daqiqa amal qiladi va faqat kirish so'ralgan qurilmada ishlaydi.\n"+
			"Agar siz so'ramagan bo'lsangiz, bu xatni e'tiborsiz qoldiring.\n",
		link, int(h.cfg.RequestTTL.Minutes()))

	if err := h.mailer.Send(email, "Tizimga kirish", body); err != nil {
		log.Printf("sign-in email to %s: %v", email, err)
	}
}

// LinkBrowser is where the link from the email leads. The device secret comes from the cookie.
// On success the refresh token is set as a cookie and the browser is redirected (docs/DECISION.md #12).
func (h *Handler) LinkBrowser(c *fiber.Ctx) error {
	ip := c.IP()
	if h.limiter.Exceeded(ip) {
		return c.Redirect(h.redirectURL("too_many_requests"))
	}

	sess, refresh, err := h.useLink(c.Query("token"), c.Cookies(deviceSecretCookie), ip)
	if errors.Is(err, session.ErrInvalidLink) {
		h.limiter.Add(ip)
		return c.Redirect(h.redirectURL("invalid_link"))
	}
	if err != nil {
		return err
	}

	clearCookie(c, deviceSecretCookie, deviceSecretPath)
	setCookie(c, refreshTokenCookie, refresh, refreshTokenPath, sess.ExpiresAt)
	return c.Redirect(h.redirectURL(""))
}

// LinkApp is used by apps and curl: the token and device secret come in the body,
// the tokens are returned in the body.
func (h *Handler) LinkApp(c *fiber.Ctx) error {
	ip := c.IP()
	if h.limiter.Exceeded(ip) {
		return fiber.ErrTooManyRequests
	}

	var body linkBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	sess, refresh, err := h.useLink(body.Token, body.DeviceSecret, ip)
	if errors.Is(err, session.ErrInvalidLink) {
		h.limiter.Add(ip)
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return err
	}
	return h.sendTokens(c, sess, refresh)
}

// useLink starts a session on the device that asked for the link.
// The link works only together with that device's secret (docs/DECISION.md #4).
func (h *Handler) useLink(token, secret, ip string) (*session.Session, string, error) {
	if token == "" || secret == "" {
		return nil, "", session.ErrInvalidLink
	}
	return h.sessions.Start(internal.HashToken(token), internal.HashToken(secret), ip)
}

// redirectURL is SIGN_IN_REDIRECT_URL, with ?error=code when code is not empty.
func (h *Handler) redirectURL(code string) string {
	if code == "" {
		return h.cfg.RedirectURL
	}
	u, err := url.Parse(h.cfg.RedirectURL)
	if err != nil {
		return h.cfg.RedirectURL
	}
	q := u.Query()
	q.Set("error", code)
	u.RawQuery = q.Encode()
	return u.String()
}

// DeleteOldRequests removes signin requests that never became a session. Rows are kept
// for the per-user limit window, because the limit counts them. Called periodically.
func (h *Handler) DeleteOldRequests() (int64, error) {
	keep := max(h.cfg.UserWindow, h.cfg.RequestTTL)
	result := h.db.Where("requested_at < ?", time.Now().Add(-keep)).Delete(&signinRequest{})
	return result.RowsAffected, result.Error
}
