package worker_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The synthetic text of the consultation's events, which no log line may hold.
const (
	spoken    = "Placeholder words the patient said"
	suggested = "Placeholder question to ask"
)

// consultStream is a consultation in progress (the past appointment of seedBooking, its practitioner joined)
// and the stream consumer running on its database and a Redis of its own.
type consultStream struct {
	db        logins
	rdb       *goredis.Client
	log       *syncBuffer
	encounter string
	patient   string
}

func newConsultStream(t *testing.T) consultStream {
	t.Helper()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	seedConsultation(t, db, b, true, false)
	c := consultStream{db: db, rdb: redistest.NewClient(t), log: &syncBuffer{}, patient: b.patient}
	if err := db.owner.QueryRow(t.Context(), "SELECT id::text FROM encounter WHERE appointment_id = $1", b.past).Scan(&c.encounter); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.NewConsultStream(db.app, c.rdb, slog.New(slog.NewJSONHandler(c.log, nil))).Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return c
}

// add puts the envelope on the encounter's stream (or, with key, another stream), as the producers do.
func (c consultStream) add(t *testing.T, env []byte, key ...string) {
	t.Helper()
	stream := transcript.StreamKey(c.encounter)
	if len(key) > 0 {
		stream = key[0]
	}
	if err := c.rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: stream, Values: []any{transcript.StreamField, env}}).Err(); err != nil {
		t.Fatal(err)
	}
}

func encode(t *testing.T, typ string, payload any) []byte {
	t.Helper()
	env, err := wire.Encode(typ, 1, payload)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func (c consultStream) segment(id string, final bool) transcript.Event {
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	return transcript.Event{
		EncounterID: c.encounter, SegmentID: id, Speaker: transcript.SpeakerPatient, Lang: "bs", Original: spoken,
		Translation: "[de] " + spoken, IsFinal: final, TsStart: at, TsEnd: at.Add(time.Second),
	}
}

func (c consultStream) suggestion(status suggestion.Status) suggestion.Event {
	return suggestion.Event{
		ID: "5a000000-0000-4000-8000-000000000001", EncounterID: c.encounter, Type: suggestion.TypeQuestion, Text: suggested,
		Status: status, Origin: suggestion.OriginAI, SourceSegmentIDs: []string{segmentA}, Confidence: 0.5,
	}
}

const (
	segmentA = "7d1e2f30-1a2b-4c3d-8e4f-00000000000a"
	segmentB = "7d1e2f30-1a2b-4c3d-8e4f-00000000000b"
)

// waitRows waits until the query counts want, for at most ten seconds.
func (c consultStream) waitRows(t *testing.T, want int, q string, args ...any) {
	t.Helper()
	deadline := wallNow().Add(10 * time.Second)
	var n int
	for {
		if err := c.db.owner.QueryRow(t.Context(), q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == want || wallNow().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if n != want {
		t.Errorf("%s = %d, want %d", q, n, want)
	}
}

// waitDrained waits until the group has acknowledged every entry of the encounter's stream.
func (c consultStream) waitDrained(t *testing.T, entries int64) {
	t.Helper()
	key := transcript.StreamKey(c.encounter)
	deadline := wallNow().Add(10 * time.Second)
	for wallNow().Before(deadline) {
		groups, err := c.rdb.XInfoGroups(t.Context(), key).Result()
		if err == nil && len(groups) == 1 && groups[0].EntriesRead == entries && groups[0].Pending == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the stream's %d entries were not all read and acknowledged", entries)
}

// What the stream carries is stored, each table by its own domain: final segments once (partials stay on the
// stream, a re-delivered final is no second row), suggestions once, as first shown (a later event's status was
// stored by whoever changed it, SAL-7, so the consumer changes nothing); drafts are SAL-8's and
// left alone. An entry that is no envelope, an unknown type or version, an invalid event and one naming another
// encounter than its stream's are acknowledged and dropped. Every write is audited as the consumer's, and the
// log holds nothing that was said or suggested.
func TestConsultStream_StoresTheFinalSegmentsAndTheSuggestions(t *testing.T) {
	c := newConsultStream(t)
	other := c.segment(segmentB, true)
	other.EncounterID = "0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5d"
	invalid := c.segment(segmentB, true)
	invalid.Speaker = "doctor"
	for _, env := range [][]byte{
		encode(t, transcript.MessageType, c.segment(segmentA, false)),
		encode(t, transcript.MessageType, c.segment(segmentA, true)),
		encode(t, transcript.MessageType, c.segment(segmentA, true)), // delivered again
		encode(t, suggestion.MessageType, c.suggestion(suggestion.StatusShown)),
		encode(t, suggestion.MessageType, c.suggestion(suggestion.StatusPinned)),
		encode(t, "draft", composition.Event{DraftID: segmentB, EncounterID: c.encounter}),
		[]byte("not an envelope"),
		[]byte(`{"type":"transcript","version":2,"payload":{}}`),
		[]byte(`{"type":"unknown","version":1,"payload":{}}`),
		[]byte(`{"type":"transcript","version":1,"payload":"a string"}`),
		encode(t, transcript.MessageType, invalid),
		encode(t, transcript.MessageType, other),
	} {
		c.add(t, env)
	}
	c.waitDrained(t, 12)

	c.waitRows(t, 1, "SELECT count(*) FROM transcript_segment WHERE encounter_id = $1", c.encounter)
	c.waitRows(t, 1, "SELECT count(*) FROM transcript_segment WHERE id = $1 AND original = $2", segmentA, spoken)
	c.waitRows(t, 1, "SELECT count(*) FROM suggestion WHERE encounter_id = $1 AND status = 'shown' AND version_id = 1", c.encounter)
	c.waitRows(t, 0, "SELECT count(*) FROM document_draft")
	c.waitRows(t, 2, `SELECT count(*) FROM audit_event WHERE agent_name = 'consult_stream' AND outcome = 0 AND entity_reference = $1
		AND entity_name IN ('transcript_segment', 'suggestion')`, "Encounter/"+c.encounter)
	for _, s := range []string{spoken, suggested} {
		if strings.Contains(c.log.String(), s) {
			t.Errorf("the log holds %q:\n%s", s, c.log.String())
		}
	}
}

// A stream that appears while the consumer runs is found and read from its first entry.
func TestConsultStream_FindsAStreamThatAppearsLater(t *testing.T) {
	c := newConsultStream(t)
	time.Sleep(50 * time.Millisecond) // the consumer has looked once and found nothing
	c.add(t, encode(t, transcript.MessageType, c.segment(segmentA, true)))
	c.add(t, encode(t, transcript.MessageType, c.segment(segmentB, true)))
	c.waitRows(t, 2, "SELECT count(*) FROM transcript_segment WHERE encounter_id = $1", c.encounter)
}

// After a revocation nothing more is stored, also for the consultation that started before it.
func TestConsultStream_StoresNothingAfterARevocation(t *testing.T) {
	c := newConsultStream(t)
	if _, err := c.db.owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1`, c.patient); err != nil {
		t.Fatal(err)
	}
	c.add(t, encode(t, transcript.MessageType, c.segment(segmentA, true)))
	c.add(t, encode(t, suggestion.MessageType, c.suggestion(suggestion.StatusShown)))
	c.waitDrained(t, 2)
	c.waitRows(t, 0, "SELECT count(*) FROM transcript_segment")
	c.waitRows(t, 0, "SELECT count(*) FROM suggestion")
	c.waitRows(t, 2, `SELECT count(*) FROM audit_event WHERE agent_name = 'consult_stream' AND outcome = 4`)
}

// wallNow is the real time, for the waits of a test against real servers.
func wallNow() time.Time { return clock.System{}.Now() }
