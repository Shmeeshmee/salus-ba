package invoice

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

// EnqueueFiscalize enqueues invoice_fiscalize on the default queue in the transaction in ctx.
func (j *riverJobs) EnqueueFiscalize(ctx context.Context, invoiceID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := FiscalizeArgs{InvoiceID: invoiceID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("invoice: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
