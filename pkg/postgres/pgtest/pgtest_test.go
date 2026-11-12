package pgtest_test

import (
	"context"
	"crypto/rand"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A small migration set proves the template is migrated before it is cloned; the second file creates the
// application role, the retention role and the stats role the way migrations/ does (roles are cluster-wide, so a parallel
// package may create them first, with these same attributes) and lets them read probe.
func TestMain(m *testing.M) {
	pgtest.Main(m, fstest.MapFS{
		"00001_probe.sql": {Data: []byte("-- +goose Up\nCREATE TABLE probe (id int);\n")},
		"00002_app_role.sql": {Data: []byte(`-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN CREATE ROLE salus_app LOGIN; EXCEPTION WHEN duplicate_object OR unique_violation THEN NULL; END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN CREATE ROLE salus_retention LOGIN NOSUPERUSER BYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN NULL; END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN CREATE ROLE salus_stats LOGIN NOSUPERUSER BYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN NULL; END $$;
-- +goose StatementEnd
GRANT SELECT ON probe TO salus_app, salus_retention, salus_stats;
`)},
	})
}

func TestNewDatabase_IsMigrated(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	var exists bool
	if err := db.QueryRow(t.Context(), "SELECT to_regclass('probe') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("the migration's table is missing from the test database")
	}
}

func TestNewDatabaseURL_PointsAtItsOwnDatabase(t *testing.T) {
	t.Parallel()
	a, b := pgtest.NewDatabaseURL(t), pgtest.NewDatabaseURL(t)

	if a == b {
		t.Fatalf("both URLs are %q, want two databases", a)
	}
}

func TestNewDatabase_GivesEachTestItsOwnDatabase(t *testing.T) {
	t.Parallel()
	a, b := pgtest.NewDatabase(t), pgtest.NewDatabase(t)

	var nameA, nameB string
	if err := a.QueryRow(t.Context(), "SELECT current_database()").Scan(&nameA); err != nil {
		t.Fatal(err)
	}
	if err := b.QueryRow(t.Context(), "SELECT current_database()").Scan(&nameB); err != nil {
		t.Fatal(err)
	}
	if nameA == nameB {
		t.Fatalf("both pools use database %q, want two databases", nameA)
	}
}

func TestNewDatabase_DatabasesAreIsolated(t *testing.T) {
	t.Parallel()
	a, b := pgtest.NewDatabase(t), pgtest.NewDatabase(t)

	if _, err := a.Exec(t.Context(), "CREATE TABLE only_in_a (id int)"); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := b.QueryRow(t.Context(), "SELECT to_regclass('only_in_a') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("a table created in one test database is visible in another")
	}
}

func TestNewAppDatabase_ConnectsToOneDatabaseAsOwnerAndAsTheAppRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)

	var ownerDB, appDB, appUser string
	if err := owner.QueryRow(t.Context(), "SELECT current_database()").Scan(&ownerDB); err != nil {
		t.Fatal(err)
	}
	if err := app.QueryRow(t.Context(), "SELECT current_database(), current_user").Scan(&appDB, &appUser); err != nil {
		t.Fatal(err)
	}
	if appUser != pgtest.AppRole {
		t.Errorf("app pool user = %q, want %q", appUser, pgtest.AppRole)
	}
	if ownerDB != appDB {
		t.Errorf("owner pool on %q, app pool on %q, want one database", ownerDB, appDB)
	}
	if _, err := app.Exec(t.Context(), "SELECT id FROM probe"); err != nil {
		t.Errorf("app role cannot use its grant: %v", err)
	}
}

func TestNewRetentionDatabase_ConnectsToOneDatabaseAsOwnerAndAsTheRetentionRole(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)

	var ownerDB, retentionDB, retentionUser string
	if err := owner.QueryRow(t.Context(), "SELECT current_database()").Scan(&ownerDB); err != nil {
		t.Fatal(err)
	}
	if err := retention.QueryRow(t.Context(), "SELECT current_database(), current_user").Scan(&retentionDB, &retentionUser); err != nil {
		t.Fatal(err)
	}
	if retentionUser != pgtest.RetentionRole {
		t.Errorf("retention pool user = %q, want %q", retentionUser, pgtest.RetentionRole)
	}
	if ownerDB != retentionDB {
		t.Errorf("owner pool on %q, retention pool on %q, want one database", ownerDB, retentionDB)
	}
}

func TestNewStatsDatabase_ConnectsToOneDatabaseAsOwnerAndAsTheStatsRole(t *testing.T) {
	t.Parallel()
	owner, stats := pgtest.NewStatsDatabase(t)

	var ownerDB, statsDB, statsUser string
	if err := owner.QueryRow(t.Context(), "SELECT current_database()").Scan(&ownerDB); err != nil {
		t.Fatal(err)
	}
	if err := stats.QueryRow(t.Context(), "SELECT current_database(), current_user").Scan(&statsDB, &statsUser); err != nil {
		t.Fatal(err)
	}
	if statsUser != pgtest.StatsRole || ownerDB != statsDB {
		t.Errorf("stats pool as %q on %q, owner on %q; want %s on one database", statsUser, statsDB, ownerDB, pgtest.StatsRole)
	}
}

// An empty database has none of the template's tables: a test migrates it step by step.
func TestNewEmptyDatabaseURL_IsNotTheTemplate(t *testing.T) {
	t.Parallel()
	pool, err := postgres.Open(t.Context(), pgtest.NewEmptyDatabaseURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var probe *string
	if err := pool.QueryRow(t.Context(), "SELECT to_regclass('probe')::text").Scan(&probe); err != nil || probe != nil {
		t.Errorf("the empty database has probe (%v, %v)", probe, err)
	}
}

func TestAppURLAndRetentionURL_LogInToTheSameDatabaseAsEachRole(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)

	for role, roleURL := range map[string]string{
		pgtest.AppRole: pgtest.AppURL(t, dbURL), pgtest.RetentionRole: pgtest.RetentionURL(t, dbURL), pgtest.StatsRole: pgtest.StatsURL(t, dbURL),
	} {
		pool, err := postgres.Open(t.Context(), roleURL)
		if err != nil {
			t.Fatalf("log in as %s: %v", role, err)
		}
		var user string
		err = pool.QueryRow(t.Context(), "SELECT current_user FROM probe UNION ALL SELECT current_user LIMIT 1").Scan(&user)
		pool.Close()
		if err != nil || user != role {
			t.Errorf("logged in as %q (%v), want %s on the database with probe", user, err, role)
		}
	}
}

// TestNewRetentionDatabase_FailsClearlyWithoutPassword re-runs this test binary for the test above without
// TEST_RETENTION_DATABASE_PASSWORD: it must fail and say how to run the tests.
func TestNewRetentionDatabase_FailsClearlyWithoutPassword(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestNewRetentionDatabase_ConnectsToOneDatabaseAsOwnerAndAsTheRetentionRole$")
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "TEST_RETENTION_DATABASE_PASSWORD=")
	})

	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("test passed without TEST_RETENTION_DATABASE_PASSWORD; output:\n%s", out)
	}
	if !strings.Contains(string(out), "TEST_RETENTION_DATABASE_PASSWORD is not set") || !strings.Contains(string(out), "make test") {
		t.Errorf("output does not explain the missing TEST_RETENTION_DATABASE_PASSWORD:\n%s", out)
	}
}

// TestNewAppDatabase_FailsClearlyWithoutPassword re-runs this test binary for the test above without
// TEST_APP_DATABASE_PASSWORD: it must fail and say how to run the tests.
func TestNewAppDatabase_FailsClearlyWithoutPassword(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestNewAppDatabase_ConnectsToOneDatabaseAsOwnerAndAsTheAppRole$")
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "TEST_APP_DATABASE_PASSWORD=")
	})

	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("test passed without TEST_APP_DATABASE_PASSWORD; output:\n%s", out)
	}
	if !strings.Contains(string(out), "TEST_APP_DATABASE_PASSWORD is not set") || !strings.Contains(string(out), "make test") {
		t.Errorf("output does not explain the missing TEST_APP_DATABASE_PASSWORD:\n%s", out)
	}
}

// TestMain_FailsClearlyWithoutTestDatabaseURL runs this test binary again without TEST_DATABASE_URL: it must
// exit non-zero and say how to run the tests, never skip and never start a server of its own.
func TestMain_FailsClearlyWithoutTestDatabaseURL(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$") // re-runs this test binary
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "TEST_DATABASE_URL=")
	})

	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("test binary exited 0 without TEST_DATABASE_URL; output:\n%s", out)
	}
	if !strings.Contains(string(out), "TEST_DATABASE_URL is not set") || !strings.Contains(string(out), "make test") {
		t.Errorf("output does not explain the missing TEST_DATABASE_URL:\n%s", out)
	}
}

// WaitForLockWait returns once a session of the database waits on a lock: here an update of a row another
// transaction holds.
func TestWaitForLockWait_ReturnsOnceASessionWaitsOnALock(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	if _, err := db.Exec(t.Context(), "INSERT INTO probe VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(t.Context(), "UPDATE probe SET id = 2"); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() {
		_, err := db.Exec(context.Background(), "UPDATE probe SET id = 3")
		waited <- err
	}()

	pgtest.WaitForLockWait(t, db)

	select {
	case err := <-waited:
		t.Fatalf("the second update returned %v before the lock was released", err)
	default:
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := <-waited; err != nil {
		t.Errorf("the second update = %v", err)
	}
}

// A test's pools are small: up to one package per CPU runs at once, each with parallel tests, and a pool
// sized by the CPU count in each could pass the server's connection limit (seen on a 28-core host).
func TestNewAppDatabase_PoolsAreCappedAtPoolMaxConns(t *testing.T) {
	owner, app := pgtest.NewAppDatabase(t)
	for name, pool := range map[string]interface{ Stat() *pgxpool.Stat }{"owner": owner, "app": app} {
		if got := pool.Stat().MaxConns(); got != pgtest.PoolMaxConns {
			t.Errorf("%s pool MaxConns = %d, want %d", name, got, pgtest.PoolMaxConns)
		}
	}
}

// An advisory lock belongs to one database and roles to the cluster: a test that sets a role's password on its
// own database (cmd/migrate's) holds the role lock on the server's, so every package's Main, which sets the
// roles' passwords there, waits instead of failing with "tuple concurrently updated".
func TestHoldRoleLock_MakesASetPasswordOnTheServersDatabaseWait(t *testing.T) {
	serverDB, err := postgres.Open(t.Context(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(serverDB.Close)
	role := "pgtest_role_lock_" + strings.ToLower(rand.Text())
	if _, err := serverDB.Exec(t.Context(), "CREATE ROLE "+role+" LOGIN"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = serverDB.Exec(context.Background(), "DROP ROLE "+role) })

	release := pgtest.HoldRoleLock(t)
	done := make(chan error, 1)
	go func() { done <- postgres.SetPassword(context.Background(), serverDB, role, "synthetic") }()

	pgtest.WaitForLockWait(t, serverDB)
	select {
	case err := <-done:
		t.Fatalf("SetPassword returned %v while the role lock was held", err)
	default:
	}
	release()
	if err := <-done; err != nil {
		t.Errorf("SetPassword after the release = %v", err)
	}
}
