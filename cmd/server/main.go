package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"auth-service/internal"
	"auth-service/internal/app"
)

func main() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	server, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		_ = server.Shutdown()
	}()

	base := cfg.BaseURL()
	log.Printf("%s %s (%s) ishlayapti: %s", app.Name, app.Version, cfg.Env, base)
	// make dev: PUBLIC_URL is the Vite dev server, the embedded web/dist on base is stale there.
	log.Printf("web: %s/", cfg.PublicURL)
	log.Printf("api: %s%s/", base, internal.APIPrefix)
	if cfg.SwaggerUI {
		log.Printf("swagger ui: %s/swagger-ui/", base)
	}

	if err := server.Listen(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
