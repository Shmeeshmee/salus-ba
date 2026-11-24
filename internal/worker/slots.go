package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// slotsWorker generates slots: of the job's schedule (enqueued by an availability edit), or of every schedule of an
// active role of every clinic, listed or not (the scheduler's hourly run). It logs the counts, nothing else.
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

// newSlots builds the slot service on pool (salus_app) at clk: generation reads the schedule, its role and the
// clinic's time zone through their services, each in the clinic's scope. The hourly run lists every clinic's id on
// statsPool (salus_stats, which reads only organization's id and time zone): salus_app sees no clinic outside a scope.
func newSlots(pool, statsPool *pgxpool.Pool, clk clock.Clock) *slot.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	roles := practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	ports := schedule.Ports{Repo: schedule.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Roles: roles}
	if statsPool != nil { // nil for a service that only holds and frees slots
		ports.Clinics = organization.NewStats(organization.NewPostgresStatsRepository(statsPool))
	}
	schedules := schedule.NewService(ports)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk, nil)
	return slot.NewService(slot.NewPostgresRepository(pool), tx, clk, schedules, clinics)
}
