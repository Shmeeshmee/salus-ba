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
	enqueueAll := func(ctx context.Context) error {
		if err := jobs.EnqueueBooked(ctx, apptA, clinicA); err != nil {
			return err
		}
		return jobs.EnqueueCancelled(ctx, apptA, clinicA)
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
	want := []string{"appointment_booked_email", "appointment_cancelled_email"}
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
		i++
	}
	if i != len(want) {
		t.Errorf("%d jobs, want %d", i, len(want))
	}

	for name, err := range map[string]error{
		"EnqueueBooked":    jobs.EnqueueBooked(ctx, apptA, clinicA),
		"EnqueueCancelled": jobs.EnqueueCancelled(ctx, apptA, clinicA),
		"EnqueueReminders": jobs.EnqueueReminders(ctx, []appointment.Reminder{{ID: apptA, OrganizationID: clinicA, DueAt: start}}),
	} {
		if !errors.Is(err, postgres.ErrNoTx) {
			t.Errorf("%s outside a transaction = %v, want postgres.ErrNoTx", name, err)
		}
	}
}

func TestRiverJobs_EnqueueTheConsultationsAccessEndOnTheDefaultQueueInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := appointment.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()

	errRollBack := errors.New("the cancellation failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueAccessEnd(ctx, apptA, clinicA); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back cancellation: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueAccessEnd(ctx, apptA, clinicA) }); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	want := `{"appointment_id": "` + apptA + `", "organization_id": "` + clinicA + `"}`
	if kind != "consultation_access_end" || queue != river.QueueDefault || args != want {
		t.Errorf("job %s on %s with %s; want consultation_access_end on %s with %s", kind, queue, args, river.QueueDefault, want)
	}

	if err := jobs.EnqueueAccessEnd(ctx, apptA, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueAccessEnd outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}

// The questionnaires' jobs (docs/questionnaire/SPEC.md §2 "Requests", "Cancellation and revocation"): the
// default's request with the booking and the discard with a cancellation, on the default queue the worker
// works, in the caller's transaction, with the two ids only.
func TestRiverJobs_EnqueueTheQuestionnaireJobsOnTheDefaultQueueInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := appointment.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	enqueueBoth := func(ctx context.Context) error {
		if err := jobs.EnqueueQuestionnaireRequest(ctx, apptA, clinicA); err != nil {
			return err
		}
		return jobs.EnqueueQuestionnaireDiscard(ctx, apptA, clinicA)
	}

	errRollBack := errors.New("the booking failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := enqueueBoth(ctx); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back transaction: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, enqueueBoth); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	rows, err := owner.Query(ctx, "SELECT kind, queue, args::text FROM river_job ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	args := `{"appointment_id": "` + apptA + `", "organization_id": "` + clinicA + `"}`
	want := []string{"questionnaire_default_request", "questionnaire_responses_discard"}
	var i int
	for rows.Next() {
		var kind, queue, got string
		if err := rows.Scan(&kind, &queue, &got); err != nil {
			t.Fatal(err)
		}
		if i >= len(want) || kind != want[i] || queue != river.QueueDefault || got != args {
			t.Errorf("job %d: %s on %s with %s; want %v on %s with %s", i, kind, queue, got, want, river.QueueDefault, args)
		}
		i++
	}
	if i != len(want) {
		t.Errorf("%d jobs, want %d", i, len(want))
	}

	for name, err := range map[string]error{
		"EnqueueQuestionnaireRequest": jobs.EnqueueQuestionnaireRequest(ctx, apptA, clinicA),
		"EnqueueQuestionnaireDiscard": jobs.EnqueueQuestionnaireDiscard(ctx, apptA, clinicA),
	} {
		if !errors.Is(err, postgres.ErrNoTx) {
			t.Errorf("%s outside a transaction = %v, want postgres.ErrNoTx", name, err)
		}
	}
}

// A reminder's dispatch is enqueued on the notifier's push queue at its due time with the two ids only, once per
// reminder: enqueued again (the sweep), the job already there is kept.
func TestRiverJobs_EnqueueEachRemindersDispatchOnceAtItsDueTime(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := appointment.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	rs := []appointment.Reminder{
		{ID: apptA, OrganizationID: clinicA, DueAt: start.Add(-24 * time.Hour)},
		{ID: apptB, OrganizationID: clinicA, DueAt: start.Add(-time.Hour)},
	}
	for range 2 {
		if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueReminders(ctx, rs) }); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := owner.Query(ctx, "SELECT kind, queue, args::text, scheduled_at FROM river_job ORDER BY scheduled_at")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var i int
	for rows.Next() {
		var kind, queue, args string
		var at time.Time
		if err := rows.Scan(&kind, &queue, &args, &at); err != nil {
			t.Fatal(err)
		}
		want := `{"reminder_id": "` + rs[i].ID + `", "organization_id": "` + clinicA + `"}`
		if kind != "appointment_reminder" || queue != "push" || args != want || !at.Equal(rs[i].DueAt) {
			t.Errorf("job %d: %s on %s with %s at %v; want appointment_reminder on push with %s at %v", i, kind, queue, args, at, want, rs[i].DueAt)
		}
		i++
	}
	if i != len(rs) {
		t.Errorf("%d jobs, want %d (one per reminder)", i, len(rs))
	}
}
