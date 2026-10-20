// Package pgtest gives each test its own throwaway Postgres database.
//
// A test package calls Main from TestMain. Main uses the server at TEST_DATABASE_URL, or starts one Postgres
// container for the package when that is unset, and creates a template database once. NewDatabase then
// clones the template for each test, so tests can run in parallel.
package pgtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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

// Main runs the package's tests against a Postgres server and exits with their status.
func Main(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	url, stop, err := serverURL(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		return 1
	}
	defer stop()

	admin, err := pgxpool.New(ctx, url)
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

	server.admin, server.url, server.template = admin, url, template
	return m.Run()
}

func serverURL(ctx context.Context) (url string, stop func(), err error) {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url, func() {}, nil
	}
	ctr, err := tcpostgres.Run(ctx, Image, tcpostgres.BasicWaitStrategies())
	if err != nil {
		return "", nil, fmt.Errorf("start container: %w", err)
	}
	stop = func() { _ = testcontainers.TerminateContainer(ctr) }
	url, err = ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		stop()
		return "", nil, fmt.Errorf("container url: %w", err)
	}
	return url, stop, nil
}

// NewDatabase creates a database cloned from the template and returns a pool on it. The database is
// dropped when the test ends.
func NewDatabase(t testing.TB) *pgxpool.Pool {
	t.Helper()
	if server.admin == nil {
		t.Fatal("pgtest: call pgtest.Main(m) from TestMain")
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

	cfg, err := pgxpool.ParseConfig(server.url)
	if err != nil {
		t.Fatalf("pgtest: parse url: %v", err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("pgtest: connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// uniqueName returns a database name that does not collide with other test packages on one server. It is
// lowercase letters, digits and underscores, so it needs no quoting.
func uniqueName(prefix string) string {
	return prefix + "_" + strings.ToLower(rand.Text())
}
