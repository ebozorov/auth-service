package session

import (
	"errors"

	"auth-service/internal"

	"github.com/gofiber/fiber/v2"
)

// Handler serves /sessions: the signed-in user's own devices.
type Handler struct {
	sessions *Service
}

func NewHandler(sessions *Service) *Handler {
	return &Handler{sessions: sessions}
}

func (h *Handler) List(c *fiber.Ctx) error {
	sessions, err := h.sessions.List(internal.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(NewResponses(sessions, internal.SessionID(c)))
}

func (h *Handler) Terminate(c *fiber.Ctx) error {
	id, err := internal.ParamID(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, ErrNotFound.Error())
	}

	userID := internal.UserID(c)
	err = h.sessions.End(id, userID, EndedBy(ReasonTerminated, userID, internal.SessionID(c)))
	if errors.Is(err, ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) TerminateOthers(c *fiber.Ctx) error {
	userID, current := internal.UserID(c), internal.SessionID(c)
	n, err := h.sessions.EndOthers(userID, current, EndedBy(ReasonTerminated, userID, current))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"terminated": n})
}
