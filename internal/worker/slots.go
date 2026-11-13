package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// slotsWorker generates slots: of the job's schedule (enqueued by an availability edit), or of every listed
// schedule (the scheduler's hourly run). It logs the counts, nothing else.
type slotsWorker struct {
	river.WorkerDefaults[schedule.SlotsGenerateArgs]
	slots *slot.Service
	log   *slog.Logger
}

func (w *slotsWorker) Work(ctx context.Context, job *river.Job[schedule.SlotsGenerateArgs]) error {
	var g slot.Generated
	var err error
	if job.Args.ScheduleID == "" {
		g, err = w.slots.GenerateAll(ctx)
	} else {
		g, err = w.slots.Generate(ctx, job.Args.OrganizationID, job.Args.ScheduleID)
	}
	w.log.InfoContext(ctx, "slots generated", "schedules", g.Schedules, "inserted", g.Inserted, "deleted", g.Deleted)
	if errors.Is(err, schedule.ErrNotFound) && job.Args.ScheduleID != "" {
		return river.JobCancel(err) // schedules are never deleted: a retry would find nothing either
	}
	return err
}

// newSlots builds the slot service on pool (salus_app) at clk: generation reads the schedule and the clinic's
// time zone through their services, which need neither roles nor jobs for that.
func newSlots(pool *pgxpool.Pool, clk clock.Clock) *slot.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	schedules := schedule.NewService(schedule.Ports{Repo: schedule.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk, nil)
	return slot.NewService(slot.NewPostgresRepository(pool), tx, clk, schedules, clinics)
}
