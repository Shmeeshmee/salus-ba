package binary_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRiverJobs_EnqueuesTheScanOnItsQueueInTheTransactionWithOnlyIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := binary.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	args := binary.ScanArgs{BinaryID: unknownID, ResponseID: responseA, OrganizationID: clinicA}

	errRollBack := errors.New("the upload failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueScan(ctx, args); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back upload: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueScan(ctx, args) }); err != nil {
		t.Fatalf("EnqueueScan() = %v", err)
	}
	var kind, queue, got string
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text FROM river_job").Scan(&kind, &queue, &got); err != nil {
		t.Fatal(err)
	}
	want := `{"binary_id": "` + unknownID + `", "organization_id": "` + clinicA + `", "questionnaire_response_id": "` + responseA + `"}`
	if kind != "binary_scan" || queue != binary.QueueScan || got != want {
		t.Errorf("job %s on %s with %s;\nwant binary_scan on %s with %s", kind, queue, got, binary.QueueScan, want)
	}
	if err := jobs.EnqueueScan(ctx, args); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("EnqueueScan() outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}

// A chat file's scan names its conversation instead of a response, and gives up after ChatScanAttempts, so a file
// that cannot be scanned is rejected within the hour (Unscannable) rather than after River's default 25 attempts.
func TestRiverJobs_EnqueuesAChatFilesScanWithItsConversationAndFewerAttempts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := binary.NewRiverJobs(client)
	args := binary.ScanArgs{BinaryID: unknownID, ConversationID: convC, OrganizationID: clinicA}
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error { return jobs.EnqueueScan(ctx, args) }); err != nil {
		t.Fatalf("EnqueueScan() = %v", err)
	}
	var got string
	var attempts int
	if err := owner.QueryRow(t.Context(), "SELECT args::text, max_attempts FROM river_job").Scan(&got, &attempts); err != nil {
		t.Fatal(err)
	}
	want := `{"binary_id": "` + unknownID + `", "conversation_id": "` + convC + `", "organization_id": "` + clinicA + `"}`
	if got != want || attempts != binary.ChatScanAttempts {
		t.Errorf("job with %s, %d attempts; want %s, %d", got, attempts, want, binary.ChatScanAttempts)
	}
}
