package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/notification"
)

// dataRequestWarningWorker sends a data request's deadline warning (data_request_deadline_warning, queued by the
// deadline job; docs/clinic-cms/WAVE5.md D5) on the email queue. A request answered or withdrawn since completes the job
// without a send. Without an operator address (dev only: config.LoadOperator) the account target's warning is logged by
// its opaque ids instead.
type dataRequestWarningWorker struct {
	river.WorkerDefaults[datarequest.WarningArgs]
	requests *datarequest.Service
	mail     *notification.Service
	log      *slog.Logger
}

func (w *dataRequestWarningWorker) Work(ctx context.Context, job *river.Job[datarequest.WarningArgs]) error {
	r, loc, err := w.requests.Warning(ctx, job.Args)
	switch {
	case errors.Is(err, datarequest.ErrNotFound):
		return nil // closed since: nothing to warn of
	case err != nil:
		return err
	}
	sent, err := w.mail.SendDataRequestDeadline(ctx, notification.DataRequestDeadline{
		Deadline: r.Deadline(), Zone: loc, RequestID: r.ID, OrganizationID: r.OrganizationID, Kind: string(r.Kind), Days: job.Args.Days,
	})
	switch {
	case errors.Is(err, notification.ErrNoOperator):
		w.log.InfoContext(ctx, "data request deadline warning not sent: no operator address", "request_id", r.ID, "days", job.Args.Days)
		return nil
	case errors.Is(err, notification.ErrRecipientNotASCII) && sent > 0:
		w.log.WarnContext(ctx, "data request deadline warning: an admin's address SMTP cannot carry", "request_id", r.ID, "organization_id", r.OrganizationID)
		return nil // the others are warned; a retry fails the same way
	case errors.Is(err, notification.ErrRecipientNotASCII):
		return river.JobCancel(err) // an address SMTP cannot carry: a retry fails the same way
	case err == nil && sent == 0:
		w.log.WarnContext(ctx, "data request deadline warning not sent: the clinic has no admin", "request_id", r.ID, "organization_id", r.OrganizationID)
	}
	return err
}

// addDataRequestMailWorkers registers the deadline warning's worker; the scheduler registers it with nil services.
func addDataRequestMailWorkers(workers *river.Workers, requests *datarequest.Service, mail *notification.Service, log *slog.Logger) {
	river.AddWorker(workers, &dataRequestWarningWorker{requests: requests, mail: mail, log: log})
}
