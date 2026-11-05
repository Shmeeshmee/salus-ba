package schedule_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRiverJobs_EnqueuesSlotsGenerateInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := schedule.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()

	errRollBack := errors.New("the edit failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueSlotsGenerate(ctx, scheduleA2, clinicA); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back edit: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueSlotsGenerate(ctx, scheduleA2, clinicA) }); err != nil {
		t.Fatalf("EnqueueSlotsGenerate() = %v", err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"schedule_id": "` + scheduleA2 + `", "organization_id": "` + clinicA + `"}`; kind != "slots_generate" || queue != river.QueueDefault || args != want {
		t.Errorf("job %s on %s with %s;\nwant slots_generate on %s with %s", kind, queue, args, river.QueueDefault, want)
	}

	if err := jobs.EnqueueSlotsGenerate(ctx, scheduleA2, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueSlotsGenerate() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
