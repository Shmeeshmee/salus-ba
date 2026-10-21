package pgtest

import (
	"testing"
	"testing/fstest"
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
		// The clone already has 00001 from TestMain; 00002 is the one that runs.
		{name: "bad SQL", url: "", migrations: fstest.MapFS{
			"00002_bad.sql": {Data: []byte("-- +goose Up\nCREATE TABLE;\n")},
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
