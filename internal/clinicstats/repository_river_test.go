package clinicstats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The refresh is enqueued on the default queue in the caller's transaction (rolled back with it), with the clinic
// and the instant only; outside a transaction it is refused.
func TestRiverQueue_EnqueuesTheRefreshInTheTransaction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	queue := clinicstats.NewRiverQueue(client)
	tr := postgres.NewTransactor(app)
	at := time.Date(2026, 11, 2, 23, 30, 0, 0, time.UTC)
	rollBack := errors.New("the signing failed")
	err = tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := queue.EnqueueStatsRefresh(ctx, clinicA, at); err != nil {
			return err
		}
		return rollBack
	})
	var n int
	if qerr := owner.QueryRow(t.Context(), "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, rollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rollback: %d jobs, %v, %v; want none", n, err, qerr)
	}
	if err := tr.WithTx(t.Context(), func(ctx context.Context) error { return queue.EnqueueStatsRefresh(ctx, clinicA, at) }); err != nil {
		t.Fatal(err)
	}
	var kind, q string
	var args clinicstats.RefreshArgs
	if err := owner.QueryRow(t.Context(), "SELECT kind, queue, args FROM river_job").Scan(&kind, &q, &args); err != nil {
		t.Fatal(err)
	}
	if kind != "clinicstats_refresh" || q != clinicstats.QueueStats || args.OrganizationID != clinicA || !args.Day.Equal(at) {
		t.Errorf("enqueued %s on %s with %+v", kind, q, args)
	}
	if o := (clinicstats.NightlyArgs{}).InsertOpts(); o.Queue != clinicstats.QueueStats {
		t.Errorf("the nightly pass goes to %q, want %q", o.Queue, clinicstats.QueueStats)
	}
	if err := queue.EnqueueStatsRefresh(t.Context(), clinicA, at); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("outside a transaction: %v, want ErrNoTx", err)
	}
}
