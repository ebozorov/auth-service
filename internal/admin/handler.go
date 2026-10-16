package admin

import (
	"errors"
	"net/mail"
	"strings"

	"auth-api/internal"
	"auth-api/internal/session"
	"auth-api/internal/user"

	"github.com/gofiber/fiber/v2"
)

// Handler serves /admin: managing users and their devices (docs/DECISION.md #9).
type Handler struct {
	users    *user.Service
	sessions *session.Service
}

func NewHandler(users *user.Service, sessions *session.Service) *Handler {
	return &Handler{users: users, sessions: sessions}
}

type createUserBody struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// updateUserBody fields are optional: only the ones sent are changed.
type updateUserBody struct {
	Name    *string `json:"name"`
	Role    *string `json:"role"`
	Enabled *bool   `json:"enabled"`
}

// RequireAdmin lets only admins through. The role is read from the database on every
// request, so a role change takes effect right away. Use after auth.RequireAuth.
func (h *Handler) RequireAdmin(c *fiber.Ctx) error {
	u, err := h.users.Get(internal.UserID(c))
	if err != nil && !errors.Is(err, user.ErrNotFound) {
		return err
	}
	if u == nil || u.Role != user.RoleAdmin {
		return fiber.NewError(fiber.StatusForbidden, "admin only")
	}
	return c.Next()
}

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	users, err := h.users.List()
	if err != nil {
		return err
	}
	return c.JSON(users)
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var body createUserBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Role == "" {
		body.Role = user.RoleUser
	}
	if _, err := mail.ParseAddress(user.NormalizeEmail(body.Email)); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}
	if strings.TrimSpace(body.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if !user.IsValidRole(body.Role) {
		return fiber.NewError(fiber.StatusBadRequest, "role must be admin or user")
	}

	u, err := h.users.Create(body.Email, body.Name, body.Role)
	if errors.Is(err, user.ErrEmailTaken) {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(u)
}

// UpdateUser changes name, role or enabled. A disabled user loses all sessions
// (docs/DECISION.md #7). Admins cannot demote or disable themselves, so at least one admin remains.
func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	u, err := h.userFromParam(c)
	if err != nil {
		return err
	}

	var body updateUserBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	adminID := internal.UserID(c)
	isSelf := u.ID == adminID
	if body.Name != nil {
		if strings.TrimSpace(*body.Name) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name is required")
		}
		u.Name = strings.TrimSpace(*body.Name)
	}
	if body.Role != nil {
		if !user.IsValidRole(*body.Role) {
			return fiber.NewError(fiber.StatusBadRequest, "role must be admin or user")
		}
		if isSelf && *body.Role != user.RoleAdmin {
			return fiber.NewError(fiber.StatusBadRequest, "you cannot remove your own admin role")
		}
		u.Role = *body.Role
	}
	if body.Enabled != nil {
		if isSelf && !*body.Enabled {
			return fiber.NewError(fiber.StatusBadRequest, "you cannot disable yourself")
		}
		u.Enabled = *body.Enabled
	}

	if err := h.users.Save(u); err != nil {
		return err
	}
	if !u.Enabled {
		ended := session.EndedBy(session.ReasonDisabled, adminID, internal.SessionID(c))
		if _, err := h.sessions.EndAll(u.ID, ended); err != nil {
			return err
		}
	}
	return c.JSON(u)
}

func (h *Handler) ListSessions(c *fiber.Ctx) error {
	u, err := h.userFromParam(c)
	if err != nil {
		return err
	}
	sessions, err := h.sessions.List(u.ID)
	if err != nil {
		return err
	}
	return c.JSON(session.NewResponses(sessions, internal.SessionID(c)))
}

func (h *Handler) TerminateSession(c *fiber.Ctx) error {
	u, err := h.userFromParam(c)
	if err != nil {
		return err
	}
	sid, err := internal.ParamID(c, "sid")
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, session.ErrNotFound.Error())
	}

	ended := session.EndedBy(session.ReasonTerminated, internal.UserID(c), internal.SessionID(c))
	err = h.sessions.End(sid, u.ID, ended)
	if errors.Is(err, session.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// userFromParam loads the user from the :id path parameter, or returns 404.
func (h *Handler) userFromParam(c *fiber.Ctx) (*user.User, error) {
	id, err := internal.ParamID(c, "id")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, user.ErrNotFound.Error())
	}
	u, err := h.users.Get(id)
	if errors.Is(err, user.ErrNotFound) {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return u, err
}
