package worker

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// retentionHour is the UTC hour of the daily retention_apply job (docs/booking/SPEC.md §2 "Periodic jobs").
const retentionHour = 3

// slotsInterval is how often slots_generate runs (docs/booking/SPEC.md §2 "Periodic jobs").
const slotsInterval = time.Hour

// schedulerQueue is the scheduler's own queue. Nothing is enqueued on it; River starts, and so elects and
// enqueues periodic jobs in, only a client that works a queue.
const schedulerQueue = "scheduler"

// NewScheduler returns the River client cmd/scheduler runs on pool (salus_app): it enqueues the periodic jobs
// and, as River's only leader, runs its maintenance. It works no job. Several replicas elect one leader, so
// each periodic job is enqueued once.
func NewScheduler(pool *pgxpool.Pool, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	return newScheduler(pool, log, dailyAt{hour: retentionHour})
}

func newScheduler(pool *pgxpool.Pool, log *slog.Logger, retentionSchedule river.PeriodicSchedule) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	addDefaultWorkers(workers, defaultServices{}, nil)
	addEmailWorkers(workers, nil)
	addChatWorkers(workers, nil)
	addWebhookWorkers(workers, nil)
	addAgentWorkers(workers, nil, nil)
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: log,
		Queues: map[string]river.QueueConfig{schedulerQueue: {MaxWorkers: 1}},
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(retentionSchedule, func() (river.JobArgs, *river.InsertOpts) {
				return RetentionApplyArgs{}, nil
			}, nil),
			// Also when the scheduler starts leading, so slots do not wait an hour after a deploy; generation
			// is idempotent.
			river.NewPeriodicJob(river.PeriodicInterval(slotsInterval), func() (river.JobArgs, *river.InsertOpts) {
				return schedule.SlotsGenerateArgs{}, nil
			}, &river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(clinicStatsAt, func() (river.JobArgs, *river.InsertOpts) {
				return clinicstats.NightlyArgs{}, nil
			}, nil),
			// Also when it starts leading: a deploy near a month's end must not wait for the next day.
			river.NewPeriodicJob(dailyAt{hour: chatPartitionsHour}, func() (river.JobArgs, *river.InsertOpts) {
				return communication.PartitionsArgs{}, nil
			}, &river.PeriodicJobOpts{RunOnStart: true}),
		},
		ReindexerIndexNames: []string{}, // as in workConfig: salus_app owns nothing
		Workers:             workers,
	})
}

// dailyAt is a River schedule that fires every day at hour:minute UTC.
type dailyAt struct{ hour, minute int }

// Next returns the first hour:minute UTC strictly after t.
func (d dailyAt) Next(t time.Time) time.Time {
	t = t.UTC()
	next := time.Date(t.Year(), t.Month(), t.Day(), d.hour, d.minute, 0, 0, time.UTC)
	if !next.After(t) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
