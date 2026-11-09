package worker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The stream consumer's settings (docs/plans/consult-intelligence.md, SAL-6).
const (
	// consultAgent names the consumer as the agent of its writes' audit events, and its consumer group.
	consultAgent = "consult_stream"
	// consultBlock is how long a read waits, and how often new streams are looked for: a consultation's first
	// events are stored within about this long of its stream's creation (the live view reads the stream
	// itself).
	consultBlock = time.Second
	// consultMinIdle is how long an entry stays pending (its write failed, or its worker died) before it is
	// written again.
	consultMinIdle = 30 * time.Second
)

// ConsultStream stores what the consultations' Redis Streams carry into Postgres, each table through its own
// domain: final transcript segments (transcript) and suggestions (suggestion). Drafts are the drafting's
// (SAL-8). It is a member of one consumer group over every stream (consult:*:events), so the worker processes
// share the streams and an entry is written at least once; the writes are idempotent, so at most one row.
// Redis is the transport, Postgres the record (ADR 0005). It logs entry ids, types and reasons, never a payload.
type ConsultStream struct {
	group       *redis.Group
	transcripts *transcript.Service
	suggestions *suggestion.Service
	log         *slog.Logger
}

// NewConsultStream builds the consumer on pool (salus_app) and rdb, as a member of a name of its own.
func NewConsultStream(pool *pgxpool.Pool, rdb *goredis.Client, log *slog.Logger) *ConsultStream {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	// Only Lookup is asked of the encounters, and only State of the consents: neither needs more.
	encounters := encounter.NewService(encounter.Ports{Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}})
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clock.System{}, nil)
	c := &ConsultStream{
		transcripts: transcript.NewService(transcript.Ports{
			Repo: transcript.NewPostgresRepository(pool), Encounters: encounters, Consents: consents, Audit: audit, Tx: tx,
		}),
		suggestions: suggestion.NewService(suggestion.Ports{
			Repo: suggestion.NewPostgresRepository(pool), Encounters: encounters, Consents: consents, Audit: audit, Tx: tx,
		}),
		log: log,
	}
	c.group = &redis.Group{
		Client: rdb, Clock: clock.System{}, Pattern: transcript.StreamKey("*"), Name: consultAgent, Consumer: consultAgent + "-" + strings.ToLower(rand.Text()),
		Block: consultBlock, MinIdle: consultMinIdle,
		OnError: func(err error) { log.Error("consult stream failed", "error", err) },
	}
	return c
}

// Run consumes the streams until ctx ends.
func (c *ConsultStream) Run(ctx context.Context) {
	c.group.Run(ctx, c.handle)
}

// handle stores one entry. A write that failed is returned, so the entry is written again later; an entry
// that can never be stored (not an envelope, an unknown type or version, an invalid event, one for another
// encounter than its stream's, an encounter not found, a revoked consent) is dropped: logged and acknowledged.
func (c *ConsultStream) handle(ctx context.Context, stream, id string, values map[string]any) error {
	raw, _ := values[transcript.StreamField].(string)
	env, err := wire.Decode([]byte(raw))
	if err != nil {
		c.drop(ctx, stream, id, "", "not an envelope")
		return nil
	}
	switch {
	case env.Type == transcript.MessageType && env.Version == transcript.MessageVersion:
		var e transcript.Event
		if json.Unmarshal(env.Payload, &e) != nil {
			err = transcript.ErrInvalid
		} else if err = sameEncounter(stream, e.EncounterID); err == nil {
			err = c.transcripts.Store(ctx, consultAgent, e)
		}
	case env.Type == suggestion.MessageType && env.Version == suggestion.MessageVersion:
		var e suggestion.Event
		if json.Unmarshal(env.Payload, &e) != nil {
			err = suggestion.ErrInvalid
		} else if err = sameEncounter(stream, e.EncounterID); err == nil {
			err = c.suggestions.Record(ctx, consultAgent, e)
		}
	case env.Type == "draft": // the drafting's own (SAL-8)
		return nil
	default:
		c.drop(ctx, stream, id, env.Type, "an unknown type or version")
		return nil
	}
	if reason := dropped(err); reason != "" {
		c.drop(ctx, stream, id, env.Type, reason)
		return nil
	}
	return err
}

// errOtherEncounter is an event on another encounter's stream.
var errOtherEncounter = errors.New("worker: an event of another encounter")

// sameEncounter refuses an event whose encounter is not the stream's.
func sameEncounter(stream, encounterID string) error {
	if stream != transcript.StreamKey(encounterID) {
		return errOtherEncounter
	}
	return nil
}

// dropped names why err means the entry can never be stored, or "" when it can be written again.
func dropped(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, transcript.ErrInvalid), errors.Is(err, suggestion.ErrInvalid):
		return "an invalid event"
	case errors.Is(err, errOtherEncounter):
		return "an event of another encounter"
	case errors.Is(err, transcript.ErrNotFound), errors.Is(err, suggestion.ErrNotFound):
		return "the encounter not found"
	case errors.Is(err, transcript.ErrConsentRequired), errors.Is(err, suggestion.ErrConsentRequired):
		return "the consent not active"
	}
	return ""
}

// drop logs an entry that is not stored: the stream's key (an opaque encounter id), the entry's id, its type
// and why, nothing of its payload.
func (c *ConsultStream) drop(ctx context.Context, stream, id, typ, reason string) {
	c.log.InfoContext(ctx, "consult event not stored", "stream", stream, "entry_id", id, "type", typ, "reason", reason)
}
