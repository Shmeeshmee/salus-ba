package worker_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// webhookKey is a synthetic WEBHOOK_KEY.
var webhookKey = bytes.Repeat([]byte{9}, 32)

// The worker opens the clinics' webhook secrets with WEBHOOK_KEY, which must be an AES-256 key.
func TestNew_RefusesAWebhookKeyThatIsNot32Bytes(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	for _, key := range [][]byte{nil, bytes.Repeat([]byte{9}, 16)} {
		if _, err := worker.New(db.app, db.retention, db.stats, liveKit(t), testFiles(t), key, slog.New(slog.DiscardHandler)); err == nil || !strings.Contains(err.Error(), "WEBHOOK_KEY") {
			t.Errorf("New(a %d-byte key) = %v, want WEBHOOK_KEY refused", len(key), err)
		}
	}
}

// TestNoop_RunsAJobEnqueuedInATransaction enqueues the job the way every write will: inside the write's
// transaction (InsertTx). The worker runs it once the transaction commits.
func TestNoop_RunsAJobEnqueuedInATransaction(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	runNoop(t, owner, retention, owner)
}

// The worker and notifier log in as the application role; its grants must cover what River does. The
// retention role has no grant on River's tables, so the job also proves River runs on the app pool.
func TestNoop_RunsAsTheAppRole(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	runNoop(t, db.app, db.retention, db.stats)
}

func runNoop(t *testing.T, pool, retention, stats *pgxpool.Pool) {
	t.Helper()
	client, err := worker.New(pool, retention, stats, liveKit(t), testFiles(t), webhookKey, slog.New(slog.DiscardHandler))
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
