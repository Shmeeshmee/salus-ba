package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/trends"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// Trends are the platform builds' settings (docs/plans/analytics.md §2a): Stats is the salus_stats pool
// (STATS_DATABASE_URL), nil while ANALYTICS_PLATFORM_ENABLED is off, and MinCount is k.
type Trends struct {
	Stats    *pgxpool.Pool
	MinCount int
}

// TrendsBuildArgs builds one closed period of the platform figures (a month, a quarter or an ISO week) and freezes
// it. The scheduler enqueues it only while the platform view is on; its argument is the period's name, nothing else.
type TrendsBuildArgs struct {
	Period string `json:"period"`
}

// Kind names the job in the river_job table.
func (TrendsBuildArgs) Kind() string { return "trends_build" }

// trendsAgent names the build in its AuditEvent.
const trendsAgent = "trends_build"

// trendsWorker runs a build and logs its counts only (A13). A period built already is done.
type trendsWorker struct {
	river.WorkerDefaults[TrendsBuildArgs]
	builder *trends.Builder
	log     *slog.Logger
}

func (w *trendsWorker) Work(ctx context.Context, job *river.Job[TrendsBuildArgs]) error {
	if w.builder == nil {
		return errors.New("worker: the platform builds are off")
	}
	s, err := w.builder.Build(ctx, job.Args.Period)
	if errors.Is(err, trends.ErrFrozen) {
		w.log.InfoContext(ctx, "trends period already built", "period", job.Args.Period)
		return nil
	}
	if err != nil {
		return err
	}
	w.log.InfoContext(ctx, "trends built", "period", job.Args.Period, "cells", s.Cells, "suppressed", s.Suppressed,
		"rises", s.Rises, "refused", s.Refused, "code_level_dropped", s.CodeDropped)
	return nil
}

// NewTrendsBuilder builds a platform build that reads the sources and writes the figures on sources (salus_stats in
// cmd/worker, the owner in salusctl), and audits on app (salus_stats cannot write audit_event); agent names it.
func NewTrendsBuilder(app, sources *pgxpool.Pool, minCount int, agent string) *trends.Builder {
	return trends.NewBuilder(trends.BuildPorts{
		Diagnoses:    condition.NewService(condition.NewPostgresRepository(sources)),
		Appointments: appointment.NewStats(appointment.NewPostgresStatsRepository(sources)),
		Specialties:  practitionerrole.NewStats(practitionerrole.NewPostgresStatsRepository(sources)),
		Locations:    organization.NewStats(organization.NewPostgresStatsRepository(sources)),
		Catalog:      terminology.NewCatalog(terminology.NewPostgresCatalogRepository(sources)),
		Repo:         trends.NewPostgresRepository(sources),
		Tx:           postgres.NewTransactor(sources),
		Audit:        auditevent.NewService(auditevent.NewPostgresRepository(app), postgres.NewTransactor(app), clock.System{}),
		Clock:        clock.System{},
		Agent:        agent,
		K:            minCount,
	})
}

// buildHour is the UTC hour of the trends builds.
const buildHour = 4

// trendsPeriodics are the scheduler's builds (§5, §6): on the 8th the month before, on the 8th after a quarter the
// quarter before, on Tuesdays the ISO week before; none while the platform view is off (§2a).
func trendsPeriodics(on bool, clk clock.Clock) []*river.PeriodicJob {
	if !on {
		return nil
	}
	build := func(name func(time.Time) string) func() (river.JobArgs, *river.InsertOpts) {
		return func() (river.JobArgs, *river.InsertOpts) {
			return TrendsBuildArgs{Period: name(clk.Now())}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
		}
	}
	return []*river.PeriodicJob{
		river.NewPeriodicJob(monthlyOn{day: 8, hour: buildHour}, build(trends.PreviousMonth), nil),
		river.NewPeriodicJob(monthlyOn{day: 8, hour: buildHour, quarterly: true}, build(trends.PreviousQuarter), nil),
		river.NewPeriodicJob(weeklyOn{weekday: time.Tuesday, hour: buildHour}, build(trends.PreviousWeek), nil),
	}
}

// monthlyOn fires at hour:00 UTC on day of every month, or of the first month of every quarter.
type monthlyOn struct {
	day, hour int
	quarterly bool
}

// Next returns the first such time strictly after t.
func (m monthlyOn) Next(t time.Time) time.Time {
	t = t.UTC()
	next := time.Date(t.Year(), t.Month(), m.day, m.hour, 0, 0, 0, time.UTC)
	for !next.After(t) || (m.quarterly && (next.Month()-1)%3 != 0) {
		next = next.AddDate(0, 1, 0)
	}
	return next
}

// weeklyOn fires at hour:00 UTC on weekday.
type weeklyOn struct {
	weekday time.Weekday
	hour    int
}

// Next returns the first such time strictly after t.
func (w weeklyOn) Next(t time.Time) time.Time {
	t = t.UTC()
	next := time.Date(t.Year(), t.Month(), t.Day(), w.hour, 0, 0, 0, time.UTC)
	for !next.After(t) || next.Weekday() != w.weekday {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
