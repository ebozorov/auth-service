package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth-service/internal"
	"auth-service/internal/admin"
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
)

const (
	appName    = "auth-service"
	appVersion = "0.2.0"
)

// cleanupInterval is how often expired sessions and old sign-in requests are cleaned up.
const cleanupInterval = time.Hour

func main() {
	startedAt := time.Now()

	cfg, err := internal.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := internal.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := internal.Migrate(db, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := internal.InitID(); err != nil {
		log.Fatalf("id: %v", err)
	}

	tokens, err := auth.NewTokenManager(cfg.JWTPrivateKey, cfg.JWTIssuer, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatalf("jwt: %v", err)
	}

	openapi, err := home.LoadOpenAPI(cfg.OpenAPIFile)
	if err != nil {
		log.Fatalf("openapi: %v", err)
	}

	users := user.NewService(db)
	sessions := session.NewService(db, cfg.RefreshTokenTTL)
	ensureAdmin(users, cfg)

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
		RequestTTL:  cfg.SignInRequestTTL,
		UserLimit:   cfg.SignInUserLimit,
		UserWindow:  cfg.SignInUserWindow,
		IPLimit:     cfg.SignInIPLimit,
		IPWindow:    cfg.SignInIPWindow,
	}, db, tokens, users, sessions, m)

	homeHandler := home.NewHandler(db, tokens, openapi, home.Options{
		Name:         appName,
		Version:      appVersion,
		Env:          cfg.Env,
		StartedAt:    startedAt,
		SwaggerUI:    cfg.SwaggerUI,
		SwaggerUIDir: cfg.SwaggerUIDir,
	})

	app := fiber.New(fiber.Config{
		AppName:               appName,
		ErrorHandler:          errorHandler,
		ProxyHeader:           proxyHeader(cfg.TrustProxy),
		DisableStartupMessage: true,
	})
	// user and sid come from the auth middleware's locals; empty on public routes.
	app.Use(recover.New(), logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | user=${locals:userID} sid=${locals:sessionID} | ${error}\n",
	}))
	// gzip/br/deflate by Accept-Encoding; JSON shrinks ~15x.
	app.Use(compress.New())

	api := app.Group(internal.APIPrefix)
	if err := home.RegisterRoutes(app, api, homeHandler); err != nil {
		log.Fatalf("home: %v", err)
	}
	auth.RegisterRoutes(api, authHandler)
	session.RegisterRoutes(api, session.NewHandler(sessions), authHandler.RequireAuth)
	admin.RegisterRoutes(api, admin.NewHandler(users, sessions), authHandler.RequireAuth)
	// Last: the web UI answers every path not taken above.
	if err := web.RegisterRoutes(app); err != nil {
		log.Fatalf("web: %v", err)
	}

	go cleanup(sessions, authHandler)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		_ = app.Shutdown()
	}()

	base := cfg.BaseURL()
	log.Printf("%s %s (%s) ishlayapti: %s", appName, appVersion, cfg.Env, base)
	// make dev: PUBLIC_URL is the Vite dev server, the embedded web/dist on base is stale there.
	log.Printf("web: %s/", cfg.PublicURL)
	log.Printf("api: %s%s/", base, internal.APIPrefix)
	if cfg.SwaggerUI {
		log.Printf("swagger ui: %s/swagger-ui/", base)
	}

	if err := app.Listen(cfg.ListenAddr()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// ensureAdmin creates the first admin from INITIAL_ADMIN_EMAIL (docs/DECISION.md #9).
func ensureAdmin(users *user.Service, cfg *internal.Config) {
	if cfg.InitialAdminEmail == "" {
		return
	}
	created, err := users.EnsureAdmin(cfg.InitialAdminEmail, cfg.InitialAdminName)
	if err != nil {
		log.Fatalf("initial admin: %v", err)
	}
	if created {
		log.Printf("birinchi admin yaratildi: %s", cfg.InitialAdminEmail)
	}
}

// cleanup moves expired sessions to the archive and deletes old sign-in requests, once an hour.
func cleanup(sessions *session.Service, authHandler *auth.Handler) {
	for {
		if _, err := sessions.EndExpired(); err != nil {
			log.Printf("cleanup sessions: %v", err)
		}
		if _, err := authHandler.DeleteOldRequests(); err != nil {
			log.Printf("cleanup sign-in requests: %v", err)
		}
		time.Sleep(cleanupInterval)
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
