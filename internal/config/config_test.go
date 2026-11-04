package config_test

import (
	"encoding/base64"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoad_ReadsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("APP_DATABASE_PASSWORD", "synthetic")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://test@localhost/test" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.HTTPAddr != ":9000" {
		t.Errorf("HTTPAddr = %q, want :9000", cfg.HTTPAddr)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.AppDatabasePassword != "synthetic" {
		t.Error("AppDatabasePassword is not read from APP_DATABASE_PASSWORD")
	}
}

func TestLoad_DefaultsHTTPAddr(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("HTTP_ADDR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
}

func TestLoad_RequiresDatabaseURLWithoutEchoingValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_ADDR", "secret-looking-value")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error for the missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not name DATABASE_URL", err)
	}
	if strings.Contains(err.Error(), "secret-looking-value") {
		t.Errorf("error %q leaks an environment value", err)
	}
}

func TestLoad_ReadsTheAuthKeysAndTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("TOTP_KEY", base64.StdEncoding.EncodeToString([]byte("synthetic 32-byte key for tests!")))
	t.Setenv("RATE_LIMIT_KEY", base64.StdEncoding.EncodeToString([]byte("synthetic rate-limit key, 32 b.")))
	t.Setenv("TRUSTED_PROXY_CIDRS", " 10.0.0.0/8, fd00::/8 ,")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(cfg.TOTPKey) != "synthetic 32-byte key for tests!" || string(cfg.RateLimitKey) != "synthetic rate-limit key, 32 b." {
		t.Error("TOTPKey or RateLimitKey is not the base64-decoded variable")
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	if !slices.Equal(cfg.TrustedProxies, want) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func TestLoad_RejectsMalformedAuthSettingsWithoutEchoingThem(t *testing.T) {
	for name, value := range map[string]string{
		"TOTP_KEY":            "not*base64*secret",
		"RATE_LIMIT_KEY":      "not*base64*secret",
		"TRUSTED_PROXY_CIDRS": "10.0.0.0/8,not*base64*secret",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
			t.Setenv(name, value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), name) || strings.Contains(err.Error(), "secret") {
				t.Errorf("Load() error = %v, want one naming %s without its value", err, name)
			}
		})
	}
}

func TestLoad_ReadsTheNotifiersSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("SMTP_ADDR", "mailpit:1025")
	t.Setenv("WEB_BASE_URL", "http://localhost:5173")

	cfg, err := config.Load()
	if err != nil || cfg.SMTPAddr != "mailpit:1025" || cfg.WebBaseURL != "http://localhost:5173" {
		t.Errorf("Load() = SMTPAddr %q, WebBaseURL %q, %v", cfg.SMTPAddr, cfg.WebBaseURL, err)
	}
}
