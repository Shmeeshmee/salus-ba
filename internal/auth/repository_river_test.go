package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const inviteID = "1a000000-0000-4000-8000-000000000001"

func TestRiverJobs_EnqueuesTheInviteEmailInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := auth.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()

	errRollBack := errors.New("the invite failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueInviteEmail(ctx, inviteID, clinicID); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back invite: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueInviteEmail(ctx, inviteID, clinicID) }); err != nil {
		t.Fatalf("EnqueueInviteEmail() = %v", err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"invite_id": "` + inviteID + `", "organization_id": "` + clinicID + `"}`; kind != "invite_email" || queue != "email" || args != want {
		t.Errorf("job %s on %s with %s;\nwant invite_email on email with %s", kind, queue, args, want)
	}

	if err := jobs.EnqueueInviteEmail(ctx, inviteID, clinicID); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueInviteEmail() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
