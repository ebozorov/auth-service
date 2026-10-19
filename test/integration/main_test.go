// Package integration tests the service from the outside, like a client: real HTTP
// requests to a running server, with Postgres and Mailpit in containers
// (docs/DECISION.md #15). Run with `make test INTEGRATION=1`; Docker is required.
package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"auth-service/internal"
	"auth-service/internal/app"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// Limits are low so the limit tests stay short. Every client has its own IP
// (X-Forwarded-For), so tests do not use up each other's IP limit.
const dynamicConfig = `
sign_in_user_limit: 3
sign_in_ip_limit: 10
`

const (
	adminEmail = "admin@test.local"
	ipLimit    = 10
	userLimit  = 3
)

var (
	baseURL    string   // the server, e.g. http://127.0.0.1:40123
	mailpitURL string   // Mailpit's HTTP API
	db         *gorm.DB // direct access, only to move time forward (expire links and sessions)
)

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		log.Printf("integration: %v", err)
		code = 1
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("auth"),
		postgres.WithUsername("auth"),
		postgres.WithPassword("auth"),
		postgres.BasicWaitStrategies(),
	)
	defer testcontainers.TerminateContainer(pg)
	if err != nil {
		return 0, fmt.Errorf("postgres: %w", err)
	}
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, err
	}

	mp, err := testcontainers.Run(ctx, "axllent/mailpit:v1.31",
		testcontainers.WithExposedPorts("1025/tcp", "8025/tcp"),
		// Without it Mailpit looks up the client's hostname, and each email waits seconds for DNS.
		testcontainers.WithEnv(map[string]string{"MP_SMTP_DISABLE_RDNS": "true"}),
		testcontainers.WithWaitStrategy(wait.ForHTTP("/readyz").WithPort("8025/tcp").WithStartupTimeout(time.Minute)),
	)
	defer testcontainers.TerminateContainer(mp)
	if err != nil {
		return 0, fmt.Errorf("mailpit: %w", err)
	}
	mpHost, err := mp.Host(ctx)
	if err != nil {
		return 0, err
	}
	smtpPort, err := mp.MappedPort(ctx, "1025/tcp")
	if err != nil {
		return 0, err
	}
	httpPort, err := mp.MappedPort(ctx, "8025/tcp")
	if err != nil {
		return 0, err
	}
	mailpitURL = "http://" + net.JoinHostPort(mpHost, httpPort.Port())

	dir, err := os.MkdirTemp("", "auth-service-integration")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	keyPath, configPath := filepath.Join(dir, "private.pem"), filepath.Join(dir, "config.yaml")
	if err := writeKey(keyPath); err != nil {
		return 0, err
	}
	if err := os.WriteFile(configPath, []byte(dynamicConfig), 0o644); err != nil {
		return 0, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	baseURL = "http://" + ln.Addr().String()

	// Built as a struct, not from env: addresses come only from the containers.
	cfg := &internal.Config{
		Env:               internal.EnvTest,
		ConfigFile:        configPath,
		PublicURL:         baseURL,
		TrustProxy:        true,
		DatabaseURL:       dsn,
		MigrationsDir:     "../../migrations",
		JWTPrivateKey:     keyPath,
		JWTIssuer:         baseURL,
		OpenAPIFile:       "../../docs/openapi.json",
		SignInRedirectURL: "/",
		InitialAdminEmail: adminEmail,
		InitialAdminName:  "Admin",
		SMTPHost:          mpHost,
		SMTPPort:          smtpPort.Port(),
		SMTPFrom:          "no-reply@test.local",
	}
	server, err := app.New(cfg)
	if err != nil {
		return 0, err
	}
	go func() {
		if err := server.Serve(ln); err != nil {
			log.Printf("server: %v", err)
		}
	}()
	defer server.Shutdown()

	db, err = internal.ConnectDB(dsn)
	if err != nil {
		return 0, err
	}

	return m.Run(), nil
}

func writeKey(path string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600)
}
