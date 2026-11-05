package videosession_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The window-close job goes on the default queue in the caller's transaction, scheduled at the window's
// close, with the session and clinic ids only.
func TestRiverJobs_EnqueueTheWindowCloseInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := videosession.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	const sessionID = "5e000000-0000-4000-8000-000000000001"
	closes := end.Add(30 * time.Minute)
	enqueue := func(ctx context.Context) error { return jobs.EnqueueWindowClose(ctx, sessionID, clinicA, closes) }

	errRollBack := errors.New("the join failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := enqueue(ctx); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back join: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, enqueue); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	var kind, queue, args string
	var at time.Time
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text, scheduled_at FROM river_job").Scan(&kind, &queue, &args, &at); err != nil {
		t.Fatal(err)
	}
	want := `{"session_id": "` + sessionID + `", "organization_id": "` + clinicA + `"}`
	if kind != "consultation_window_close" || queue != river.QueueDefault || args != want || !at.Equal(closes) {
		t.Errorf("job %s on %s with %s at %v; want consultation_window_close on %s with %s at %v", kind, queue, args, at, river.QueueDefault, want, closes)
	}
	if err := enqueue(ctx); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueWindowClose outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
