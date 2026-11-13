package appointment

import (
	"context"
	"fmt"
	"time"

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

func (j *riverJobs) EnqueueBooked(ctx context.Context, id, org string) error {
	return j.insert(ctx, BookedEmailArgs{AppointmentID: id, OrganizationID: org}, time.Time{})
}

func (j *riverJobs) EnqueueCancelled(ctx context.Context, id, org string) error {
	return j.insert(ctx, CancelledEmailArgs{AppointmentID: id, OrganizationID: org}, time.Time{})
}

func (j *riverJobs) EnqueueReminder(ctx context.Context, id, org string, at time.Time) error {
	return j.insert(ctx, ReminderEmailArgs{AppointmentID: id, OrganizationID: org}, at)
}

// EnqueueAccessEnd enqueues consultation_access_end on the default queue, which the worker works.
func (j *riverJobs) EnqueueAccessEnd(ctx context.Context, id, org string) error {
	return j.insertOn(ctx, river.QueueDefault, AccessEndArgs{AppointmentID: id, OrganizationID: org}, time.Time{})
}

// EnqueueQuestionnaireRequest enqueues questionnaire_default_request on the default queue, which the worker
// works.
func (j *riverJobs) EnqueueQuestionnaireRequest(ctx context.Context, id, org string) error {
	return j.insertOn(ctx, river.QueueDefault, QuestionnaireRequestArgs{AppointmentID: id, OrganizationID: org}, time.Time{})
}

// EnqueueQuestionnaireDiscard enqueues questionnaire_responses_discard on the default queue, which the worker
// works.
func (j *riverJobs) EnqueueQuestionnaireDiscard(ctx context.Context, id, org string) error {
	return j.insertOn(ctx, river.QueueDefault, QuestionnaireDiscardArgs{AppointmentID: id, OrganizationID: org}, time.Time{})
}

// EnqueueSlotFreed enqueues waitlist_slot_freed on the default queue, which the worker works.
func (j *riverJobs) EnqueueSlotFreed(ctx context.Context, slotID, org string) error {
	return j.insertOn(ctx, river.QueueDefault, SlotFreedArgs{SlotID: slotID, OrganizationID: org}, time.Time{})
}

// NewRiverOfferJobs returns the OfferJobs port on a River client (SAL-35).
func NewRiverOfferJobs(client *river.Client[pgx.Tx]) OfferJobs {
	return &riverJobs{client: client}
}

// NewRiverOfferNotices returns the OfferNotices port on a River client: the waitlist_offer_notify job, until SAL-33's
// deliveries take its place (SAL-35).
func NewRiverOfferNotices(client *river.Client[pgx.Tx]) OfferNotices {
	return &riverJobs{client: client}
}

// EnqueueOfferExpire enqueues waitlist_offer_expire on the default queue, which the worker works, at the offer's end.
func (j *riverJobs) EnqueueOfferExpire(ctx context.Context, offerID, org string, at time.Time) error {
	return j.insertOn(ctx, river.QueueDefault, OfferExpireArgs{OfferID: offerID, OrganizationID: org}, at)
}

// NotifyOffer enqueues waitlist_offer_notify on the email queue, which the notifier works.
func (j *riverJobs) NotifyOffer(ctx context.Context, offerID, org string) error {
	return j.insert(ctx, OfferNotifyArgs{OfferID: offerID, OrganizationID: org}, time.Time{})
}

// insert enqueues args on the email queue in the transaction in ctx, to run at at (zero for now).
func (j *riverJobs) insert(ctx context.Context, args river.JobArgs, at time.Time) error {
	return j.insertOn(ctx, emailQueue, args, at)
}

// insertOn enqueues args on queue in the transaction in ctx, to run at at (zero for now).
func (j *riverJobs) insertOn(ctx context.Context, queue string, args river.JobArgs, at time.Time) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: queue, ScheduledAt: at}); err != nil {
		return fmt.Errorf("appointment: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
