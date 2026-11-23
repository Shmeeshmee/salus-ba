package subscription

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// emailQueue is the notifier's queue (internal/worker).
const emailQueue = "email"

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

// EnqueueInvoiceEmail enqueues subscription_invoice_email on the notifier's queue in the transaction in ctx.
func (j *riverJobs) EnqueueInvoiceEmail(ctx context.Context, a InvoiceEmailArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, a, &river.InsertOpts{Queue: emailQueue}); err != nil {
		return fmt.Errorf("subscription: enqueue %s: %w", a.Kind(), err)
	}
	return nil
}
