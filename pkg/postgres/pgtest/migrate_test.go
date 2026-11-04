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

func TestSetAppPassword_SkipsWithoutThePassword(t *testing.T) {
	t.Setenv("TEST_APP_DATABASE_PASSWORD", "")

	if err := setAppPassword(t.Context(), server.admin); err != nil {
		t.Fatalf("setAppPassword() error = %v, want nil when there is no password to set", err)
	}
}

func TestSetAppPassword_ReturnsTheQueryError(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), server.url)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()

	if err := setAppPassword(t.Context(), pool); err == nil {
		t.Fatal("setAppPassword() error = nil on a closed pool")
	}
}
