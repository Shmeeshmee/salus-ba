package worker_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// TestNoop_RunsAJobEnqueuedInATransaction enqueues the job the way every write will: inside the write's
// transaction (InsertTx). The worker runs it once the transaction commits.
func TestNoop_RunsAJobEnqueuedInATransaction(t *testing.T) {
	t.Parallel()
	pool := pgtest.NewDatabase(t)
	client, err := worker.New(pool, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	completed, cancel := client.Subscribe(river.EventKindJobCompleted)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = client.Stop(t.Context()) }()

	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.InsertTx(t.Context(), tx, worker.NoopArgs{}, nil); err != nil {
		t.Fatalf("InsertTx() error = %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-completed:
		if ev.Job.Kind != "noop" {
			t.Errorf("completed job kind = %q, want noop", ev.Job.Kind)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the noop job did not complete within 10s")
	}
}
