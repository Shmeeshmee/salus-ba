package worker

import (
	"context"
	"crypto/rand"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// Suggestions is the live suggestions worker (docs/plans/consult-intelligence.md, SAL-7) in cmd/worker, beside
// the stream consumer: it follows every consultation's stream, sends its recent final lines to the model and
// shows the practitioner what is new (suggestion.Assistant). The worker processes share the encounters, one
// each at a time, by a lease in Redis.
type Suggestions struct {
	assistant *suggestion.Assistant
	owner     string
}

// NewSuggestions builds the worker on pool (salus_app), rdb and the model, as a member of a name of its own.
func NewSuggestions(pool *pgxpool.Pool, rdb *goredis.Client, model llm.Suggester, log *slog.Logger) *Suggestions {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	// Only Lookup is asked of the encounters, only State of the consents, only DefaultLanguage of the clinics.
	encounters := encounter.NewService(encounter.Ports{Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}})
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clock.System{}, nil)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{})
	// No Records: the worker sets no practitioner's status.
	svc := suggestion.NewService(suggestion.Ports{
		Repo: suggestion.NewPostgresRepository(pool), Encounters: encounters, Consents: consents, Publisher: suggestion.NewRedisPublisher(rdb), Audit: audit, Tx: tx,
	})
	return &Suggestions{
		assistant: suggestion.NewAssistant(svc, suggestion.AssistPorts{
			Suggester: model, Live: suggestion.NewRedisLive(rdb), Clinics: clinics, Clock: clock.System{}, Log: log,
		}),
		owner: "suggestions-" + strings.ToLower(rand.Text()),
	}
}

// Run follows the streams until ctx ends.
func (s *Suggestions) Run(ctx context.Context) {
	s.assistant.Run(ctx, s.owner)
}
