package appointment

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// emailQueue is the queue the notifier works (ADR 0007: e-mail is sent by the notifier); pushQueue its other one,
// where a reminder's dispatch runs (SAL-33).
const (
	emailQueue = "email"
	pushQueue  = "push"
)

// reminderAttempts bounds a reminder's dispatch: a reminder is stale within hours, and the sweep repairs a lost one.
const reminderAttempts = 10

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

// EnqueueReminders enqueues each reminder's dispatch on the notifier's push queue at its due time, unique by the
// reminder in River's default states: the sweep's repeat of a job still there (or completed) keeps it.
func (j *riverJobs) EnqueueReminders(ctx context.Context, rs []Reminder) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	params := make([]river.InsertManyParams, 0, len(rs))
	for _, r := range rs {
		params = append(params, river.InsertManyParams{
			Args: ReminderArgs{ReminderID: r.ID, OrganizationID: r.OrganizationID},
			InsertOpts: &river.InsertOpts{
				Queue: pushQueue, MaxAttempts: reminderAttempts, ScheduledAt: r.DueAt, UniqueOpts: river.UniqueOpts{ByArgs: true},
			},
		})
	}
	if _, err := j.client.InsertManyTx(ctx, tx, params); err != nil {
		return fmt.Errorf("appointment: enqueue %s: %w", ReminderArgs{}.Kind(), err)
	}
	return nil
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
