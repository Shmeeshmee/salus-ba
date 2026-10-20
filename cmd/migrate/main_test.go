package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The template stays empty: these tests apply the migrations themselves.
func TestMain(m *testing.M) { pgtest.Main(m, nil) }

func TestRun_UpAppliesMigrationsAndStatusListsThemApplied(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	var out bytes.Buffer

	if err := run(t.Context(), []string{"up"}, &out); err != nil {
		t.Fatalf("up: %v", err)
	}
	out.Reset()
	if err := run(t.Context(), []string{"status"}, &out); err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out.String(), "applied  00001_extensions.sql") {
		t.Errorf("status output does not list 00001_extensions.sql as applied:\n%s", out.String())
	}
	if strings.Contains(out.String(), "pending") {
		t.Errorf("status output still has pending migrations after up:\n%s", out.String())
	}
}

func TestRun_StatusListsPendingBeforeUp(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	var out bytes.Buffer

	if err := run(t.Context(), []string{"status"}, &out); err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out.String(), "pending  00001_extensions.sql") {
		t.Errorf("status output does not list 00001_extensions.sql as pending:\n%s", out.String())
	}
}

func TestRun_CreateWritesTheNextNumberedFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"00001_first.sql", "00007_seventh.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- +goose Up\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer

	if err := run(t.Context(), []string{"create", "-dir", dir, "add_thing"}, &out); err != nil {
		t.Fatalf("create: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "00008_add_thing.sql"))
	if err != nil {
		t.Fatalf("next file not written: %v", err)
	}
	if !strings.HasPrefix(string(body), "-- +goose Up\n") {
		t.Errorf("new file must start with -- +goose Up, got %q", body)
	}
}

func TestRun_CreateRejectsBadNames(t *testing.T) {
	for _, name := range []string{"", "Add-Thing", "../escape"} {
		if err := run(t.Context(), []string{"create", "-dir", t.TempDir(), name}, &bytes.Buffer{}); err == nil {
			t.Errorf("create %q: error = nil, want an error", name)
		}
	}
}

func TestRun_RejectsUnknownCommand(t *testing.T) {
	if err := run(t.Context(), []string{"down"}, &bytes.Buffer{}); err == nil {
		t.Fatal("run(down) error = nil, want an error: migrations are forward-only")
	}
}
