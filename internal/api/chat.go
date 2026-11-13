package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
)

// chatHubBuffer is how many live frames one socket may fall behind before the hub drops for it (it then fills
// from Postgres, ADR 0012).
const chatHubBuffer = 64

// chatPorts are what the chat is built on.
type chatPorts struct {
	db         postgres.Querier
	rdb        *goredis.Client
	audit      *auditevent.Service
	tx         *postgres.Transactor
	consents   *consent.Service
	visits     *appointment.Facts
	roles      *practitionerrole.Service
	encounters *encounter.Service
	jobs       *river.Client[pgx.Tx] // a send enqueues its notifications (SAL-29)
	caller     func(ctx context.Context) (appointment.Caller, bool, error)
	active     func(r *http.Request) (bool, error) // the socket's session is still open (auth.Handler.Active)
	summarizer llm.Summarizer                      // the AI summary's engine (config.LoadChatSummary: the fake)
	// SAL-30, each optional: the conversations' files (binary with a conversation as their parent), and the caller's
	// UI language for the terms' tagging (English without it).
	files    *binary.Service
	language func(ctx context.Context, accountID string) (string, error)
}

// newChat builds the chat (docs/plans/chat.md): its routes, its socket, and the hub that carries the live frames
// over one Redis connection, which the App closes. Receipts and typing go through the same Redis (SAL-30).
func newChat(p chatPorts, log *slog.Logger) (routes, stream func(*http.ServeMux), hub *redis.Hub) {
	hub = redis.NewHub(context.Background(), p.rdb, chatHubBuffer)
	ports := communication.Ports{
		Repo: communication.NewPostgresRepository(p.db), Publisher: hub, Consents: p.consents, Visits: p.visits,
		Roles: p.roles, Encounters: p.encounters, ChatConsents: p.consents, Directory: p.roles, Audit: p.audit, Tx: p.tx,
		Clock: clock.System{}, Summarizer: p.summarizer, Log: log, Jobs: communication.NewRiverJobs(p.jobs),
		Presence: communication.NewRedisPresence(p.rdb), Terms: terminology.NewService(terminology.NewPostgresRepository(p.db), nil),
	}
	if p.files != nil {
		ports.Files = p.files
	}
	if p.language != nil {
		ports.Languages = languageFunc(p.language)
	}
	svc := communication.NewService(ports)
	h := communication.NewHTTPHandler(svc, hub, p.caller, p.active, clock.System{}, communication.CheckEvery, log)
	group := communication.NewGroupHandler(svc, h)
	routes = func(mux *http.ServeMux) {
		h.Routes(mux)
		group.Routes(mux) // the group consult (SAL-28)
	}
	return routes, h.StreamRoutes, hub
}

// chatNotificationPorts are what the chat's settings, devices and webhooks are built on.
type chatNotificationPorts struct {
	pool     *pgxpool.Pool
	audit    *auditevent.Service
	tx       *postgres.Transactor
	consents *consent.Service
	accounts *auth.Service
	patients *patient.Service
	box      *auth.AESSecretBox // WEBHOOK_KEY's box: the webhook secrets
}

// chatNotificationRoutes are the routes of SAL-29: the clinic's chat settings and hours (organization), the app's
// devices and the clinic's webhooks (notification).
func chatNotificationRoutes(p chatNotificationPorts, log *slog.Logger) []func(*http.ServeMux) {
	settings := organization.NewChat(organization.ChatPorts{
		Repo: organization.NewPostgresChatRepository(p.pool), Consents: p.consents, Audit: p.audit, Tx: p.tx, Clock: clock.System{},
	})
	notifications := notification.NewChat(notification.ChatPorts{
		Repo: notification.NewPostgresRepository(p.pool), Box: p.box, Audit: p.audit, Tx: p.tx, Clock: clock.System{},
	})
	apptCaller := appointmentCaller(p.accounts, p.patients)
	settingsCaller := func(ctx context.Context) (organization.Caller, bool, error) {
		c, ok, err := apptCaller(ctx)
		return organization.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PatientID: c.PatientID, Admin: c.Admin}, ok, err
	}
	clinicCaller := organizationCaller(p.accounts)
	notificationCaller := func(ctx context.Context) (notification.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return notification.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	// The clinic's appointment settings (docs/plans/appointments.md, step 0) sit beside its chat settings.
	appointmentRules := organization.NewAppointments(organization.AppointmentsPorts{
		Repo: organization.NewPostgresAppointmentsRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
	})
	return []func(*http.ServeMux){
		organization.NewHTTPHandler(settings, settingsCaller, log).Routes,
		notification.NewHTTPHandler(notifications, notificationCaller, log).Routes,
		organization.NewAppointmentsHandler(appointmentRules, clinicCaller, log).Routes,
	}
}

// languageFunc is a function as the chat's Languages port.
type languageFunc func(ctx context.Context, accountID string) (string, error)

func (f languageFunc) Language(ctx context.Context, accountID string) (string, error) {
	return f(ctx, accountID)
}

// newChatFiles is binary for the chat's files (SAL-30): the questionnaire files' object store, key-encryption key
// and scan queue, with a conversation as the files' parent; no holder (the chat decides and audits each call).
func newChatFiles(pool *pgxpool.Pool, files config.ObjectStore, kek *envelope.KEK, jobs *river.Client[pgx.Tx], tx *postgres.Transactor) *binary.Service {
	store := binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, files.Endpoint, files.Region, files.Bucket, files.AccessKeyID,
		files.SecretAccessKey, s3PageSize)
	return binary.NewService(binary.Ports{
		Repo: binary.NewPostgresRepository(pool), Store: store, Tx: tx, Jobs: binary.NewRiverJobs(jobs), Clock: clock.System{}, KEK: kek,
	})
}
