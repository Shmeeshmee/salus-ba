package consent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRiverJobs_EnqueuesConsentRevokedInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := consent.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()

	errRollBack := errors.New("the revocation failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueConsentRevoked(ctx, patientA, clinicA); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back revocation: %d jobs (%v, %v), want none", n, err, qerr)
	}

	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueConsentRevoked(ctx, patientA, clinicA) }); err != nil {
		t.Fatalf("EnqueueConsentRevoked() = %v", err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"patient_id": "` + patientA + `", "organization_id": "` + clinicA + `"}`; kind != "consent_revoked" || queue != river.QueueDefault || args != want {
		t.Errorf("job %s on %s with %s;\nwant consent_revoked on %s with %s", kind, queue, args, river.QueueDefault, want)
	}

	if err := jobs.EnqueueConsentRevoked(ctx, patientA, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueConsentRevoked() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}

func TestRiverJobs_EnqueuesTranscriptionConsentRevokedInTheTransactionWithOnlyTheTwoIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := consent.NewRiverJobs(client)
	ctx := t.Context()
	if err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
		return jobs.EnqueueTranscriptionRevoked(ctx, patientA, clinicA)
	}); err != nil {
		t.Fatalf("EnqueueTranscriptionRevoked() = %v", err)
	}
	var kind, queue, args string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"patient_id": "` + patientA + `", "organization_id": "` + clinicA + `"}`; kind != "transcription_consent_revoked" || queue != river.QueueDefault || args != want {
		t.Errorf("job %s on %s with %s;\nwant transcription_consent_revoked on %s with %s", kind, queue, args, river.QueueDefault, want)
	}
	if err := jobs.EnqueueTranscriptionRevoked(ctx, patientA, clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueTranscriptionRevoked() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
