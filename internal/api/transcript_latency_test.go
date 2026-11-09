//go:build loadtest

package api_test

import (
	"slices"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/transcript"
)

// The gateway's latency (SAL-6, docs/plans/consult-intelligence.PROGRESS.md): from an entry's XADD to its
// envelope's arrival at the practitioner's WebSocket, through api.New on the compose Postgres and Redis, with
// the access check after every read. 500 entries one at a time, then a 2,000-entry replay. Not part of make test.
func TestTranscriptStreamLatency(t *testing.T) {
	s := newStreamApp(t)
	conn, _, err := s.dialEvents(t, s.d.practitioner, "")
	if err != nil {
		t.Fatal(err)
	}
	var took []time.Duration
	for n := range 500 {
		began := wallNow()
		s.add(t, n)
		if _, err := next(t, conn); err != nil {
			t.Fatal(err)
		}
		took = append(took, wallNow().Sub(began))
	}
	slices.Sort(took)
	q := func(p float64) time.Duration { return took[int(p*float64(len(took)-1))] }
	t.Logf("live, XADD to frame, 500 entries: p50 %v, p95 %v, p99 %v, max %v", q(0.5), q(0.95), q(0.99), took[len(took)-1])

	pipe := s.rdb.Pipeline()
	env := s.add(t, 9999)
	for range 1999 {
		pipe.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(s.encounter), Values: []any{transcript.StreamField, env}})
	}
	if _, err := pipe.Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	began := wallNow()
	conn, _, err = s.dialEvents(t, s.d.practitioner, "0")
	if err != nil {
		t.Fatal(err)
	}
	for range 2500 {
		if _, err := next(t, conn); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("replay of 2,500 entries from a reconnect: %v", wallNow().Sub(began))
}
