package worker

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// statsWorkers is how many clinic statistics' builds run at a time (clinicstats.QueueStats): each holds a
// salus_stats connection, so the stats pool needs no more than these and the default queue never waits on them.
const statsWorkers = 2

// clinicStatsAt is the daily time of the clinicstats_nightly job, 02:30 UTC (docs/plans/analytics.md §4).
var clinicStatsAt = dailyAt{hour: 2, minute: 30}

// clinicStatsRefreshWorker recomputes a clinic-day of the clinic's statistics (clinicstats_refresh). It logs
// nothing: a failure is retried by River with its error.
type clinicStatsRefreshWorker struct {
	river.WorkerDefaults[clinicstats.RefreshArgs]
	builds *clinicstats.Builder
}

func (w *clinicStatsRefreshWorker) Work(ctx context.Context, job *river.Job[clinicstats.RefreshArgs]) error {
	return w.builds.Refresh(ctx, job.Args.OrganizationID, job.Args.Day)
}

// clinicStatsNightlyWorker recomputes the last 35 days of every clinic (clinicstats_nightly), which repairs a lost
// refresh.
type clinicStatsNightlyWorker struct {
	river.WorkerDefaults[clinicstats.NightlyArgs]
	builds *clinicstats.Builder
}

func (w *clinicStatsNightlyWorker) Work(ctx context.Context, _ *river.Job[clinicstats.NightlyArgs]) error {
	return w.builds.RefreshRecent(ctx)
}

// newClinicStats builds the clinic statistics' Builder on statsPool (salus_stats): the sources' stats readers
// and the aggregates' writes, in one transaction of that pool.
func newClinicStats(statsPool *pgxpool.Pool, clk clock.Clock) *clinicstats.Builder {
	return clinicstats.NewBuilder(clinicstats.BuildPorts{
		Repo: clinicstats.NewPostgresBuildRepository(statsPool), Tx: postgres.NewTransactor(statsPool), Clock: clk,
		Clinics:      organization.NewStats(organization.NewPostgresStatsRepository(statsPool)),
		Appointments: appointment.NewStats(appointment.NewPostgresStatsRepository(statsPool)),
		Conditions:   condition.NewService(condition.NewPostgresRepository(statsPool)),
		Specialties:  practitionerrole.NewStats(practitionerrole.NewPostgresStatsRepository(statsPool)),
		Groups:       terminology.NewCatalog(terminology.NewPostgresCatalogRepository(statsPool)),
	})
}
