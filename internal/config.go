package internal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

// Config holds the static settings: they change only with a restart. Every value
// comes from an environment variable and has a default (see config/.env.local and
// docs/DECISION.md #10). Limits and token lifetimes are in Dynamic (#25).
type Config struct {
	// app
	Env        string
	ConfigFile string
	Host       string
	Port       string
	PublicURL  string
	TrustProxy bool

	// db
	DatabaseURL   string
	MigrationsDir string

	// jwt
	JWTPrivateKey string
	JWTIssuer     string

	// swagger
	SwaggerUI    bool
	SwaggerUIDir string
	OpenAPIFile  string

	// sign-in
	SignInRedirectURL string

	// admin
	InitialAdminEmail string
	InitialAdminName  string

	// smtp
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func LoadConfig() (*Config, error) {
	var errs []error
	boolean := func(key, fallback string) bool {
		b, err := strconv.ParseBool(getEnv(key, fallback))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: must be true or false", key))
		}
		return b
	}

	cfg := &Config{
		Env:        getEnv("APP_ENV", EnvDev),
		Host:       getEnv("APP_HOST", "0.0.0.0"),
		Port:       getEnv("APP_PORT", "5000"),
		ConfigFile: getEnv("CONFIG_FILE", "config/config.yaml"),
		TrustProxy: boolean("TRUST_PROXY", "false"),

		DatabaseURL:   getEnv("DATABASE_URL", "postgres://auth:auth@localhost:5433/auth?sslmode=disable"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),

		JWTPrivateKey: getEnv("JWT_PRIVATE_KEY_PATH", "certs/private.pem"),
		JWTIssuer:     getEnv("JWT_ISSUER", "http://localhost:5000"),

		SwaggerUIDir: getEnv("SWAGGER_UI_DIR", "swagger-ui"),
		OpenAPIFile:  getEnv("OPENAPI_FILE", "docs/openapi.json"),

		SignInRedirectURL: getEnv("SIGN_IN_REDIRECT_URL", "/"),

		InitialAdminName: getEnv("INITIAL_ADMIN_NAME", "Admin"),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "no-reply@localhost"),
	}
	cfg.PublicURL = getEnv("PUBLIC_URL", cfg.BaseURL())

	switch cfg.Env {
	case EnvDev, EnvTest, EnvProd:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV: must be one of %s, %s, %s", EnvDev, EnvTest, EnvProd))
	}

	// Swagger UI is for dev/test only: on by default there, never allowed in prod.
	cfg.SwaggerUI = boolean("SWAGGER_UI_ENABLED", strconv.FormatBool(cfg.Env != EnvProd))
	if cfg.SwaggerUI && cfg.Env == EnvProd {
		errs = append(errs, fmt.Errorf("SWAGGER_UI_ENABLED: not allowed when APP_ENV=%s", EnvProd))
	}

	// Dev gets a ready admin so it runs without make; other envs must set it explicitly.
	initialAdminEmail := ""
	if cfg.Env == EnvDev {
		initialAdminEmail = "admin@localhost"
	}
	cfg.InitialAdminEmail = getEnv("INITIAL_ADMIN_EMAIL", initialAdminEmail)

	// Without SMTP, sign-in links are only written to the log. Fine for dev, never for prod.
	if cfg.SMTPHost == "" && cfg.Env == EnvProd {
		errs = append(errs, fmt.Errorf("SMTP_HOST: required when APP_ENV=%s", EnvProd))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

// ListenAddr is the address the server binds to.
func (c *Config) ListenAddr() string {
	return net.JoinHostPort(c.Host, c.Port)
}

// BaseURL is the address to open in a browser; wildcard hosts are shown as localhost.
func (c *Config) BaseURL() string {
	host := c.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, c.Port)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
