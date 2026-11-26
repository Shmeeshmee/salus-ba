//go:build loadtest

package load_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The guard holds every pool the run opens to the load stack's server, not only the owner's: the app's and the
// statistics' URLs are given apart (LOAD_APP_DATABASE_URL, LOAD_STATS_DATABASE_URL), and one pointing elsewhere would
// take the seed's statistics writes or the driver's requests there (WAVE7 integration's review). The test stack's
// Postgres is not the load stack's, so each is refused there, by its name.
func TestGuard_RefusesAPoolOnAnyServerButTheLoadStacks(t *testing.T) {
	t.Parallel()
	pool := openPool(t, pgtest.NewDatabaseURL(t))
	if err := checkGuard(t.Context(), confirmValue, "dev", pool); !errors.Is(err, errRefused) || !strings.Contains(err.Error(), "cluster_name") {
		t.Errorf("checkGuard on the test stack = %v, want the cluster_name refusal", err)
	}
	err := guardPools(t.Context(), map[string]*pgxpool.Pool{"LOAD_APP_DATABASE_URL": pool})
	if !errors.Is(err, errRefused) || !strings.Contains(err.Error(), "LOAD_APP_DATABASE_URL") {
		t.Errorf("guardPools on the test stack = %v, want the refusal naming LOAD_APP_DATABASE_URL", err)
	}
}
