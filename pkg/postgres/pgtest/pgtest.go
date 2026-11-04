// Package pgtest gives each test its own throwaway Postgres database.
//
// A test package calls Main from TestMain. Main uses the Postgres server at TEST_DATABASE_URL (the compose
// Postgres when run through `make test`) and creates a template database once, migrated with the given goose
// migrations. NewDatabase then clones the template for each test, so tests can run in parallel. Without
// TEST_DATABASE_URL the tests fail; they never skip.
//
// NewAppDatabase also connects as the application role (AppRole, created by the migrations) with the
// password in TEST_APP_DATABASE_PASSWORD, so a test sees exactly what the api and worker see: grants and
// row-level security apply.
package pgtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// AppRole is the role the application binaries log in as: no superuser, no BYPASSRLS, owner of nothing.
const AppRole = "salus_app"

// server is set once by Main, before any test runs, and only read afterwards: TestMain has no other way to
// hand it to the tests.
var server struct {
	admin    *pgxpool.Pool
	url      string
	template string
}

// Main runs the package's tests against a Postgres server and exits with their status. The template
// database is migrated with the goose files at the root of migrations; nil leaves it empty.
func Main(m *testing.M, migrations fs.FS) {
	os.Exit(run(m, migrations))
}

func run(m *testing.M, migrations fs.FS) int {
	ctx := context.Background()
	serverURL := os.Getenv("TEST_DATABASE_URL")
	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "pgtest: TEST_DATABASE_URL is not set; run the tests with `make test`, which points it at the compose Postgres")
		return 1
	}

	admin, err := pgxpool.New(ctx, serverURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		return 1
	}
	defer admin.Close()

	template := uniqueName("salus_template")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+template); err != nil {
		fmt.Fprintln(os.Stderr, "pgtest: create template:", err)
		return 1
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP DATABASE "+template+" WITH (FORCE)") }()

	if migrations != nil {
		if err := migrate(ctx, withDatabase(serverURL, template), migrations); err != nil {
			fmt.Fprintln(os.Stderr, "pgtest: migrate template:", err)
			return 1
		}
	}
	if err := setAppPassword(ctx, admin); err != nil {
		fmt.Fprintln(os.Stderr, "pgtest: set the", AppRole, "password:", err)
		return 1
	}

	server.admin, server.url, server.template = admin, serverURL, template
	return m.Run()
}

// migrate applies the migrations and closes every connection, so the database can serve as a template.
func migrate(ctx context.Context, dbURL string, migrations fs.FS) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// NewDatabaseURL creates a database cloned from the template and returns its URL. The database is dropped
// when the test ends.
func NewDatabaseURL(t testing.TB) string {
	t.Helper()
	if server.admin == nil {
		t.Fatal("pgtest: call pgtest.Main(m, migrations) from TestMain")
	}
	ctx := context.Background()
	name := uniqueName("salus_test")
	if _, err := server.admin.Exec(ctx, "CREATE DATABASE "+name+" TEMPLATE "+server.template); err != nil {
		t.Fatalf("pgtest: create database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := server.admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("pgtest: drop database: %v", err)
		}
	})
	return withDatabase(server.url, name)
}

// NewDatabase is NewDatabaseURL with a pool on the new database, closed when the test ends.
func NewDatabase(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return connect(t, NewDatabaseURL(t))
}

// NewAppDatabase creates a database cloned from the template and returns two pools on it, closed when the
// test ends: owner (the server's user, who owns the schema) and app, logged in as AppRole with the password
// Main set from TEST_APP_DATABASE_PASSWORD.
func NewAppDatabase(t testing.TB) (owner, app *pgxpool.Pool) {
	t.Helper()
	password := os.Getenv("TEST_APP_DATABASE_PASSWORD")
	if password == "" {
		t.Fatal("pgtest: TEST_APP_DATABASE_PASSWORD is not set; run the tests with `make test`, which sets it for the compose Postgres")
	}
	dbURL := NewDatabaseURL(t)
	u, _ := url.Parse(dbURL) // built by withDatabase from a URL that parsed
	u.User = url.UserPassword(AppRole, password)
	return connect(t, dbURL), connect(t, u.String())
}

// setAppPassword sets AppRole's password from TEST_APP_DATABASE_PASSWORD when the migrations created the
// role and the variable is set. The role is cluster-wide and every test package sets the same value (in the
// compose stack, the one cmd/migrate sets too); SetPassword retries the race between packages.
func setAppPassword(ctx context.Context, admin *pgxpool.Pool) error {
	password := os.Getenv("TEST_APP_DATABASE_PASSWORD")
	var exists bool
	if err := admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", AppRole).Scan(&exists); err != nil {
		return err
	}
	if !exists || password == "" {
		return nil
	}
	return postgres.SetPassword(ctx, admin, AppRole, password)
}

func connect(t testing.TB, dbURL string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgtest: connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// withDatabase returns serverURL pointing at database name.
func withDatabase(serverURL, name string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		// The URL already connected in Main, so it parses.
		panic(fmt.Sprintf("pgtest: parse server url: %v", err))
	}
	u.Path = "/" + name
	return u.String()
}

// uniqueName returns a database name that does not collide with other test packages on one server. It is
// lowercase letters, digits and underscores, so it needs no quoting.
func uniqueName(prefix string) string {
	return prefix + "_" + strings.ToLower(rand.Text())
}
