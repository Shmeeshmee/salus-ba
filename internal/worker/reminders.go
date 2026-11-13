package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
)

// The reminders (SAL-33, docs/plans/appointments.md, ADR 0015 "Reminders"): the notifier dispatches each reminder at
// its due time on the push queue and sends its deliveries on the email, push and sms queues; the worker sweeps the
// lost ones every hour (the scheduler enqueues the sweep).

// smsQueue is the notifier's queue of the SMS sends.
const smsQueue = "sms"

// remindersSweepInterval is how often appointment_reminders_sweep runs; sweepPage how many reminders one of its
// transactions takes.
const (
	remindersSweepInterval = time.Hour
	sweepPage              = 1000
)

// addReminderWorkers registers the notifier's reminder jobs; the scheduler registers them with a nil mail. (The
// appointment_reminder_email job of before SAL-33 is no longer enqueued, and still sends until its jobs drain:
// addEmailWorkers.)
func addReminderWorkers(workers *river.Workers, mail *notification.Service) {
	river.AddWorker(workers, &reminderDispatchWorker{mail: mail})
	river.AddWorker(workers, &deliveryWorker[notification.EmailDeliveryArgs]{mail: mail})
	river.AddWorker(workers, &deliveryWorker[notification.PushDeliveryArgs]{mail: mail})
	river.AddWorker(workers, &deliveryWorker[notification.SMSDeliveryArgs]{mail: mail})
}

// reminderDispatchWorker dispatches a reminder; one not of its clinic cancels the job, as a retry would fail the same.
type reminderDispatchWorker struct {
	river.WorkerDefaults[appointment.ReminderArgs]
	mail *notification.Service
}

func (w *reminderDispatchWorker) Work(ctx context.Context, job *river.Job[appointment.ReminderArgs]) error {
	err := w.mail.DispatchReminder(ctx, job.Args)
	if errors.Is(err, appointment.ErrNotFound) {
		return river.JobCancel(err)
	}
	return err
}

// deliveryArgs are the three send jobs' arguments.
type deliveryArgs interface {
	notification.EmailDeliveryArgs | notification.PushDeliveryArgs | notification.SMSDeliveryArgs
	river.JobArgs
}

// deliveryWorker sends one delivery; its last attempt fails the delivery for good.
type deliveryWorker[T deliveryArgs] struct {
	river.WorkerDefaults[T]
	mail *notification.Service
}

func (w *deliveryWorker[T]) Work(ctx context.Context, job *river.Job[T]) error {
	return w.mail.SendDelivery(ctx, notification.DeliveryArgs(job.Args).DeliveryID, job.Attempt >= job.MaxAttempts)
}

// remindersSweepWorker enqueues again the dispatch of the reminders whose job was lost (on the default queue).
type remindersSweepWorker struct {
	river.WorkerDefaults[appointment.RemindersSweepArgs]
	appointments *appointment.Service
	log          *slog.Logger
}

func (w *remindersSweepWorker) Work(ctx context.Context, _ *river.Job[appointment.RemindersSweepArgs]) error {
	n, err := w.appointments.SweepReminders(ctx, sweepPage)
	if n > 0 {
		w.log.InfoContext(ctx, "reminders swept", "overdue", n)
	}
	return err
}
