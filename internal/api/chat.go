package api

import (
	"context"
	"log/slog"
	"net/http"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
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
	caller     func(ctx context.Context) (appointment.Caller, bool, error)
	active     func(r *http.Request) (bool, error) // the socket's session is still open (auth.Handler.Active)
	summarizer llm.Summarizer                      // the AI summary's engine (config.LoadChatSummary: the fake)
}

// newChat builds the chat (docs/plans/chat.md): its routes, its socket, and the hub that carries the live frames
// over one Redis connection, which the App closes.
func newChat(p chatPorts, log *slog.Logger) (routes, stream func(*http.ServeMux), hub *redis.Hub) {
	hub = redis.NewHub(context.Background(), p.rdb, chatHubBuffer)
	svc := communication.NewService(communication.Ports{
		Repo: communication.NewPostgresRepository(p.db), Publisher: hub, Consents: p.consents, Visits: p.visits,
		Roles: p.roles, Encounters: p.encounters, Audit: p.audit, Tx: p.tx, Clock: clock.System{}, Summarizer: p.summarizer, Log: log,
	})
	h := communication.NewHTTPHandler(svc, hub, p.caller, p.active, clock.System{}, communication.CheckEvery, log)
	return h.Routes, h.StreamRoutes, hub
}
