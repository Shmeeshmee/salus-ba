package transcript_test

import (
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// Each envelope is an entry of the encounter's stream, under the envelope field, unchanged; the stream expires
// half an hour after its last entry.
func TestRedisPublisher_AppendsTheEnvelopeToTheEncountersStream(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	p := transcript.NewRedisPublisher(rdb)
	for _, env := range []string{`{"type":"transcript","version":1,"payload":{"n":1}}`, `{"type":"transcript","version":1,"payload":{"n":2}}`} {
		if err := p.Publish(t.Context(), encounterID, []byte(env)); err != nil {
			t.Fatalf("Publish() = %v", err)
		}
	}
	key := transcript.StreamKey(encounterID)
	entries, err := rdb.XRange(t.Context(), key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Values["envelope"] != `{"type":"transcript","version":1,"payload":{"n":1}}` || len(entries[1].Values) != 1 {
		t.Errorf("entries %+v; want both envelopes, in order, under envelope alone", entries)
	}
	ttl, err := rdb.TTL(t.Context(), key).Result()
	if err != nil || ttl <= 29*time.Minute || ttl > 30*time.Minute {
		t.Errorf("TTL = %v, %v; want about 30 minutes", ttl, err)
	}
}

// The stream is capped near StreamMaxLen entries (XADD MAXLEN ~): the oldest go first.
func TestRedisPublisher_CapsTheStream(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	p := transcript.NewRedisPublisher(rdb)
	key := transcript.StreamKey(encounterID)
	pipe := rdb.Pipeline()
	for range transcript.StreamMaxLen + 500 {
		pipe.XAdd(t.Context(), &goredis.XAddArgs{Stream: key, Values: []any{"envelope", "{}"}})
	}
	if _, err := pipe.Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := p.Publish(t.Context(), encounterID, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	n, err := rdb.XLen(t.Context(), key).Result()
	if err != nil || n > transcript.StreamMaxLen+200 || n < transcript.StreamMaxLen {
		t.Errorf("XLEN = %d, %v; want about %d", n, err, transcript.StreamMaxLen)
	}
}
