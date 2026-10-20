package pgtest_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m) }

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
