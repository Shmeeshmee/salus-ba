// Package pgtest gives each test its own throwaway Postgres database.
//
// A test package calls Main from TestMain. Main uses the Postgres server at TEST_DATABASE_URL (the compose
// Postgres when run through `make test`) and creates a template database once, migrated with the given goose
// migrations. NewDatabase then clones the template for each test, so tests can run in parallel. Without
// TEST_DATABASE_URL the tests fail; they never skip.
//
// NewAppDatabase also connects as the application role (AppRole, created by the migrations) with the
// password in TEST_APP_DATABASE_PASSWORD, so a test sees exactly what the api and worker see: grants and
// row-level security apply. NewRetentionDatabase connects as the retention job's role (RetentionRole) with
// the password in TEST_RETENTION_DATABASE_PASSWORD.
package pgtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// AppRole is the role the application binaries log in as: no superuser, no BYPASSRLS, owner of nothing.
const AppRole = "salus_app"

// RetentionRole is the retention job's role: BYPASSRLS, no superuser, owner of nothing, granted only SELECT
// and DELETE on the tables whose rows are deleted after their period.
const RetentionRole = "salus_retention"

// The variables that hold the passwords Main sets on the roles, and the tests log in with.
const (
	appPasswordVar       = "TEST_APP_DATABASE_PASSWORD"
	retentionPasswordVar = "TEST_RETENTION_DATABASE_PASSWORD"
)

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
	for role, variable := range map[string]string{AppRole: appPasswordVar, RetentionRole: retentionPasswordVar} {
		if err := setRolePassword(ctx, admin, role, os.Getenv(variable)); err != nil {
			fmt.Fprintln(os.Stderr, "pgtest: set the", role, "password:", err)
			return 1
		}
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
	return newRoleDatabase(t, AppRole, appPasswordVar)
}

// NewRetentionDatabase is NewAppDatabase for RetentionRole, with the password Main set from
// TEST_RETENTION_DATABASE_PASSWORD.
func NewRetentionDatabase(t testing.TB) (owner, retention *pgxpool.Pool) {
	t.Helper()
	return newRoleDatabase(t, RetentionRole, retentionPasswordVar)
}

// AppURL returns dbURL logged in as AppRole, for a test that needs it beside another role's login.
func AppURL(t testing.TB, dbURL string) string {
	t.Helper()
	return roleURL(t, dbURL, AppRole, appPasswordVar)
}

// RetentionURL returns dbURL logged in as RetentionRole, for a test that hands a binary its URL.
func RetentionURL(t testing.TB, dbURL string) string {
	t.Helper()
	return roleURL(t, dbURL, RetentionRole, retentionPasswordVar)
}

func newRoleDatabase(t testing.TB, role, passwordVar string) (owner, as *pgxpool.Pool) {
	t.Helper()
	dbURL := NewDatabaseURL(t)
	return connect(t, dbURL), connect(t, roleURL(t, dbURL, role, passwordVar))
}

// roleURL returns dbURL with role and the password in the variable passwordVar.
func roleURL(t testing.TB, dbURL, role, passwordVar string) string {
	t.Helper()
	password := os.Getenv(passwordVar)
	if password == "" {
		t.Fatal("pgtest: " + passwordVar + " is not set; run the tests with `make test`, which sets it for the compose Postgres")
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal("pgtest: the database URL does not parse")
	}
	u.User = url.UserPassword(role, password)
	return u.String()
}

// setRolePassword sets role's password when the migrations created the role and password is not empty.
// Roles are cluster-wide and every test package sets the same value (in the compose stack, the one
// cmd/migrate sets too); SetPassword's lock on the server's database orders the packages.
func setRolePassword(ctx context.Context, admin *pgxpool.Pool, role, password string) error {
	var exists bool
	if err := admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", role).Scan(&exists); err != nil {
		return err
	}
	if !exists || password == "" {
		return nil
	}
	return postgres.SetPassword(ctx, admin, role, password)
}

// PoolMaxConns caps every pool pgtest opens. go test runs up to one package per CPU at once, each with
// parallel tests and a pool per database; pools sized by the CPU count (pgx's default) could together pass the
// server's connection limit on a large host. A test that measures concurrency opens its own pool on
// NewDatabaseURL with the size it needs.
const PoolMaxConns = 8

func connect(t testing.TB, dbURL string) *pgxpool.Pool {
	t.Helper()
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal("pgtest: the database URL does not parse")
	}
	q := u.Query()
	q.Set("pool_max_conns", strconv.Itoa(PoolMaxConns))
	u.RawQuery = q.Encode()
	pool, err := postgres.Open(context.Background(), u.String())
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

// HoldRoleLock takes postgres.RoleLock on the server's database until release is called or the test ends, for
// a test that sets a role's password on a database of its own (cmd/migrate's): SetPassword's lock orders only
// the callers on one database, and every package's Main sets the roles' passwords on the server's, which then
// waits for the test instead of failing with "tuple concurrently updated".
func HoldRoleLock(t testing.TB) (release func()) {
	t.Helper()
	if server.admin == nil {
		t.Fatal("pgtest: call pgtest.Main(m, migrations) from TestMain")
	}
	ctx := context.Background()
	conn, err := server.admin.Acquire(ctx)
	if err != nil {
		t.Fatalf("pgtest: hold the role lock: %v", err)
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext($1))", postgres.RoleLock); err != nil {
		conn.Release()
		t.Fatalf("pgtest: hold the role lock: %v", err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			// Closing the session releases its advisory lock, so the pool never hands out a connection that holds it.
			if err := conn.Conn().Close(ctx); err != nil {
				t.Errorf("pgtest: release the role lock: %v", err)
			}
			conn.Release()
		})
	}
	t.Cleanup(release)
	return release
}

// WaitForLockWait returns once a session on db's database waits on a lock (a row another transaction holds),
// for a test of what a lock makes wait. It fails the test when none does within lockWaitTimeout. db must see
// the other sessions' activity: the owner pool does.
func WaitForLockWait(t testing.TB, db *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), lockWaitTimeout)
	defer cancel()
	for {
		var waiting bool
		err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock')`).Scan(&waiting)
		switch {
		case err != nil:
			t.Fatalf("pgtest: no session waited on a lock: %v", err)
		case waiting:
			return
		}
	}
}

// lockWaitTimeout bounds WaitForLockWait.
const lockWaitTimeout = 30 * time.Second
