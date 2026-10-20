// Package pgtest gives each test its own throwaway Postgres database.
//
// A test package calls Main from TestMain. Main uses the server at TEST_DATABASE_URL, or starts one Postgres
// container for the package when that is unset, and creates a template database once, migrated with the
// given goose migrations. NewDatabase then clones the template for each test, so tests can run in parallel.
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
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Image is the Postgres image started when TEST_DATABASE_URL is unset. Keep it in step with compose.
const Image = "postgres:18.6"

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
	serverURL, stop, err := startServer(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		return 1
	}
	defer stop()

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

	server.admin, server.url, server.template = admin, serverURL, template
	return m.Run()
}

func startServer(ctx context.Context) (serverURL string, stop func(), err error) {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u, func() {}, nil
	}
	ctr, err := tcpostgres.Run(ctx, Image, tcpostgres.BasicWaitStrategies())
	if err != nil {
		return "", nil, fmt.Errorf("start container: %w", err)
	}
	stop = func() { _ = testcontainers.TerminateContainer(ctr) }
	serverURL, err = ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		stop()
		return "", nil, fmt.Errorf("container url: %w", err)
	}
	return serverURL, stop, nil
}

// migrate applies the migrations and closes every connection, so the database can serve as a template.
func migrate(ctx context.Context, dbURL string, migrations fs.FS) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
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
	pool, err := pgxpool.New(context.Background(), NewDatabaseURL(t))
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
