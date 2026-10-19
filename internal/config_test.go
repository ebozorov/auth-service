package internal

import (
	"strings"
	"testing"
)

// configKeys are every variable LoadConfig reads.
var configKeys = []string{
	"APP_ENV", "APP_HOST", "APP_PORT", "CONFIG_FILE", "PUBLIC_URL", "TRUST_PROXY",
	"DATABASE_URL", "MIGRATIONS_DIR", "JWT_PRIVATE_KEY_PATH", "JWT_ISSUER",
	"SWAGGER_UI_ENABLED", "SWAGGER_UI_DIR", "OPENAPI_FILE", "SIGN_IN_REDIRECT_URL",
	"INITIAL_ADMIN_EMAIL", "INITIAL_ADMIN_NAME",
	"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM",
}

// setEnv clears every config variable (empty counts as unset), then sets the given ones.
func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, k := range configKeys {
		t.Setenv(k, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	setEnv(t, nil)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvDev {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDev)
	}
	if cfg.PublicURL != "http://localhost:5000" {
		t.Errorf("PublicURL = %q", cfg.PublicURL)
	}
	if !cfg.SwaggerUI {
		t.Error("SwaggerUI is off in dev")
	}
	if cfg.TrustProxy {
		t.Error("TrustProxy is on by default")
	}
	if cfg.InitialAdminEmail != "admin@localhost" {
		t.Errorf("InitialAdminEmail = %q", cfg.InitialAdminEmail)
	}
}

func TestLoadConfigTestEnv(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": EnvTest})
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitialAdminEmail != "" {
		t.Errorf("InitialAdminEmail = %q, want none outside dev", cfg.InitialAdminEmail)
	}
	if !cfg.SwaggerUI {
		t.Error("SwaggerUI is off in test")
	}
}

func TestLoadConfigProd(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": EnvProd, "SMTP_HOST": "smtp.example.com"})
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SwaggerUI {
		t.Error("SwaggerUI is on in prod")
	}
	if cfg.InitialAdminEmail != "" {
		t.Errorf("InitialAdminEmail = %q, want none outside dev", cfg.InitialAdminEmail)
	}
}

func TestLoadConfigPublicURL(t *testing.T) {
	setEnv(t, map[string]string{"APP_HOST": "127.0.0.1", "APP_PORT": "8080"})
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicURL != "http://127.0.0.1:8080" {
		t.Errorf("PublicURL = %q", cfg.PublicURL)
	}

	setEnv(t, map[string]string{"PUBLIC_URL": "https://auth.example.com"})
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicURL != "https://auth.example.com" {
		t.Errorf("PublicURL = %q", cfg.PublicURL)
	}
}

func TestLoadConfigRejects(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"unknown env":       {map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		"bad trust proxy":   {map[string]string{"TRUST_PROXY": "yes please"}, "TRUST_PROXY"},
		"bad swagger flag":  {map[string]string{"SWAGGER_UI_ENABLED": "on"}, "SWAGGER_UI_ENABLED"},
		"prod without smtp": {map[string]string{"APP_ENV": EnvProd}, "SMTP_HOST"},
		"prod with swagger": {map[string]string{"APP_ENV": EnvProd, "SMTP_HOST": "smtp", "SWAGGER_UI_ENABLED": "true"}, "SWAGGER_UI_ENABLED"},
	} {
		t.Run(name, func(t *testing.T) {
			setEnv(t, tc.env)
			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error about %s", err, tc.want)
			}
		})
	}
}

// All problems are reported at once, not one per restart.
func TestLoadConfigJoinsErrors(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": EnvProd, "TRUST_PROXY": "x", "SWAGGER_UI_ENABLED": "true"})
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("want error")
	}
	for _, key := range []string{"TRUST_PROXY", "SWAGGER_UI_ENABLED", "SMTP_HOST"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error does not mention %s: %v", key, err)
		}
	}
}

func TestConfigAddresses(t *testing.T) {
	for _, tc := range []struct {
		host, listen, base string
	}{
		{"0.0.0.0", "0.0.0.0:5000", "http://localhost:5000"},
		{"", ":5000", "http://localhost:5000"},
		{"::", "[::]:5000", "http://localhost:5000"},
		{"127.0.0.1", "127.0.0.1:5000", "http://127.0.0.1:5000"},
		{"::1", "[::1]:5000", "http://[::1]:5000"},
	} {
		c := Config{Host: tc.host, Port: "5000"}
		if got := c.ListenAddr(); got != tc.listen {
			t.Errorf("host %q: ListenAddr = %q, want %q", tc.host, got, tc.listen)
		}
		if got := c.BaseURL(); got != tc.base {
			t.Errorf("host %q: BaseURL = %q, want %q", tc.host, got, tc.base)
		}
	}
}
