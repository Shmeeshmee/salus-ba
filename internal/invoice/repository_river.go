package invoice

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

// EnqueueFiscalize enqueues invoice_fiscalize on the default queue in the transaction in ctx, unless the receipt's
// report is already queued or running: a retry clicked twice reports it once (clinic-cms H). A job that finished,
// gave up (discarded) or was cancelled does not count, so a failed receipt can always be reported again.
func (j *riverJobs) EnqueueFiscalize(ctx context.Context, invoiceID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := FiscalizeArgs{InvoiceID: invoiceID, OrganizationID: org}
	opts := &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: []rivertype.JobState{
		rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable, rivertype.JobStateRunning, rivertype.JobStateScheduled,
	}}}
	if _, err := j.client.InsertTx(ctx, tx, args, opts); err != nil {
		return fmt.Errorf("invoice: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}

// emailQueue is the queue the notifier works (ADR 0007: e-mail is sent by the notifier).
const emailQueue = "email"

// EnqueueQuoteEmail enqueues quote_email on the email queue in the transaction in ctx.
func (j *riverJobs) EnqueueQuoteEmail(ctx context.Context, appointmentID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := QuoteEmailArgs{AppointmentID: appointmentID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: emailQueue}); err != nil {
		return fmt.Errorf("invoice: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
