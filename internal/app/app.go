// Package app builds the HTTP server. cmd/server runs it, the integration tests
// start it on their own config (docs/DECISION.md #15).
package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"auth-service/internal"
	"auth-service/internal/auth"
	"auth-service/internal/home"
	"auth-service/internal/mailer"
	"auth-service/internal/session"
	"auth-service/internal/user"
	"auth-service/web"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const (
	Name    = "auth-service"
	Version = "0.3.1"
)

// cleanupInterval is how often expired sessions and old sign-in requests are cleaned up.
const cleanupInterval = time.Hour

// Server is the whole service: the database is migrated and the routes are mounted.
type Server struct {
	cfg         *internal.Config
	app         *fiber.App
	sessions    *session.Service
	authHandler *auth.Handler
}

// New connects to the database, applies migrations and mounts all routes.
// It does not listen yet; call Listen or Serve.
func New(cfg *internal.Config) (*Server, error) {
	startedAt := time.Now()

	settings, err := internal.LoadDynamic(cfg.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("dynamic config: %w", err)
	}
	// Without a watch the server still runs, only with the values read at startup.
	if err := settings.Watch(); err != nil {
		slog.Warn("dynamic config: fayl kuzatilmaydi", "path", cfg.ConfigFile, "error", err)
	}

	db, err := internal.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	if err := internal.Migrate(db, cfg.MigrationsDir); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	if err := internal.InitID(); err != nil {
		return nil, fmt.Errorf("id: %w", err)
	}

	tokens, err := auth.NewTokenManager(cfg.JWTPrivateKey, cfg.JWTIssuer, settings)
	if err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}

	openapi, err := home.LoadOpenAPI(cfg.OpenAPIFile)
	if err != nil {
		return nil, fmt.Errorf("openapi: %w", err)
	}

	users := user.NewService(db)
	sessions := session.NewService(db, settings)
	if err := ensureAdmin(users, cfg); err != nil {
		return nil, fmt.Errorf("initial admin: %w", err)
	}

	m := mailer.New(mailer.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})

	authHandler := auth.NewHandler(auth.Config{
		PublicURL:   cfg.PublicURL,
		RedirectURL: cfg.SignInRedirectURL,
	}, settings, db, tokens, users, sessions, m)

	homeHandler := home.NewHandler(db, tokens, openapi, home.Options{
		Name:         Name,
		Version:      Version,
		Env:          cfg.Env,
		StartedAt:    startedAt,
		SwaggerUI:    cfg.SwaggerUI,
		SwaggerUIDir: cfg.SwaggerUIDir,
	})

	app := fiber.New(fiber.Config{
		AppName:               Name,
		ErrorHandler:          errorHandler,
		ProxyHeader:           proxyHeader(cfg.TrustProxy),
		DisableStartupMessage: true,
	})
	// request_id ties the request log to other logs of the same request; requestid must run before the logger.
	// An X-Request-ID from upstream is kept; otherwise a Sonyflake id, time-ordered like the other ids.
	app.Use(requestid.New(requestid.Config{Generator: newRequestID}))
	app.Use(logRequest)
	// After the logger: a panic becomes an error that the request log shows.
	app.Use(recover.New())
	// gzip/br/deflate by Accept-Encoding; JSON shrinks ~15x.
	app.Use(compress.New())

	api := app.Group(internal.APIPrefix)
	if err := home.RegisterRoutes(app, api, homeHandler); err != nil {
		return nil, fmt.Errorf("home: %w", err)
	}
	auth.RegisterRoutes(api, authHandler)
	session.RegisterRoutes(api, session.NewHandler(sessions), authHandler.RequireAuth)
	user.RegisterRoutes(api, user.NewHandler(users, sessions), authHandler.RequireAuth)
	// Last: the web UI answers every path not taken above.
	if err := web.RegisterRoutes(app); err != nil {
		return nil, fmt.Errorf("web: %w", err)
	}

	return &Server{cfg: cfg, app: app, sessions: sessions, authHandler: authHandler}, nil
}

// Listen serves on cfg.ListenAddr until Shutdown.
func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr())
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve starts the periodic cleanup and serves on ln until Shutdown.
func (s *Server) Serve(ln net.Listener) error {
	go s.cleanup()
	return s.app.Listener(ln)
}

// Shutdown lets running requests finish, then stops the server.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// ensureAdmin creates the first admin from INITIAL_ADMIN_EMAIL (docs/DECISION.md #9).
func ensureAdmin(users *user.Service, cfg *internal.Config) error {
	if cfg.InitialAdminEmail == "" {
		return nil
	}
	created, err := users.EnsureAdmin(cfg.InitialAdminEmail, cfg.InitialAdminName)
	if err != nil {
		return err
	}
	if created {
		slog.Info("birinchi admin yaratildi", "email", internal.MaskEmail(cfg.InitialAdminEmail))
	}
	return nil
}

// cleanup moves expired sessions to the archive and deletes old sign-in requests, once an hour.
func (s *Server) cleanup() {
	for {
		if _, err := s.sessions.EndExpired(); err != nil {
			slog.Error("cleanup: sessions", "error", err)
		}
		if _, err := s.authHandler.DeleteOldRequests(); err != nil {
			slog.Error("cleanup: sign-in requests", "error", err)
		}
		time.Sleep(cleanupInterval)
	}
}

func newRequestID() string {
	return strconv.FormatInt(internal.NewID(), 10)
}

// logRequest writes one line per request (docs/DECISION.md #26). It logs c.Path(), never
// the query string: the sign-in link carries its token there. Headers and bodies are not
// logged either (cookies, tokens, emails). Like Fiber's logger it runs the error handler
// itself, so the line shows the status the client gets; 5xx is logged as an error.
func logRequest(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	if err != nil {
		if herr := c.App().ErrorHandler(c, err); herr != nil {
			_ = c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	status := c.Response().StatusCode()
	attrs := []slog.Attr{
		slog.Any("request_id", c.Locals(requestid.ConfigDefault.ContextKey)),
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", status),
		slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
		slog.String("ip", c.IP()),
		slog.Int("bytes", responseSize(c)),
	}
	level := slog.LevelInfo
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	if status >= fiber.StatusInternalServerError {
		level = slog.LevelError
	}
	slog.LogAttrs(c.UserContext(), level, "request", attrs...)
	return nil
}

// responseSize is the body size that is actually sent; -1 when it is unknown. The logger
// runs before compress, so it is the compressed size, the bytes that went over the wire.
func responseSize(c *fiber.Ctx) int {
	res := c.Response()
	status := res.StatusCode()
	// fasthttp drops the body of these, e.g. the "No Content" text that SendStatus(204) sets.
	if c.Method() == fiber.MethodHead || status < 200 || status == fiber.StatusNoContent || status == fiber.StatusNotModified {
		return 0
	}
	if res.IsBodyStream() {
		// Static files are streamed; Content-Length is -1 when the size is unknown (chunked).
		return res.Header.ContentLength()
	}
	return len(res.Body())
}

func proxyHeader(trust bool) string {
	if trust {
		return fiber.HeaderXForwardedFor
	}
	return ""
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"

	// Other errors are not shown to the client; logRequest logs them.
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
