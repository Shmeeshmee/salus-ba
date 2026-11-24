package datarequest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A deadline warning is queued in the transaction, on the notifier's e-mail queue, with the request's id only.
func TestRiverJobs_EnqueuesTheWarningInTheTransactionWithOnlyItsID(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := datarequest.NewRiverJobs(client)
	args := datarequest.WarningArgs{RequestID: profileA, Days: datarequest.WarnDays}
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error { return jobs.EnqueueWarning(ctx, args) }); err != nil {
		t.Fatalf("EnqueueWarning() = %v", err)
	}
	var kind, queue, got string
	if err := owner.QueryRow(t.Context(), "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &got); err != nil {
		t.Fatal(err)
	}
	if want := `{"days": 7, "request_id": "` + profileA + `"}`; kind != "data_request_deadline_warning" || queue != "email" || got != want {
		t.Errorf("job %s on %s with %s; want data_request_deadline_warning on email with %s", kind, queue, got, want)
	}
	if err := jobs.EnqueueWarning(t.Context(), args); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueWarning() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
