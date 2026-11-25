//go:build loadtest

package load_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// wall is the clock the load reads its readings and today from: the real one, as the api and the workers it drives.
var wall clock.Clock = clock.System{}

// The busy year is written only to the load stack's own database (scripts/load.sh, docker-compose.load.yml), never
// to a development database with other data and never to production. guard refuses unless all of these hold:
//
//   - LOAD_CONFIRM is confirmValue: nobody seeds by accident;
//   - SALUS_ENV is dev, as only the local stacks set it;
//   - the server's cluster_name is loadCluster, which only docker-compose.load.yml sets;
//   - the database holds no clinic but the seed's own (every organization's name starts with seedPrefix).
const (
	confirmValue = "synthetic-busy-year"
	loadCluster  = "salus-load"
)

// guard checks the rules above on owner, the owner's pool of the database the seed and the driver use.
func guard(ctx context.Context, owner *pgxpool.Pool) error {
	return checkGuard(ctx, os.Getenv("LOAD_CONFIRM"), os.Getenv("SALUS_ENV"), owner)
}

// errRefused is every refusal of guard.
var errRefused = errors.New("load: refused")

func checkGuard(ctx context.Context, confirm, env string, owner *pgxpool.Pool) error {
	if confirm != confirmValue {
		return fmt.Errorf("%w: LOAD_CONFIRM is not %q", errRefused, confirmValue)
	}
	if env != "dev" {
		return fmt.Errorf("%w: SALUS_ENV is not dev", errRefused)
	}
	var cluster string
	if err := owner.QueryRow(ctx, `SELECT current_setting('cluster_name')`).Scan(&cluster); err != nil {
		return fmt.Errorf("load: cluster_name: %w", err)
	}
	if cluster != loadCluster {
		return fmt.Errorf("%w: the server's cluster_name is not %q (only the load stack's Postgres has it)", errRefused, loadCluster)
	}
	var others int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM organization WHERE name NOT LIKE $1 || '%'`, seedPrefix).Scan(&others); err != nil {
		return fmt.Errorf("load: organizations: %w", err)
	}
	if others > 0 {
		return fmt.Errorf("%w: the database holds %d clinics that are not the seed's", errRefused, others)
	}
	return nil
}
