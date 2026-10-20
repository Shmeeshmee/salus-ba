package pgtest_test

import (
	"testing"
	"testing/fstest"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A one-file migration set proves the template is migrated before it is cloned.
func TestMain(m *testing.M) {
	pgtest.Main(m, fstest.MapFS{
		"00001_probe.sql": {Data: []byte("-- +goose Up\nCREATE TABLE probe (id int);\n")},
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
