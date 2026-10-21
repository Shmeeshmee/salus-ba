// Package worker builds the River client the worker binary runs and registers its jobs.
//
// A job is enqueued with InsertTx in the same transaction as the write that causes it. Job arguments carry
// opaque ids only, never personal data; the job loads what it needs.
package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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

// New returns a River client on pool that works every registered job.
func New(pool *pgxpool.Pool, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &noopWorker{})
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:  log,
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: maxWorkers}},
		Workers: workers,
	})
}
