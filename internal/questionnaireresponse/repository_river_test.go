package questionnaireresponse_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The binary_discard job goes in with the transaction that no longer lists the files, with the response's and
// the clinic's ids only: a rollback leaves none, and outside a transaction it is refused.
func TestRiverJobs_EnqueuesTheDiscardInTheTransactionWithOnlyIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := questionnaireresponse.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	args := binary.DiscardArgs{ResponseID: fileA, OrganizationID: clinicA}

	errRollBack := errors.New("the submit failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueDiscard(ctx, args); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back submit: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueDiscard(ctx, args) }); err != nil {
		t.Fatalf("EnqueueDiscard() = %v", err)
	}
	var kind, got string
	if err := owner.QueryRow(ctx, "SELECT kind, args::text FROM river_job").Scan(&kind, &got); err != nil {
		t.Fatal(err)
	}
	want := `{"organization_id": "` + clinicA + `", "questionnaire_response_id": "` + fileA + `"}`
	if kind != "binary_discard" || got != want {
		t.Errorf("job %s with %s;\nwant binary_discard with %s", kind, got, want)
	}
	if err := jobs.EnqueueDiscard(ctx, args); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueDiscard() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
