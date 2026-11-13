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

// The send jobs of the reminders' deliveries (SAL-33), one queue per channel: the notifier works email, push and sms.
const (
	smsQueue    = "sms"
	smsAttempts = 5
)

// NewRiverDeliveryJobs returns the DeliveryJobs port on a River client (an insert-only one will do).
func NewRiverDeliveryJobs(client *river.Client[pgx.Tx]) DeliveryJobs {
	return &riverJobs{client: client}
}

// EnqueueDelivery enqueues the send of the delivery id on its channel's queue at at, unique by the delivery.
func (j *riverJobs) EnqueueDelivery(ctx context.Context, channel Channel, id string, at time.Time) error {
	unique := river.UniqueOpts{ByArgs: true}
	switch channel {
	case ChannelEmail:
		return j.insert(ctx, EmailDeliveryArgs{DeliveryID: id}, &river.InsertOpts{Queue: emailQueue, MaxAttempts: emailAttempts, ScheduledAt: at, UniqueOpts: unique})
	case ChannelPush:
		return j.insert(ctx, PushDeliveryArgs{DeliveryID: id}, &river.InsertOpts{Queue: pushQueue, MaxAttempts: pushAttempts, ScheduledAt: at, UniqueOpts: unique})
	case ChannelSMS:
		return j.insert(ctx, SMSDeliveryArgs{DeliveryID: id}, &river.InsertOpts{Queue: smsQueue, MaxAttempts: smsAttempts, ScheduledAt: at, UniqueOpts: unique})
	}
	return fmt.Errorf("notification: no queue for the channel %q", channel)
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
