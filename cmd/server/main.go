package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"auth-service/internal"
	"auth-service/internal/app"
)

func main() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fatal("config", err)
	}

	// From here every log line, the standard log package's too, goes through this logger.
	logger, err := internal.NewLogger(os.Stdout, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		fatal("logger", err)
	}
	slog.SetDefault(logger)

	server, err := app.New(cfg)
	if err != nil {
		fatal("server", err)
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		_ = server.Shutdown()
	}()

	base := cfg.BaseURL()
	attrs := []any{
		"version", app.Version,
		"env", cfg.Env,
		"url", base,
		// make dev: PUBLIC_URL is the Vite dev server, the embedded web/dist on base is stale there.
		"web", cfg.PublicURL + "/",
		"api", base + internal.APIPrefix + "/",
	}
	if cfg.SwaggerUI {
		attrs = append(attrs, "swagger_ui", base+"/swagger-ui/")
	}
	slog.Info(app.Name+" ishlayapti", attrs...)

	if err := server.Listen(); err != nil {
		fatal("server", err)
	}
}

// fatal logs the error and exits. Before SetDefault it goes to slog's default text output.
func fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	os.Exit(1)
}
