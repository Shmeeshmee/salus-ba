package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
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

// NewNotifier returns the River client cmd/notifier runs: it works the email queue with the invite_email
// worker, which sends through SMTP at smtpAddr (SMTP_ADDR) with links to webBaseURL (WEB_BASE_URL).
func NewNotifier(pool *pgxpool.Pool, log *slog.Logger, smtpAddr, webBaseURL string) (*river.Client[pgx.Tx], error) {
	if smtpAddr == "" || webBaseURL == "" {
		return nil, errors.New("worker: the notifier needs SMTP_ADDR and WEB_BASE_URL")
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{})
	// Issuing a token needs only the repository, the transaction and the clock.
	invites := auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Tx: tx, Clock: clock.System{}})
	mail := notification.NewService(invites, clinics, notification.NewSMTPSender(smtpAddr), webBaseURL)

	workers := river.NewWorkers()
	river.AddWorker(workers, &inviteEmailWorker{mail: mail})
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:              log,
		Queues:              map[string]river.QueueConfig{emailQueue: {MaxWorkers: maxWorkers}},
		ReindexerIndexNames: []string{}, // as in config: the notifier logs in as salus_app, which owns nothing
		Workers:             workers,
	})
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
