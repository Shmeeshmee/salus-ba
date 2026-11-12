package main

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The template stays empty: these tests apply the migrations themselves.
func TestMain(m *testing.M) { pgtest.Main(m, nil) }

func TestRun_UpAppliesMigrationsAndStatusListsThemApplied(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	t.Setenv("APP_DATABASE_PASSWORD", "") // provisioned outside migrate: the roles are left alone
	t.Setenv("RETENTION_DATABASE_PASSWORD", "")
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

// roleSecret reads the stored password verifier of role (pg_authid needs the superuser the compose Postgres
// gives the tests). SCRAM verifiers are salted, so setting even the same password changes it.
func roleSecret(t *testing.T, dbURL, role string) string {
	t.Helper()
	pool, err := postgres.Open(t.Context(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var secret string
	if err := pool.QueryRow(t.Context(), "SELECT coalesce(rolpassword, '') FROM pg_authid WHERE rolname = $1", role).Scan(&secret); err != nil {
		t.Fatal(err)
	}
	return secret
}

func TestRun_UpSetsTheAppRolePasswordAndTheRoleCanLogIn(t *testing.T) {
	dbURL := pgtest.NewDatabaseURL(t)
	// up sets the password on this test's database, out of reach of the lock every other package's Main takes
	// on the server's: hold that lock too, so they wait instead of failing.
	pgtest.HoldRoleLock(t)
	// The role is cluster-wide: set the value every other test package uses, or their logins would break.
	password := os.Getenv("TEST_APP_DATABASE_PASSWORD")
	if password == "" {
		t.Fatal("TEST_APP_DATABASE_PASSWORD is not set; run the tests with `make test`")
	}
	t.Setenv("DATABASE_URL", dbURL)
	t.Setenv("APP_DATABASE_PASSWORD", password)
	if err := run(t.Context(), []string{"up"}, &bytes.Buffer{}); err != nil { // creates the role on a new server
		t.Fatalf("up: %v", err)
	}
	before := roleSecret(t, dbURL, pgtest.AppRole)
	var out bytes.Buffer

	if err := run(t.Context(), []string{"up"}, &out); err != nil {
		t.Fatalf("up: %v", err)
	}
	if roleSecret(t, dbURL, pgtest.AppRole) == before {
		t.Error("up did not set the salus_app password")
	}
	if strings.Contains(out.String(), password) {
		t.Errorf("up output holds the password:\n%s", out.String())
	}
	u, _ := url.Parse(dbURL)
	u.User = url.UserPassword(pgtest.AppRole, password)
	pool, err := postgres.Open(t.Context(), u.String())
	if err != nil {
		t.Fatalf("log in as %s: %v", pgtest.AppRole, err)
	}
	pool.Close()
}

func TestRun_UpSetsTheRetentionRolePasswordAndTheRoleCanLogIn(t *testing.T) {
	dbURL := pgtest.NewDatabaseURL(t)
	// up sets the password on this test's database, out of reach of the lock every other package's Main takes
	// on the server's: hold that lock too, so they wait instead of failing.
	pgtest.HoldRoleLock(t)
	// Cluster-wide, like salus_app: the value every other test package logs in with.
	password := os.Getenv("TEST_RETENTION_DATABASE_PASSWORD")
	if password == "" {
		t.Fatal("TEST_RETENTION_DATABASE_PASSWORD is not set; run the tests with `make test`")
	}
	t.Setenv("DATABASE_URL", dbURL)
	t.Setenv("APP_DATABASE_PASSWORD", "")
	t.Setenv("RETENTION_DATABASE_PASSWORD", password)
	if err := run(t.Context(), []string{"up"}, &bytes.Buffer{}); err != nil { // creates the role on a new server
		t.Fatalf("up: %v", err)
	}
	before := roleSecret(t, dbURL, pgtest.RetentionRole)
	var out bytes.Buffer

	if err := run(t.Context(), []string{"up"}, &out); err != nil {
		t.Fatalf("up: %v", err)
	}
	if roleSecret(t, dbURL, pgtest.RetentionRole) == before {
		t.Error("up did not set the salus_retention password")
	}
	if strings.Contains(out.String(), password) || strings.Contains(out.String(), "set the password of "+pgtest.AppRole) {
		t.Errorf("up output holds the password, or set the app role's without APP_DATABASE_PASSWORD:\n%s", out.String())
	}
	pool, err := postgres.Open(t.Context(), pgtest.RetentionURL(t, dbURL))
	if err != nil {
		t.Fatalf("log in as %s: %v", pgtest.RetentionRole, err)
	}
	pool.Close()
}

func TestRun_UpSetsTheStatsRolePasswordAndTheRoleCanLogIn(t *testing.T) {
	dbURL := pgtest.NewDatabaseURL(t)
	// up sets the password on this test's database, out of reach of the lock every other package's Main takes
	// on the server's: hold that lock too, so they wait instead of failing.
	pgtest.HoldRoleLock(t)
	// Cluster-wide, like salus_app: the value every other test package logs in with.
	password := os.Getenv("TEST_STATS_DATABASE_PASSWORD")
	if password == "" {
		t.Fatal("TEST_STATS_DATABASE_PASSWORD is not set; run the tests with `make test`")
	}
	t.Setenv("DATABASE_URL", dbURL)
	t.Setenv("APP_DATABASE_PASSWORD", "")
	t.Setenv("STATS_DATABASE_PASSWORD", password)
	if err := run(t.Context(), []string{"up"}, &bytes.Buffer{}); err != nil { // creates the role on a new server
		t.Fatalf("up: %v", err)
	}
	before := roleSecret(t, dbURL, pgtest.StatsRole)
	var out bytes.Buffer

	if err := run(t.Context(), []string{"up"}, &out); err != nil {
		t.Fatalf("up: %v", err)
	}
	if roleSecret(t, dbURL, pgtest.StatsRole) == before {
		t.Error("up did not set the salus_stats password")
	}
	if strings.Contains(out.String(), password) || strings.Contains(out.String(), "set the password of "+pgtest.AppRole) {
		t.Errorf("up output holds the password, or set the app role's without APP_DATABASE_PASSWORD:\n%s", out.String())
	}
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, dbURL))
	if err != nil {
		t.Fatalf("log in as %s: %v", pgtest.StatsRole, err)
	}
	pool.Close()
}
