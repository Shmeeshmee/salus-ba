package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// erasureHour is the UTC hour of the daily account_erasure job, after retention_apply.
const erasureHour = 4

// AccountErasureArgs erases the closed accounts whose erasure date has come (datarequest.Erasure,
// docs/clinic-cms/WAVE5.md S6). The scheduler enqueues it daily; it has no arguments.
type AccountErasureArgs struct{}

// Kind names the job in the river_job table.
func (AccountErasureArgs) Kind() string { return "account_erasure" }

// accountEraser is datarequest.Erasure; the worker's test fakes it.
type accountEraser interface {
	Run(ctx context.Context) ([]datarequest.Erased, error)
}

// accountErasureWorker runs the erasure and logs, per account erased, the opaque ids and how many clinics keep a
// record: nothing else.
type accountErasureWorker struct {
	river.WorkerDefaults[AccountErasureArgs]
	erasure accountEraser
	log     *slog.Logger
}

func (w *accountErasureWorker) Work(ctx context.Context, _ *river.Job[AccountErasureArgs]) error {
	if w.erasure == nil { // not built in this binary yet: cancelled, not retried
		return river.JobCancel(errors.New("worker: the account erasure is not wired"))
	}
	erased, err := w.erasure.Run(ctx)
	for _, e := range erased {
		w.log.InfoContext(ctx, "account erased", "account_id", e.AccountID, "data_request_id", e.RequestID, "clinics_kept", len(e.Kept))
	}
	return err
}
