package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// deadlinesInterval is how often data_request_deadlines runs (docs/clinic-cms/WAVE5.md D1, S3): hourly, so the 1-day
// warning goes out within the hour its day begins.
const deadlinesInterval = time.Hour

// dataRequestDeadlinesWorker queues the deadline warnings of the data subject's requests (data_request_deadlines). It
// logs a count only.
type dataRequestDeadlinesWorker struct {
	river.WorkerDefaults[datarequest.DeadlinesArgs]
	requests *datarequest.Service
	log      *slog.Logger
}

func (w *dataRequestDeadlinesWorker) Work(ctx context.Context, _ *river.Job[datarequest.DeadlinesArgs]) error {
	n, err := w.requests.QueueWarnings(ctx)
	if n > 0 {
		w.log.InfoContext(ctx, "data request deadline warnings queued", "count", n)
	}
	return err
}

// newDataRequests builds the requests' service on pool (salus_app) for the deadline job: its warnings are enqueued
// with inserter in the job's transaction.
func newDataRequests(pool *pgxpool.Pool, inserter *river.Client[pgx.Tx], clk clock.Clock) *datarequest.Service {
	tx := postgres.NewTransactor(pool)
	return datarequest.NewService(datarequest.Ports{
		Repo: datarequest.NewPostgresRepository(pool), Clock: clk, Tx: tx,
		Audit: auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk), Jobs: datarequest.NewRiverJobs(inserter),
	})
}
