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

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/videosession"
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
// purgers and nowhere else; provider, the video server, to the consultation's jobs.
func New(pool, retentionPool *pgxpool.Pool, provider videosession.Provider, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	if retentionPool == nil {
		return nil, errors.New("worker: the retention_apply job needs the retention role's pool")
	}
	if provider == nil {
		return nil, errors.New("worker: the consultation's jobs need the video provider")
	}
	rules, err := newRetention(pool, retentionPool)
	if err != nil {
		return nil, err
	}
	inserter, err := NewInserter(pool) // the revocation enqueues its cancellation e-mails in its transaction
	if err != nil {
		return nil, err
	}
	workers := river.NewWorkers()
	addDefaultWorkers(workers, defaultServices{
		rules: rules, slots: newSlots(pool, clock.System{}), appointments: newRevocations(pool, inserter, clock.System{}),
		sessions: newSessions(pool, provider, clock.System{}), responses: newQuestionnaireResponses(pool, clock.System{}),
	}, log)
	return river.NewClient(riverpgxv5.New(pool), workConfig(log, river.QueueDefault, workers))
}

// defaultServices are what the default queue's jobs work with.
type defaultServices struct {
	rules        *retention.Service
	slots        *slot.Service
	appointments *appointment.Service
	sessions     *videosession.Service
	responses    *questionnaireresponse.Service
}

// addDefaultWorkers registers the default queue's jobs. The scheduler registers them too, with no services:
// it never works them, but as the leader its rescuer must know every kind.
func addDefaultWorkers(workers *river.Workers, s defaultServices, log *slog.Logger) {
	river.AddWorker(workers, &noopWorker{})
	river.AddWorker(workers, &retentionWorker{rules: s.rules, log: log})
	river.AddWorker(workers, &slotsWorker{slots: s.slots, log: log})
	river.AddWorker(workers, &revocationWorker{appointments: s.appointments, sessions: s.sessions, log: log})
	river.AddWorker(workers, &windowCloseWorker{sessions: s.sessions, log: log})
	river.AddWorker(workers, &accessEndWorker{sessions: s.sessions, log: log})
	river.AddWorker(workers, &questionnaireRequestWorker{responses: s.responses, log: log})
	river.AddWorker(workers, &questionnaireDiscardWorker{responses: s.responses, log: log})
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
