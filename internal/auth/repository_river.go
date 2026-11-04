package auth

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

func (j *riverJobs) EnqueueInviteEmail(ctx context.Context, inviteID, organizationID string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := InviteEmailArgs{InviteID: inviteID, OrganizationID: organizationID}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: emailQueue}); err != nil {
		return fmt.Errorf("auth: enqueue invite_email: %w", err)
	}
	return nil
}
