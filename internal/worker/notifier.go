package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// emailQueue is the only queue the notifier works (ADR 0007); cmd/worker keeps the default one.
const emailQueue = "email"

// NewInserter returns an insert-only River client on pool, for the api and salusctl to enqueue jobs with
// InsertTx. It works no queue.
func NewInserter(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

// Notifier is what the notifier sends through and links to: the SMTP server of config.LoadSMTP; the push sender (the
// fake, pkg/push/pushtest, until the platforms' credentials are set: config.LoadPush); the SMS sender (the fake,
// pkg/sms/smstest, until an SMS provider is chosen: config.LoadSMS); WEB_BASE_URL (invites), PATIENT_APP_URL
// (appointments) and APPOINTMENT_LINK_URL (the reminders' one-time links, the api's public origin).
type Notifier struct {
	Sender        notification.Sender
	Pusher        push.Sender
	SMS           sms.Sender
	WebBaseURL    string
	PatientAppURL string
	LinkURL       string
}

// NewNotifier returns the River client cmd/notifier runs: it works the email queue with the invite_email and
// appointment e-mail workers, the chat's notifications (SAL-29) on the push queue (the fan-out and the pushes) and
// the unread e-mails, and the reminders (SAL-33): their dispatch on the push queue and their sends on the email, push
// and sms queues.
func NewNotifier(pool *pgxpool.Pool, log *slog.Logger, n Notifier) (*river.Client[pgx.Tx], error) {
	if n.Sender == nil || n.Pusher == nil || n.SMS == nil || n.WebBaseURL == "" || n.PatientAppURL == "" || n.LinkURL == "" {
		return nil, errors.New("worker: the notifier needs an SMTP sender, a push sender, an SMS sender, WEB_BASE_URL, PATIENT_APP_URL and APPOINTMENT_LINK_URL")
	}
	inserter, err := NewInserter(pool) // the fan-out and the dispatch enqueue their sends in their transaction
	if err != nil {
		return nil, err
	}
	clk := clock.System{}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk, nil)
	// Issuing a token and reading an account's e-mail need only the repository, the audit, the transaction
	// and the clock.
	accounts := auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	// A notice reads the appointment and its patient's profile: nothing else of either service.
	patients := patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk, nil, nil)
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Patients: patients,
		Reminders: appointment.NewPostgresReminderRepository(pool), Tokens: appointment.NewPostgresTokenRepository(pool), LinkURL: n.LinkURL,
	})
	mail := notification.NewService(notification.Ports{
		Invites: accounts, Appointments: appointments, Accounts: accounts, Clinics: clinics,
		Questionnaires: newQuestionnaireResponses(pool, clk), Sender: n.Sender, WebBaseURL: n.WebBaseURL, PatientAppURL: n.PatientAppURL,
		Deliveries: notification.NewPostgresDeliveryRepository(pool), DeliveryJobs: notification.NewRiverDeliveryJobs(inserter),
		Devices: notification.NewPostgresRepository(pool), Reminders: appointments,
		Rules:     organization.NewAppointments(organization.AppointmentsPorts{Repo: organization.NewPostgresAppointmentsRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Languages: accounts, Phones: patients, Pusher: n.Pusher, SMS: n.SMS, Audit: audit, Tx: tx, Clock: clk,
	})

	chat := newChatNotifier(chatNotifierPorts{
		pool: pool, inserter: inserter, audit: audit, tx: tx, accounts: accounts, sender: n.Sender, pusher: n.Pusher,
		webBaseURL: n.WebBaseURL, patientAppURL: n.PatientAppURL,
	}, clk)

	workers := river.NewWorkers()
	addEmailWorkers(workers, mail)
	addChatWorkers(workers, chat)
	addReminderWorkers(workers, mail)
	cfg := workConfig(log, emailQueue, workers)
	cfg.Queues[pushQueue] = river.QueueConfig{MaxWorkers: maxWorkers}
	cfg.Queues[smsQueue] = river.QueueConfig{MaxWorkers: maxWorkers}
	return river.NewClient(riverpgxv5.New(pool), cfg)
}

// addEmailWorkers registers the email queue's jobs; the scheduler registers them with a nil mail, like
// addDefaultWorkers.
func addEmailWorkers(workers *river.Workers, mail *notification.Service) {
	river.AddWorker(workers, &inviteEmailWorker{mail: mail})
	river.AddWorker(workers, &appointmentEmailWorker[appointment.BookedEmailArgs]{send: func(ctx context.Context, a appointment.BookedEmailArgs) error {
		return mail.SendAppointmentBooked(ctx, a.AppointmentID, a.OrganizationID)
	}})
	river.AddWorker(workers, &appointmentEmailWorker[appointment.CancelledEmailArgs]{send: func(ctx context.Context, a appointment.CancelledEmailArgs) error {
		return mail.SendAppointmentCancelled(ctx, a.AppointmentID, a.OrganizationID)
	}})
	// No longer enqueued (SAL-33's reminders replace it): sent for the appointments booked before, until its jobs drain
	// (the plan's "Legacy reminder"); it goes in a later release.
	river.AddWorker(workers, &appointmentEmailWorker[appointment.ReminderEmailArgs]{send: func(ctx context.Context, a appointment.ReminderEmailArgs) error {
		return mail.SendAppointmentReminder(ctx, a.AppointmentID, a.OrganizationID)
	}})
	river.AddWorker(workers, &appointmentEmailWorker[appointment.RescheduledEmailArgs]{send: func(ctx context.Context, a appointment.RescheduledEmailArgs) error {
		return mail.SendAppointmentRescheduled(ctx, a.AppointmentID, a.OrganizationID)
	}})
	river.AddWorker(workers, &appointmentEmailWorker[composition.SignedEmailArgs]{send: func(ctx context.Context, a composition.SignedEmailArgs) error {
		return mail.SendDocumentSigned(ctx, a.AppointmentID, a.OrganizationID)
	}})
}

// appointmentEmailWorker sends one of an appointment's e-mails. An e-mail that no longer applies sends
// nothing and completes; an appointment that does not exist in the clinic, or a recipient address SMTP cannot
// carry, cancels the job, as a retry would fail the same way.
type appointmentEmailWorker[T river.JobArgs] struct {
	river.WorkerDefaults[T]
	send func(ctx context.Context, args T) error
}

func (w *appointmentEmailWorker[T]) Work(ctx context.Context, job *river.Job[T]) error {
	err := w.send(ctx, job.Args)
	if errors.Is(err, appointment.ErrNotFound) || errors.Is(err, notification.ErrRecipientNotASCII) {
		return river.JobCancel(err)
	}
	return err
}

// inviteEmailWorker sends an invite's e-mail. A retry issues a new token, so only the newest link works
// (docs/accounts/PROGRESS.md, package 5).
type inviteEmailWorker struct {
	river.WorkerDefaults[auth.InviteEmailArgs]
	mail *notification.Service
}

func (w *inviteEmailWorker) Work(ctx context.Context, job *river.Job[auth.InviteEmailArgs]) error {
	err := w.mail.SendInvite(ctx, job.Args.InviteID, job.Args.OrganizationID)
	if errors.Is(err, auth.ErrInvalidInvite) {
		return river.JobCancel(err) // replaced, accepted or expired: nothing to send, now or on a retry
	}
	if errors.Is(err, notification.ErrRecipientNotASCII) {
		return river.JobCancel(err) // an address SMTP cannot carry: a retry fails the same way
	}
	return err
}
