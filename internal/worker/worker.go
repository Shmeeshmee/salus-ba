// Package worker builds the River clients the worker, notifier and scheduler binaries run, and registers
// their jobs.
//
// A job is enqueued with InsertTx in the same transaction as the write that causes it. Job arguments carry
// opaque ids only, never personal data; the job loads what it needs.
//
// River elects one leader per database among the clients that work queues; the leader runs the periodic jobs
// and River's maintenance (scheduled jobs, retries, rescue, cleanup). The periodic jobs are configured only in
// the scheduler, so the worker and the notifier never lead (workConfig) and the scheduler always does.
package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// maxWorkers bounds the jobs one worker process runs at once.
const maxWorkers = 10

// NoopArgs is a job that does nothing. It proves the queue end to end until real jobs exist.
type NoopArgs struct{}

// Kind names the job in the river_job table.
func (NoopArgs) Kind() string { return "noop" }

type noopWorker struct {
	river.WorkerDefaults[NoopArgs]
}

func (noopWorker) Work(context.Context, *river.Job[NoopArgs]) error { return nil }

// New returns the River client cmd/worker runs: it works the default queue on pool, the salus_app login.
// retentionPool, the salus_retention login (RETENTION_DATABASE_URL), goes to the retention_apply job's
// purgers and nowhere else.
func New(pool, retentionPool *pgxpool.Pool, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	if retentionPool == nil {
		return nil, errors.New("worker: the retention_apply job needs the retention role's pool")
	}
	rules, err := newRetention(pool, retentionPool)
	if err != nil {
		return nil, err
	}
	workers := river.NewWorkers()
	addDefaultWorkers(workers, rules, newSlots(pool, clock.System{}), log)
	return river.NewClient(riverpgxv5.New(pool), workConfig(log, river.QueueDefault, workers))
}

// addDefaultWorkers registers the default queue's jobs. The scheduler registers them too, with nil
// dependencies: it never works them, but as the leader its rescuer must know every kind.
func addDefaultWorkers(workers *river.Workers, rules *retention.Service, slots *slot.Service, log *slog.Logger) {
	river.AddWorker(workers, &noopWorker{})
	river.AddWorker(workers, &retentionWorker{rules: rules, log: log})
	river.AddWorker(workers, &slotsWorker{slots: slots, log: log})
}

// workConfig is the configuration of a client that works queue with workers and never leads.
func workConfig(log *slog.Logger, queue string, workers *river.Workers) *river.Config {
	return &river.Config{
		Logger:                 log,
		Queues:                 map[string]river.QueueConfig{queue: {MaxWorkers: maxWorkers}},
		LeaderElectionDisabled: true,
		// No reindexing: REINDEX needs the table's owner, and the clients log in as salus_app, which owns
		// nothing. Index maintenance on River's tables is an owner-side ops task.
		ReindexerIndexNames: []string{},
		Workers:             workers,
	}
}
