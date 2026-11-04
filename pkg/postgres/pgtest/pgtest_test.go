package pgtest_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A small migration set proves the template is migrated before it is cloned; the second file creates the
// application role the way migrations/ does (roles are cluster-wide, so a parallel package may create it
// first) and lets it read probe.
func TestMain(m *testing.M) {
	pgtest.Main(m, fstest.MapFS{
		"00001_probe.sql": {Data: []byte("-- +goose Up\nCREATE TABLE probe (id int);\n")},
		"00002_app_role.sql": {Data: []byte(`-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN CREATE ROLE salus_app LOGIN; EXCEPTION WHEN duplicate_object OR unique_violation THEN NULL; END $$;
-- +goose StatementEnd
GRANT SELECT ON probe TO salus_app;
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
