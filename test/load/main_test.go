//go:build loadtest

package load_test

import (
	"os"
	"testing"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// TestMain: the load stack's run (TestLoad, LOAD_DATABASE_URL set by docker-compose.load.yml) uses that database
// alone; the budget test (TestBudgets) gets pgtest's migrated template on the compose Postgres, as make test does.
func TestMain(m *testing.M) {
	if os.Getenv("LOAD_DATABASE_URL") != "" {
		os.Exit(m.Run())
	}
	pgtest.Main(m, migrations.FS)
}
