package composition_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The signature's e-mail is enqueued on the notifier's queue in the caller's transaction, with the two ids only.
func TestRiverJobs_EnqueueTheSignedEmailInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := composition.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	errRollBack := errors.New("the signature failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueSigned(ctx, apptA, clinicA); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back signature: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueSigned(ctx, apptA, clinicA) }); err != nil {
		t.Fatal(err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	want := `{"appointment_id": "` + apptA + `", "organization_id": "` + clinicA + `"}`
	if kind != "document_signed_email" || queue != "email" || args != want {
		t.Errorf("job = %s on %s with %s; want document_signed_email on email with %s", kind, queue, args, want)
	}
	if err := jobs.EnqueueSigned(ctx, apptA, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueSigned() outside a transaction = %v, want ErrNoTx", err)
	}
}
