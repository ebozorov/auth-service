package home

import (
	"context"
	"time"

	"auth-api/internal"

	"github.com/gofiber/fiber/v2"
)

const healthCheckTimeout = 2 * time.Second

// Info returns basic service information and health. It responds 503 if the database is unreachable.
func (h *Handler) Info(c *fiber.Ctx) error {
	status, dbStatus, code := "ok", "ok", fiber.StatusOK
	if err := h.pingDB(c.Context()); err != nil {
		status, dbStatus, code = "degraded", "unavailable", fiber.StatusServiceUnavailable
	}

	links := map[string]string{
		"web":                  "/",
		"openapi":              internal.APIPrefix + "/openapi.json",
		"jwks":                 "/.well-known/jwks.json",
		"openid_configuration": "/.well-known/openid-configuration",
	}
	if h.opts.SwaggerUI {
		links["swagger_ui"] = "/swagger-ui/"
	}

	uptime := time.Since(h.opts.StartedAt)
	return c.Status(code).JSON(infoResponse{
		Name:          h.opts.Name,
		Version:       h.opts.Version,
		Env:           h.opts.Env,
		Status:        status,
		Checks:        map[string]string{"database": dbStatus},
		StartedAt:     h.opts.StartedAt.UTC().Format(time.RFC3339),
		Uptime:        uptime.Truncate(time.Second).String(),
		UptimeSeconds: int64(uptime.Seconds()),
		Time:          time.Now().UTC().Format(time.RFC3339),
		Links:         links,
	})
}

func (h *Handler) pingDB(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	return sqlDB.PingContext(ctx)
}
