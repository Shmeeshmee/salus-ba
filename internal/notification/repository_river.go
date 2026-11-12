package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// The queues of the chat's notifications: the notifier works push (the fan-out and the pushes) and email (the
// unread e-mails); the worker works webhook, so a clinic's slow or failing endpoint never holds up a person's
// notification, and only the api and the worker hold the webhook key.
const (
	pushQueue    = "push"
	emailQueue   = "email"
	webhookQueue = "webhook"
)

// The attempts of each job: a push is stale within hours; a webhook is retried with River's backoff (attempt⁴
// seconds) for about an hour and a half before it is dead-lettered.
const (
	pushAttempts    = 5
	emailAttempts   = 10
	webhookAttempts = 8
)

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

func (j *riverJobs) EnqueuePush(ctx context.Context, a ChatPushArgs) error {
	return j.insert(ctx, a, &river.InsertOpts{Queue: pushQueue, MaxAttempts: pushAttempts})
}

func (j *riverJobs) EnqueueUnreadEmail(ctx context.Context, a ChatUnreadEmailArgs, at time.Time) error {
	// Unique by the participant and their read position, in River's default states: a completed check keeps the key
	// until River's cleaner removes it (a day), so no second e-mail while the participant reads nothing.
	return j.insert(ctx, a, &river.InsertOpts{Queue: emailQueue, MaxAttempts: emailAttempts, ScheduledAt: at, UniqueOpts: river.UniqueOpts{ByArgs: true}})
}

func (j *riverJobs) EnqueueWebhook(ctx context.Context, a WebhookDeliveryArgs) error {
	return j.insert(ctx, a, &river.InsertOpts{Queue: webhookQueue, MaxAttempts: webhookAttempts, UniqueOpts: river.UniqueOpts{ByArgs: true}})
}

// insert enqueues args with opts in the transaction in ctx; a unique job already there is kept, not an error.
func (j *riverJobs) insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, args, opts); err != nil {
		return fmt.Errorf("notification: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
