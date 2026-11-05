package pgtest

import (
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrate_ReturnsTheError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		migrations fstest.MapFS
		name, url  string
	}{
		{name: "bad URL", url: "postgres://[::1", migrations: fstest.MapFS{
			"00001_probe.sql": {Data: []byte("-- +goose Up\nCREATE TABLE probe (id int);\n")},
		}},
		{name: "no migrations", url: server.url, migrations: fstest.MapFS{}},
		// The clone already has 00001 and 00002 from TestMain; 00003 is the one that runs.
		{name: "bad SQL", url: "", migrations: fstest.MapFS{
			"00003_bad.sql": {Data: []byte("-- +goose Up\nCREATE TABLE;\n")},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := tt.url
			if url == "" {
				url = NewDatabaseURL(t)
			}

			if err := migrate(t.Context(), url, tt.migrations); err == nil {
				t.Error("migrate returned nil, want an error")
			}
		})
	}
}

func TestWithDatabase_PanicsOnAnUnparsableURL(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Error("withDatabase did not panic")
		}
	}()

	withDatabase("postgres://[::1", "db")
}

func TestSetRolePassword_SkipsWithoutThePassword(t *testing.T) {
	if err := setRolePassword(t.Context(), server.admin, AppRole, ""); err != nil {
		t.Fatalf("setRolePassword() error = %v, want nil when there is no password to set", err)
	}
}

func TestSetRolePassword_SkipsARoleTheMigrationsDidNotCreate(t *testing.T) {
	if err := setRolePassword(t.Context(), server.admin, "salus_no_such_role", "synthetic"); err != nil {
		t.Fatalf("setRolePassword() error = %v, want nil for a role that does not exist", err)
	}
}

func TestSetRolePassword_ReturnsTheQueryError(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), server.url)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()

	if err := setRolePassword(t.Context(), pool, AppRole, "synthetic"); err == nil {
		t.Fatal("setRolePassword() error = nil on a closed pool")
	}
}
