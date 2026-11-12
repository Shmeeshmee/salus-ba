package worker

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// The queues of the chat's notifications (notification.NewRiverJobs): the notifier works push beside email, the
// worker works webhook beside its own.
const (
	pushQueue    = "push"
	webhookQueue = "webhook"
)

// webhookTimeout bounds one delivery attempt to a clinic's endpoint.
const webhookTimeout = 10 * time.Second

// webhookKeyLen is WEBHOOK_KEY's length: AES-256.
const webhookKeyLen = 32

// noticeChats is the chat's service for the notifications' reads alone: the repository, the care and chat consent
// checks (a doctor of another clinic is told only while the patient's chat consent for them is active), the audit,
// the transaction and the clock.
func noticeChats(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, clk clock.Clock) *communication.Service {
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil)
	return communication.NewService(communication.Ports{Repo: communication.NewPostgresRepository(pool), Consents: consents, ChatConsents: consents, Audit: audit, Tx: tx, Clock: clk})
}

// chatNotifierPorts are what the notifier's chat notifications run on.
type chatNotifierPorts struct {
	pool                      *pgxpool.Pool
	inserter                  *river.Client[pgx.Tx]
	audit                     *auditevent.Service
	tx                        *postgres.Transactor
	accounts                  *auth.Service
	sender                    notification.Sender
	pusher                    push.Sender
	webBaseURL, patientAppURL string
}

// newChatNotifier is the chat's notifications as the notifier runs them: the fan-out, the pushes and the unread
// e-mails.
func newChatNotifier(p chatNotifierPorts, clk clock.Clock) *notification.Chat {
	return notification.NewChat(notification.ChatPorts{
		Repo: notification.NewPostgresRepository(p.pool), Jobs: notification.NewRiverJobs(p.inserter),
		Chats:    noticeChats(p.pool, p.audit, p.tx, clk),
		Patients: patient.NewService(patient.NewPostgresRepository(p.pool), p.audit, p.tx, clk, nil, nil),
		Roles:    practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clk}),
		Accounts: p.accounts,
		Clinics:  organization.NewChat(organization.ChatPorts{Repo: organization.NewPostgresChatRepository(p.pool), Tx: p.tx, Clock: clk}),
		Pusher:   p.pusher, Sender: p.sender, Audit: p.audit, Tx: p.tx, Clock: clk,
		WebBaseURL: p.webBaseURL, PatientAppURL: p.patientAppURL,
	})
}

// newWebhooks is the clinics' webhook deliveries as the worker runs them, with WEBHOOK_KEY opening their secrets.
func newWebhooks(pool *pgxpool.Pool, key []byte, clk clock.Clock) (*notification.Chat, error) {
	if len(key) != webhookKeyLen {
		return nil, errors.New("worker: WEBHOOK_KEY must be 32 bytes")
	}
	box, err := auth.NewAESSecretBox(key)
	if err != nil {
		return nil, errors.New("worker: WEBHOOK_KEY must be 32 bytes")
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	return notification.NewChat(notification.ChatPorts{
		Repo: notification.NewPostgresRepository(pool), Chats: noticeChats(pool, audit, tx, clk),
		Poster: notification.NewWebhookPoster(webhookTimeout), Box: box, Audit: audit, Tx: tx, Clock: clk,
	}), nil
}

// addChatWorkers registers the notifier's chat jobs; the scheduler registers them with a nil chat.
func addChatWorkers(workers *river.Workers, chat *notification.Chat) {
	river.AddWorker(workers, &messageSentWorker{chat: chat})
	river.AddWorker(workers, &chatPushWorker{chat: chat})
	river.AddWorker(workers, &unreadEmailWorker{send: func(ctx context.Context, a notification.ChatUnreadEmailArgs) error { return chat.UnreadEmail(ctx, a) }})
}

// addWebhookWorkers registers the worker's webhook deliveries; the scheduler registers them with a nil chat.
func addWebhookWorkers(workers *river.Workers, chat *notification.Chat) {
	river.AddWorker(workers, &webhookWorker{deliver: func(ctx context.Context, a notification.WebhookDeliveryArgs, attempt, maxAttempts int) error {
		return chat.DeliverWebhook(ctx, a, attempt, maxAttempts)
	}})
}

// messageSentWorker fans a new message out to its notifications (SAL-29).
type messageSentWorker struct {
	river.WorkerDefaults[communication.MessageSentArgs]
	chat *notification.Chat
}

func (w *messageSentWorker) Work(ctx context.Context, job *river.Job[communication.MessageSentArgs]) error {
	return w.chat.MessageSent(ctx, job.Args)
}

// chatPushWorker sends one device's push.
type chatPushWorker struct {
	river.WorkerDefaults[notification.ChatPushArgs]
	chat *notification.Chat
}

func (w *chatPushWorker) Work(ctx context.Context, job *river.Job[notification.ChatPushArgs]) error {
	return w.chat.Push(ctx, job.Args)
}

// unreadEmailWorker e-mails a participant who has not read a message; an address SMTP cannot carry cancels the job,
// as a retry would fail the same way.
type unreadEmailWorker struct {
	river.WorkerDefaults[notification.ChatUnreadEmailArgs]
	send func(ctx context.Context, a notification.ChatUnreadEmailArgs) error
}

func (w *unreadEmailWorker) Work(ctx context.Context, job *river.Job[notification.ChatUnreadEmailArgs]) error {
	err := w.send(ctx, job.Args)
	if errors.Is(err, notification.ErrRecipientNotASCII) {
		return river.JobCancel(err)
	}
	return err
}

// webhookWorker makes one attempt of a delivery: River retries a failure with its backoff, and the last one is
// dead-lettered by the service; an endpoint deleted meanwhile, a conversation the clinic no longer sees, or a secret
// that does not open (dead-lettered at once) cancels it.
type webhookWorker struct {
	river.WorkerDefaults[notification.WebhookDeliveryArgs]
	deliver func(ctx context.Context, a notification.WebhookDeliveryArgs, attempt, maxAttempts int) error
}

func (w *webhookWorker) Work(ctx context.Context, job *river.Job[notification.WebhookDeliveryArgs]) error {
	err := w.deliver(ctx, job.Args, job.Attempt, job.MaxAttempts)
	if errors.Is(err, notification.ErrNotFound) || errors.Is(err, notification.ErrUndeliverable) {
		return river.JobCancel(err)
	}
	return err
}
