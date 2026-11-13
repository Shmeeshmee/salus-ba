package communication

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// pushQueue is the notifier's queue of the chat's notifications (cmd/notifier works it beside the e-mail queue).
const pushQueue = "push"

// messageSentAttempts bounds the retries of a fan-out: past them a message's notifications are given up (the
// message itself is in the record and the app shows it).
const messageSentAttempts = 10

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

func (j *riverJobs) EnqueueMessageSent(ctx context.Context, a MessageSentArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, a, &river.InsertOpts{Queue: pushQueue, MaxAttempts: messageSentAttempts}); err != nil {
		return fmt.Errorf("communication: enqueue %s: %w", a.Kind(), err)
	}
	return nil
}
