// Package app builds the HTTP server. cmd/server runs it, the integration tests
// start it on their own config (docs/DECISION.md #15).
package app

import (
	"errors"
	"fmt"
	"log"
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
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const (
	Name    = "auth-service"
	Version = "0.3.0"
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
		log.Printf("dynamic config: %s kuzatilmaydi: %v", cfg.ConfigFile, err)
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
	app.Use(recover.New())
	// rid ties the request log to other logs of the same request; requestid must run before logger.
	// An X-Request-ID from upstream is kept; otherwise a Sonyflake id, time-ordered like the other ids.
	app.Use(requestid.New(requestid.Config{Generator: newRequestID}))
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${latency} | ${ip} | ${locals:requestid} | ${status} | ${method} | ${path} | ${error}\n",
		CustomTags: map[string]logger.LogFunc{
			logger.TagMethod: logMethod,
			logger.TagIP:     logIP,
			logger.TagPath:   logPath,
			logger.TagError:  logErrorOrSize,
		},
	}))
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
		log.Printf("birinchi admin yaratildi: %s", cfg.InitialAdminEmail)
	}
	return nil
}

// cleanup moves expired sessions to the archive and deletes old sign-in requests, once an hour.
func (s *Server) cleanup() {
	for {
		if _, err := s.sessions.EndExpired(); err != nil {
			log.Printf("cleanup sessions: %v", err)
		}
		if _, err := s.authHandler.DeleteOldRequests(); err != nil {
			log.Printf("cleanup sign-in requests: %v", err)
		}
		time.Sleep(cleanupInterval)
	}
}

func newRequestID() string {
	return strconv.FormatInt(internal.NewID(), 10)
}

// logMethod writes the method padded to 7 characters (OPTIONS is the longest), so the
// columns after it line up, like ${status} and ${latency} do. Colors match Fiber's own
// ${method}; Fiber turns them on because Output is the default os.Stdout.
func logMethod(output logger.Buffer, c *fiber.Ctx, _ *logger.Data, _ string) (int, error) {
	colors := c.App().Config().ColorScheme
	return fmt.Fprintf(output, "%s%-7s%s", methodColor(c.Method(), colors), c.Method(), colors.Reset)
}

// logIP writes the client IP padded to 15 characters, the longest IPv4 address, so the
// columns after it line up. IPv6 addresses are longer and push the line.
func logIP(output logger.Buffer, c *fiber.Ctx, _ *logger.Data, _ string) (int, error) {
	return fmt.Fprintf(output, "%-15s", c.IP())
}

// logPath writes the path padded to 36 characters, enough for most API paths.
// Longer ones (e.g. /api/v1/admin/users/{id}/sessions/{sid}) push the line; they are not cut.
func logPath(output logger.Buffer, c *fiber.Ctx, _ *logger.Data, _ string) (int, error) {
	return fmt.Fprintf(output, "%-36s", c.Path())
}

// logErrorOrSize writes the error in red like Fiber's ${error}; on success it writes the
// response body size instead of "-", to compare requests. The logger runs before compress,
// so it is the compressed size, the bytes that went over the wire.
func logErrorOrSize(output logger.Buffer, c *fiber.Ctx, data *logger.Data, _ string) (int, error) {
	if data.ChainErr != nil {
		colors := c.App().Config().ColorScheme
		return fmt.Fprintf(output, "%s%s%s", colors.Red, data.ChainErr.Error(), colors.Reset)
	}
	return output.WriteString(formatSize(responseSize(c)))
}

// responseSize is the body size that is actually sent; -1 when it is unknown.
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

func formatSize(n int) string {
	switch {
	case n < 0:
		return "stream"
	case n < 1<<10:
		return fmt.Sprintf("%dB", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1fKB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	}
}

// methodColor is a copy of Fiber's unexported logger.methodColor.
func methodColor(method string, colors fiber.Colors) string {
	switch method {
	case fiber.MethodGet:
		return colors.Cyan
	case fiber.MethodPost:
		return colors.Green
	case fiber.MethodPut:
		return colors.Yellow
	case fiber.MethodDelete:
		return colors.Red
	case fiber.MethodPatch:
		return colors.White
	case fiber.MethodHead:
		return colors.Magenta
	case fiber.MethodOptions:
		return colors.Blue
	default:
		return colors.Reset
	}
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

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	} else {
		log.Printf("error: %v", err)
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
