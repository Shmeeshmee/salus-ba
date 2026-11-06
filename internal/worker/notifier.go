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
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// emailQueue is the only queue the notifier works (ADR 0007); cmd/worker keeps the default one.
const emailQueue = "email"

// NewInserter returns an insert-only River client on pool, for the api and salusctl to enqueue jobs with
// InsertTx. It works no queue.
func NewInserter(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

// NewNotifier returns the River client cmd/notifier runs: it works the email queue with the invite_email and
// appointment e-mail workers, which send through SMTP at smtpAddr (SMTP_ADDR) with links to webBaseURL
// (WEB_BASE_URL, invites) and patientAppURL (PATIENT_APP_URL, appointments).
func NewNotifier(pool *pgxpool.Pool, log *slog.Logger, smtpAddr, webBaseURL, patientAppURL string) (*river.Client[pgx.Tx], error) {
	if smtpAddr == "" || webBaseURL == "" || patientAppURL == "" {
		return nil, errors.New("worker: the notifier needs SMTP_ADDR, WEB_BASE_URL and PATIENT_APP_URL")
	}
	clk := clock.System{}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk)
	// Issuing a token and reading an account's e-mail need only the repository, the audit, the transaction
	// and the clock.
	accounts := auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	// A notice reads the appointment and its patient's profile: nothing else of either service.
	patients := patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk, nil, nil)
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Patients: patients,
	})
	mail := notification.NewService(notification.Ports{
		Invites: accounts, Appointments: appointments, Accounts: accounts, Clinics: clinics,
		Questionnaires: newQuestionnaireResponses(pool, clk), Sender: notification.NewSMTPSender(smtpAddr), WebBaseURL: webBaseURL, PatientAppURL: patientAppURL,
	})

	workers := river.NewWorkers()
	addEmailWorkers(workers, mail)
	return river.NewClient(riverpgxv5.New(pool), workConfig(log, emailQueue, workers))
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
	river.AddWorker(workers, &appointmentEmailWorker[appointment.ReminderEmailArgs]{send: func(ctx context.Context, a appointment.ReminderEmailArgs) error {
		return mail.SendAppointmentReminder(ctx, a.AppointmentID, a.OrganizationID)
	}})
}

// appointmentEmailWorker sends one of an appointment's e-mails. An e-mail that no longer applies sends
// nothing and completes; an appointment that does not exist in the clinic cancels the job, as a retry would
// find nothing either.
type appointmentEmailWorker[T river.JobArgs] struct {
	river.WorkerDefaults[T]
	send func(ctx context.Context, args T) error
}

func (w *appointmentEmailWorker[T]) Work(ctx context.Context, job *river.Job[T]) error {
	err := w.send(ctx, job.Args)
	if errors.Is(err, appointment.ErrNotFound) {
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
	return err
}
