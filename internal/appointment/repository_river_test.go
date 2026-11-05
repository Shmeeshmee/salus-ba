package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRiverJobs_EnqueueTheEmailsInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := appointment.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	remindAt := start.Add(-24 * time.Hour)
	enqueueAll := func(ctx context.Context) error {
		if err := jobs.EnqueueBooked(ctx, apptA, clinicA); err != nil {
			return err
		}
		if err := jobs.EnqueueCancelled(ctx, apptA, clinicA); err != nil {
			return err
		}
		return jobs.EnqueueReminder(ctx, apptA, clinicA, remindAt)
	}

	errRollBack := errors.New("the booking failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := enqueueAll(ctx); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back booking: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, enqueueAll); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	rows, err := owner.Query(ctx, "SELECT kind, queue, args::text, scheduled_at FROM river_job ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	args := `{"appointment_id": "` + apptA + `", "organization_id": "` + clinicA + `"}`
	want := []string{"appointment_booked_email", "appointment_cancelled_email", "appointment_reminder_email"}
	var i int
	for rows.Next() {
		var kind, queue, got string
		var at time.Time
		if err := rows.Scan(&kind, &queue, &got, &at); err != nil {
			t.Fatal(err)
		}
		if i >= len(want) || kind != want[i] || queue != "email" || got != args {
			t.Errorf("job %d: %s on %s with %s; want %v on email with %s", i, kind, queue, got, want, args)
		}
		if kind == "appointment_reminder_email" && !at.Equal(remindAt) {
			t.Errorf("the reminder is scheduled at %v, want %v", at, remindAt)
		}
		i++
	}
	if i != len(want) {
		t.Errorf("%d jobs, want %d", i, len(want))
	}

	for name, err := range map[string]error{
		"EnqueueBooked":    jobs.EnqueueBooked(ctx, apptA, clinicA),
		"EnqueueCancelled": jobs.EnqueueCancelled(ctx, apptA, clinicA),
		"EnqueueReminder":  jobs.EnqueueReminder(ctx, apptA, clinicA, remindAt),
	} {
		if !errors.Is(err, postgres.ErrNoTx) {
			t.Errorf("%s outside a transaction = %v, want postgres.ErrNoTx", name, err)
		}
	}
}
