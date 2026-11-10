package composition

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// emailQueue is the queue the notifier works (ADR 0007: e-mail is sent by the notifier).
const emailQueue = "email"

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

// EnqueueSigned enqueues document_signed_email on the email queue in the transaction in ctx.
func (j *riverJobs) EnqueueSigned(ctx context.Context, appointmentID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := SignedEmailArgs{AppointmentID: appointmentID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: emailQueue}); err != nil {
		return fmt.Errorf("composition: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
