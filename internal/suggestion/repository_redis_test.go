package suggestion_test

import (
	"bytes"
	"slices"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// Publish appends the envelope under the stream's one field, capped and expiring as every producer's entries.
func TestRedisPublisher_AppendsTheEnvelopeToTheEncountersStream(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	env := []byte(`{"type":"suggestion","version":1,"payload":{}}`)
	if err := suggestion.NewRedisPublisher(rdb).Publish(t.Context(), encounterID, env); err != nil {
		t.Fatal(err)
	}
	key := transcript.StreamKey(encounterID)
	msgs, err := rdb.XRange(t.Context(), key, "-", "+").Result()
	if err != nil || len(msgs) != 1 || msgs[0].Values[transcript.StreamField] != string(env) {
		t.Fatalf("stream = %+v, %v; want the envelope", msgs, err)
	}
	if ttl := rdb.TTL(t.Context(), key).Val(); ttl <= transcript.StreamTTL-time.Minute || ttl > transcript.StreamTTL {
		t.Errorf("TTL %v, want %v", ttl, transcript.StreamTTL)
	}
}

func TestRedisPublisher_ReturnsRedisErrors(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	if err := rdb.Set(t.Context(), transcript.StreamKey(encounterID), "not a stream", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := suggestion.NewRedisPublisher(rdb).Publish(t.Context(), encounterID, []byte("{}")); err == nil {
		t.Error("Publish() onto a string succeeded, want WRONGTYPE")
	}
}

// Encounters names the encounters with a stream, and nothing else matching the pattern.
func TestRedisLive_EncountersAreThoseWithAStream(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	other := "0b6c3a52-6a0c-4c52-9e1b-000000000002"
	for _, id := range []string{encounterID, other} {
		rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(id), Values: []any{transcript.StreamField, "{}"}})
	}
	rdb.Set(t.Context(), transcript.StreamKey(unknownID), "a string", 0)
	got, err := suggestion.NewRedisLive(rdb).Encounters(t.Context())
	slices.Sort(got)
	if want := []string{other, encounterID}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Encounters() = %v, %v; want %v", got, err, want)
	}
}

// Entries gives the entries after the cursor ("" for the first), at most count, with their envelopes' bytes.
func TestRedisLive_EntriesAfterTheCursor(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	live := suggestion.NewRedisLive(rdb)
	if got, err := live.Entries(t.Context(), encounterID, "", 10); err != nil || len(got) != 0 {
		t.Fatalf("Entries() of no stream = %v, %v; want none", got, err)
	}
	var ids []string
	for _, b := range []string{"a", "b", "c"} {
		ids = append(ids, rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(encounterID), Values: []any{transcript.StreamField, b}}).Val())
	}
	rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(encounterID), Values: []any{"other", "x"}})
	got, err := live.Entries(t.Context(), encounterID, "", 2)
	if err != nil || len(got) != 2 || got[0].ID != ids[0] || !bytes.Equal(got[1].Envelope, []byte("b")) {
		t.Errorf("Entries(first 2) = %+v, %v", got, err)
	}
	got, err = live.Entries(t.Context(), encounterID, ids[1], 10)
	if err != nil || len(got) != 2 || got[0].ID != ids[2] || got[1].Envelope != nil {
		t.Errorf("Entries(after the second) = %+v, %v; want the third and one without an envelope", got, err)
	}
}

// One owner holds an encounter's lease at a time, renews it, and loses it once it expires.
func TestRedisLive_ClaimHoldsOneOwnerAtATime(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	live := suggestion.NewRedisLive(rdb)
	claim := func(owner string, ttl time.Duration) bool {
		t.Helper()
		held, err := live.Claim(t.Context(), encounterID, owner, ttl)
		if err != nil {
			t.Fatal(err)
		}
		return held
	}
	if !claim("a", 100*time.Millisecond) || claim("b", time.Second) || !claim("a", 100*time.Millisecond) {
		t.Fatal("a does not hold the lease alone")
	}
	time.Sleep(150 * time.Millisecond)
	if !claim("b", time.Second) || claim("a", time.Second) {
		t.Error("b does not take the expired lease")
	}
}
