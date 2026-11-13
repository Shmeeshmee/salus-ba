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
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/communication"
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

// New returns the River client cmd/worker runs: it works the default queue and the scan queue on pool, the
// salus_app login. retentionPool, the salus_retention login (RETENTION_DATABASE_URL), goes to the
// retention_apply job's purgers and nowhere else; statsPool, the salus_stats login (STATS_DATABASE_URL), to the
// clinic statistics' builds; provider, the video server, to the consultation's jobs; files to the binary jobs
// and binary's purger; webhookKey (WEBHOOK_KEY) opens the secrets of the clinics' webhooks, which the webhook queue
// delivers.
func New(pool, retentionPool, statsPool *pgxpool.Pool, provider videosession.Provider, files Files, webhookKey []byte, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	if retentionPool == nil {
		return nil, errors.New("worker: the retention_apply job needs the retention role's pool")
	}
	if statsPool == nil {
		return nil, errors.New("worker: the clinic statistics' jobs need the stats role's pool")
	}
	if provider == nil {
		return nil, errors.New("worker: the consultation's jobs need the video provider")
	}
	if err := files.check(); err != nil {
		return nil, err
	}
	rules, err := newRetention(pool, retentionPool, files.Store)
	if err != nil {
		return nil, err
	}
	webhooks, err := newWebhooks(pool, webhookKey, clock.System{})
	if err != nil {
		return nil, err
	}
	inserter, err := NewInserter(pool) // the revocation enqueues its cancellation e-mails in its transaction
	if err != nil {
		return nil, err
	}
	responses := newQuestionnaireResponses(pool, clock.System{})
	workers := river.NewWorkers()
	addDefaultWorkers(workers, defaultServices{
		rules: rules, slots: newSlots(pool, clock.System{}), appointments: newRevocations(pool, inserter, clock.System{}),
		sessions: newSessions(pool, provider, inserter, clock.System{}), responses: responses,
		files: newFiles(pool, files, responses, clock.System{}), chat: newChatPartitions(pool, clock.System{}),
		clinicStats: newClinicStats(statsPool, clock.System{}),
	}, log)
	addWebhookWorkers(workers, webhooks)
	cfg := workConfig(log, river.QueueDefault, workers)
	cfg.Queues = workerQueues()
	return river.NewClient(riverpgxv5.New(pool), cfg)
}

// workerQueues are the queues cmd/worker works: the default one, the scan queue at scanWorkers, the clinics'
// webhooks, and the clinic statistics' builds at statsWorkers.
func workerQueues() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		river.QueueDefault: {MaxWorkers: maxWorkers}, binary.QueueScan: {MaxWorkers: scanWorkers}, webhookQueue: {MaxWorkers: maxWorkers},
		clinicstats.QueueStats: {MaxWorkers: statsWorkers},
	}
}

// defaultServices are what the default queue's jobs work with.
type defaultServices struct {
	rules        *retention.Service
	slots        *slot.Service
	appointments *appointment.Service
	sessions     *videosession.Service
	responses    *questionnaireresponse.Service
	files        *binary.Service
	chat         *communication.Service
	clinicStats  *clinicstats.Builder
}

// addDefaultWorkers registers the default queue's jobs. The scheduler registers them too, with no services:
// it never works them, but as the leader its rescuer must know every kind.
func addDefaultWorkers(workers *river.Workers, s defaultServices, log *slog.Logger) {
	river.AddWorker(workers, &noopWorker{})
	river.AddWorker(workers, &retentionWorker{rules: s.rules, log: log})
	river.AddWorker(workers, &slotsWorker{slots: s.slots, log: log})
	river.AddWorker(workers, &chatPartitionsWorker{chat: s.chat})
	river.AddWorker(workers, &revocationWorker{appointments: s.appointments, sessions: s.sessions, log: log})
	river.AddWorker(workers, &transcriptionRevocationWorker{sessions: s.sessions})
	river.AddWorker(workers, &windowCloseWorker{sessions: s.sessions, log: log})
	river.AddWorker(workers, &accessEndWorker{sessions: s.sessions, log: log})
	river.AddWorker(workers, &questionnaireRequestWorker{responses: s.responses, log: log})
	river.AddWorker(workers, &questionnaireDiscardWorker{responses: s.responses, log: log})
	river.AddWorker(workers, &binaryScanWorker{files: s.files, clock: clock.System{}, log: log})
	river.AddWorker(workers, &binaryDiscardWorker{files: s.files, log: log})
	river.AddWorker(workers, &clinicStatsRefreshWorker{builds: s.clinicStats})
	river.AddWorker(workers, &clinicStatsNightlyWorker{builds: s.clinicStats})
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
