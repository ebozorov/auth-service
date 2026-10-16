package home

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
)

// LoadOpenAPI reads the OpenAPI document once at startup and fails fast if it is missing or invalid.
func LoadOpenAPI(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("%s: invalid JSON", path)
	}
	return b, nil
}

// OpenAPI serves the API document. It is public in every environment.
func (h *Handler) OpenAPI(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(h.openapi)
}
