package transcript_test

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// add appends the envelope to the encounter's stream, as a producer does, and returns its id.
func add(t *testing.T, rdb *goredis.Client, encounter string, envelope []byte) string {
	t.Helper()
	id, err := rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(encounter), Values: []any{transcript.StreamField, envelope}}).Result()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func envelope(n int) []byte {
	return []byte(`{"type":"transcript","version":1,"payload":{"segment":` + strconv.Itoa(n) + `}}`)
}

// Read gives the entries strictly after the cursor, oldest first, each with its envelope's bytes as they were
// added.
func TestRedisStream_ReadGivesTheEntriesAfterTheCursorWithTheirBytes(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	var ids []string
	for i := range 3 {
		ids = append(ids, add(t, rdb, encounterID, envelope(i)))
	}
	s := transcript.NewRedisStream(rdb)
	for _, tt := range []struct {
		after string
		from  int
	}{{after: "0", from: 0}, {after: ids[0], from: 1}, {after: ids[2], from: 3}} {
		got, err := s.Read(t.Context(), encounterID, tt.after, 10*time.Millisecond)
		if err != nil || len(got) != 3-tt.from {
			t.Fatalf("Read(after %s) = %d entries, %v; want %d", tt.after, len(got), err, 3-tt.from)
		}
		for i, e := range got {
			if e.ID != ids[tt.from+i] || !bytes.Equal(e.Envelope, envelope(tt.from+i)) {
				t.Errorf("Read(after %s)[%d] = %s %s, want %s %s", tt.after, i, e.ID, e.Envelope, ids[tt.from+i], envelope(tt.from+i))
			}
		}
	}
}

// With nothing after the cursor (or no stream at all) Read waits, and gives an entry added meanwhile at once.
func TestRedisStream_ReadWaitsForTheNextEntry(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	s := transcript.NewRedisStream(rdb)

	began := time.Now()
	got, err := s.Read(t.Context(), encounterID, "0", 200*time.Millisecond)
	if err != nil || len(got) != 0 || time.Since(began) < 150*time.Millisecond {
		t.Errorf("Read(no stream) = %v, %v after %v; want none after the wait", got, err, time.Since(began))
	}

	done := make(chan []transcript.Entry, 1)
	go func() {
		got, _ := s.Read(t.Context(), encounterID, "0", 5*time.Second)
		done <- got
	}()
	time.Sleep(100 * time.Millisecond) // let the read block; it is given the entry whenever it starts
	id := add(t, rdb, encounterID, envelope(7))
	select {
	case got := <-done:
		if len(got) != 1 || got[0].ID != id || !bytes.Equal(got[0].Envelope, envelope(7)) {
			t.Errorf("Read() = %+v, want the entry added", got)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Read() did not return the entry added while it waited")
	}
}

// A page holds at most 100 entries; reading on from its last id gives the rest.
func TestRedisStream_ReadGivesAPageAtATime(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	for i := range 150 {
		add(t, rdb, encounterID, envelope(i))
	}
	s := transcript.NewRedisStream(rdb)
	first, err := s.Read(t.Context(), encounterID, "0", 10*time.Millisecond)
	if err != nil || len(first) != 100 {
		t.Fatalf("first page = %d entries, %v; want 100", len(first), err)
	}
	rest, err := s.Read(t.Context(), encounterID, first[99].ID, 10*time.Millisecond)
	if err != nil || len(rest) != 50 || !bytes.Equal(rest[0].Envelope, envelope(100)) {
		t.Errorf("second page = %d entries, %v; want 50 from the 101st", len(rest), err)
	}
}

// An entry without the envelope's field (no producer of ours writes one) is given without an envelope.
func TestRedisStream_ReadGivesAnEntryWithoutTheFieldWithoutAnEnvelope(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	id, err := rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(encounterID), Values: []any{"other", "x"}}).Result()
	if err != nil {
		t.Fatal(err)
	}
	got, err := transcript.NewRedisStream(rdb).Read(t.Context(), encounterID, "0", 10*time.Millisecond)
	if err != nil || len(got) != 1 || got[0].ID != id || got[0].Envelope != nil {
		t.Errorf("Read() = %+v, %v; want the entry without an envelope", got, err)
	}
}

func TestRedisStream_ReadFailsWhenRedisFails(t *testing.T) {
	t.Parallel()
	rdb := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { _ = rdb.Close() })
	if _, err := transcript.NewRedisStream(rdb).Read(t.Context(), encounterID, "0", 10*time.Millisecond); err == nil {
		t.Error("Read() error = nil, want the connection's")
	}
}
