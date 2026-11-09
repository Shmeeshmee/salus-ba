package worker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// scriptedSuggester answers its n-th call with the n-th script, nothing once they are used up.
type scriptedSuggester struct {
	script [][]llm.Suggestion
	calls  int
	mu     sync.Mutex
}

func (s *scriptedSuggester) Suggest(_ context.Context, req llm.SuggestRequest) ([]llm.Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls > len(s.script) {
		return nil, nil
	}
	return s.script[s.calls-1], nil
}

// line is the id of the n-th final line said.
func line(n int) string { return fmt.Sprintf("7d1e2f30-1a2b-4c3d-8e4f-%012d", n) }

// The suggestions worker in cmd/worker, on the real Postgres and Redis: five final lines on the encounter's
// stream bring a suggestion, stored and published on the stream labelled as AI; once the practitioner has
// dismissed it, the model's repeating it shows nothing, while a new one is shown. Nothing said or suggested
// reaches the log.
func TestSuggestions_ShowsWhatIsNewAndNeverWhatWasDismissed(t *testing.T) {
	c := newConsultStream(t)
	model := &scriptedSuggester{script: [][]llm.Suggestion{
		{{Kind: "question", Text: suggested, Confidence: 0.6, SourceIDs: []string{line(1)}}},
		{
			{Kind: "question", Text: suggested, Confidence: 0.9, SourceIDs: []string{line(6)}},
			{Kind: "red_flag", Text: "Placeholder red flag to check", Confidence: 0.5, SourceIDs: []string{line(7)}},
		},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.NewSuggestions(c.db.app, c.rdb, model, slog.New(slog.NewJSONHandler(c.log, nil))).Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	say := func(from, to int) {
		for n := from; n <= to; n++ {
			e := c.segment(line(n), true)
			c.add(t, encode(t, transcript.MessageType, e))
		}
	}
	say(1, suggestion.RunEveryLines)
	c.waitRows(t, 1, "SELECT count(*) FROM suggestion WHERE encounter_id = $1 AND text = $2 AND status = 'shown'", c.encounter, suggested)
	if got := c.published(t); len(got) != 1 || got[0].Text != suggested || got[0].Origin != suggestion.OriginAI {
		t.Fatalf("published %+v, want the suggestion, labelled as AI", got)
	}
	if _, err := c.db.owner.Exec(t.Context(), "UPDATE suggestion SET status = 'dismissed' WHERE encounter_id = $1", c.encounter); err != nil {
		t.Fatal(err)
	}
	time.Sleep(suggestion.MinGap) // the next run waits as long after the first
	say(suggestion.RunEveryLines+1, 2*suggestion.RunEveryLines)
	c.waitRows(t, 1, "SELECT count(*) FROM suggestion WHERE encounter_id = $1 AND type = 'red_flag'", c.encounter)
	c.waitRows(t, 2, "SELECT count(*) FROM suggestion WHERE encounter_id = $1", c.encounter)
	if got := c.published(t); len(got) != 2 || got[1].Type != suggestion.TypeRedFlag {
		t.Errorf("published %+v, want the red flag only after the first", got)
	}
	c.waitRows(t, 2, `SELECT count(*) FROM audit_event WHERE agent_name = 'suggestions_worker' AND action = 'C' AND outcome = 0`)
	for _, s := range []string{spoken, suggested} {
		if strings.Contains(c.log.String(), s) {
			t.Errorf("the log holds %q:\n%s", s, c.log.String())
		}
	}
}

// published is the suggestion events on the encounter's stream, in order.
func (c consultStream) published(t *testing.T) []suggestion.Event {
	t.Helper()
	msgs, err := c.rdb.XRange(t.Context(), transcript.StreamKey(c.encounter), "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	var out []suggestion.Event
	for _, m := range msgs {
		raw, _ := m.Values[transcript.StreamField].(string)
		env, err := wire.Decode([]byte(raw))
		if err != nil || env.Type != suggestion.MessageType {
			continue
		}
		var e suggestion.Event
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}
