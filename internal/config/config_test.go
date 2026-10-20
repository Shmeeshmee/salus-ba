package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoad_ReadsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("HTTP_ADDR", ":9000")

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
