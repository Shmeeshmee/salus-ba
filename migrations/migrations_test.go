package migrations_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

var fileName = regexp.MustCompile(`^\d{5}_[a-z0-9_]+\.sql$`)

func TestFiles_AreForwardOnlyNumberedGooseFiles(t *testing.T) {
	t.Parallel()
	names, err := fs.Glob(migrations.FS, "*")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no migrations embedded")
	}
	for _, name := range names {
		if !fileName.MatchString(name) {
			t.Errorf("%s: want NNNNN_short_name.sql", name)
			continue
		}
		body, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(body), "-- +goose Up\n") {
			t.Errorf("%s: first line must be -- +goose Up", name)
		}
		if strings.Contains(string(body), "+goose Down") {
			t.Errorf("%s: migrations are forward-only; remove the Down section", name)
		}
	}
}

func TestUp_CreatesExtensions(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	for _, ext := range []string{"btree_gist", "pgcrypto"} {
		var installed bool
		err := db.QueryRow(t.Context(), "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = $1)", ext).Scan(&installed)
		if err != nil {
			t.Fatal(err)
		}
		if !installed {
			t.Errorf("extension %s is not installed", ext)
		}
	}
}

func TestUp_CreatesRiverSchema(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	// River 007 drops river_client and river_client_queue again; these are the tables it ends with.
	want := map[string]bool{
		"river_job": true, "river_leader": true, "river_queue": true, "river_notification": true,
		"river_client": false, "river_client_queue": false,
	}
	for table, wantExists := range want {
		var exists bool
		if err := db.QueryRow(t.Context(), "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != wantExists {
			t.Errorf("table %s exists = %v, want %v", table, exists, wantExists)
		}
	}
}
