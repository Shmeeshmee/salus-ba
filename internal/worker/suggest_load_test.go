//go:build loadtest

package worker_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// freshSuggester answers every call with three new suggestions citing the window's last lines, at no cost of
// its own: what is timed is the worker's.
type freshSuggester struct {
	n  int
	mu sync.Mutex
}

func (s *freshSuggester) Suggest(_ context.Context, req llm.SuggestRequest) ([]llm.Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []llm.Suggestion
	for _, kind := range []string{"question", "red_flag", "term", "diagnosis", "question", "term"} {
		s.n++
		out = append(out, llm.Suggestion{Kind: kind, Text: fmt.Sprintf("Placeholder suggestion number %d", s.n), Confidence: 0.5,
			SourceIDs: []string{req.Transcript[len(req.Transcript)-1].ID}})
	}
	return out, nil
}

// The worker's own cost of one window (make bench-suggest): from reading the stream to the last suggestion
// published, on the compose Postgres and Redis, the model answering at once with six suggestions of which
// three are shown. The stream holds a consultation's worth: 30 final lines, each after ten partials; the
// encounter's suggestions shown so far grow by three a run, up to 3 x runs.
func TestSuggestionWindowCost(t *testing.T) {
	c := newConsultStream(t)
	for n := 1; n <= suggestion.WindowLines; n++ {
		for range 10 {
			c.add(t, encode(t, transcript.MessageType, c.segment(line(n), false)))
		}
		c.add(t, encode(t, transcript.MessageType, c.segment(line(n), true)))
	}
	pool := c.db.app
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	svc := suggestion.NewService(suggestion.Ports{
		Repo:       suggestion.NewPostgresRepository(pool),
		Encounters: encounter.NewService(encounter.Ports{Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}}),
		Consents:   consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clock.System{}, nil),
		Publisher:  suggestion.NewRedisPublisher(c.rdb), Audit: audit, Tx: tx,
	})
	a := suggestion.NewAssistant(svc, suggestion.AssistPorts{
		Suggester: &freshSuggester{}, Live: suggestion.NewRedisLive(c.rdb),
		Clinics: organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{}),
		Clock:   clock.System{}, Log: slog.New(slog.DiscardHandler),
	})
	const runs = 200
	took := make([]time.Duration, 0, runs)
	for range runs {
		s := a.NewSession(c.encounter) // a fresh session reads the whole stream and runs at once
		start := wallNow()
		if err := a.Step(t.Context(), s); err != nil {
			t.Fatal(err)
		}
		took = append(took, wallNow().Sub(start))
	}
	slices.Sort(took)
	pct := func(p int) time.Duration { return took[(len(took)-1)*p/100] }
	t.Logf("one window, %d runs: p50 %v, p95 %v, p99 %v, max %v", runs, pct(50), pct(95), pct(99), took[len(took)-1])
	c.waitRows(t, 3*runs, "SELECT count(*) FROM suggestion WHERE encounter_id = $1", c.encounter)
}
