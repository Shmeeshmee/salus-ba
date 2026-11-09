package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The consultation's event stream through api.New (SAL-6): the encounter's Redis Stream forwarded over a
// WebSocket to the encounter's practitioner only, resumed after the last cursor with no gap and no duplicate,
// and ended at once by a revocation; and the stored transcript's export.

// streamApp is a consultation in progress: the practitioner joined (the encounter started) five minutes after
// the appointment's start, so a revocation leaves the appointment past and shown at the name level.
type streamApp struct {
	consultationApp
	srv       *httptest.Server
	rdb       *goredis.Client
	encounter string
}

func newStreamApp(t *testing.T) streamApp {
	t.Helper()
	c := newConsultationApp(t)
	now := clock.System{}.Now()
	appt := c.book(t, now.Add(-5*time.Minute))
	expectHTTP(t, c.d.practitioner, http.MethodPost, "/appointments/"+appt+"/consultation/join", http.StatusOK, `"token"`)
	c.joined(t, appt, "practitioner", now.Add(-time.Minute).UTC().Truncate(time.Second))
	s := streamApp{consultationApp: c, srv: httptest.NewServer(c.h)}
	t.Cleanup(s.srv.Close)
	if err := c.owner.QueryRow(t.Context(), "SELECT id::text FROM encounter WHERE appointment_id = $1", appt).Scan(&s.encounter); err != nil {
		t.Fatal(err)
	}
	opts, err := goredis.ParseURL(c.redisURL)
	if err != nil {
		t.Fatal(err)
	}
	s.rdb = goredis.NewClient(opts)
	t.Cleanup(func() { _ = s.rdb.Close() })
	return s
}

// add puts the n-th synthetic event on the encounter's stream, as the agent and the suggestions do: a
// transcript segment, or every third one a suggestion citing it. It returns the envelope.
func (s streamApp) add(t *testing.T, n int) []byte {
	t.Helper()
	segment := "7d1e2f30-1a2b-4c3d-8e4f-" + strings.Repeat("0", 12-len(itoa(n))) + itoa(n)
	var env []byte
	var err error
	if n%3 == 2 {
		env, err = wire.Encode(suggestion.MessageType, suggestion.MessageVersion, suggestion.Event{
			ID: "5a000000-0000-4000-8000-" + segment[24:], EncounterID: s.encounter, Type: suggestion.TypeQuestion,
			Text: "Placeholder question " + itoa(n), Status: suggestion.StatusShown, SourceSegmentIDs: []string{segment}, Confidence: 0.5,
		})
	} else {
		start := time.Date(2030, 1, 7, 9, 0, n, 0, time.UTC)
		env, err = wire.Encode(transcript.MessageType, transcript.MessageVersion, transcript.Event{
			EncounterID: s.encounter, SegmentID: segment, Speaker: transcript.SpeakerPatient, Lang: "bs",
			Original: "Placeholder original " + itoa(n), IsFinal: n%2 == 0, TsStart: start, TsEnd: start.Add(time.Second),
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := s.rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: transcript.StreamKey(s.encounter), Values: []any{transcript.StreamField, env}}).Err(); err != nil {
		t.Fatal(err)
	}
	return env
}

func itoa(n int) string { return strconv.Itoa(n) }

// dialEvents opens the encounter's event stream as cl, from the API's own origin, after the cursor.
func (s streamApp) dialEvents(t *testing.T, cl *client, after string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(s.srv.URL, "http") + "/encounters/" + s.encounter + "/events"
	if after != "" {
		url += "?after=" + after
	}
	h := http.Header{"Origin": {s.srv.URL}, "Authorization": {"Bearer " + cl.bearer}}
	conn, res, err := websocket.DefaultDialer.DialContext(t.Context(), url, h)
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
	}
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	return conn, res, err
}

// frame is one forwarded entry: its cursor and its envelope.
type frame struct {
	cursor   string
	envelope []byte
}

// next reads the next forwarded entry: a cursor frame, then the envelope it announces.
func next(t *testing.T, conn *websocket.Conn) (frame, error) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, c, err := conn.ReadMessage()
	if err != nil {
		return frame{}, err
	}
	var cur struct {
		Type    string
		Payload transcript.Cursor
		Version int
	}
	if err := json.Unmarshal(c, &cur); err != nil || cur.Type != transcript.CursorType || cur.Version != 1 {
		t.Fatalf("frame %s, want a cursor (%v)", c, err)
	}
	_, env, err := conn.ReadMessage()
	return frame{cursor: cur.Payload.ID, envelope: env}, err
}

// A practitioner who loses the connection resumes after the last cursor they received: they get every entry
// they missed, in order, then the live ones, each exactly once, also while entries keep arriving during the
// replay (more than a page of them). The envelopes arrive as the producers wrote them. Each connection is one
// audited subscription.
func TestNew_ResumesTheEventStreamAfterTheLastCursorWithNoGapOrDuplicate(t *testing.T) {
	s := newStreamApp(t)
	var want [][]byte
	for n := range 3 {
		want = append(want, s.add(t, n))
	}
	conn, _, err := s.dialEvents(t, s.d.practitioner, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []frame
	for range 2 {
		f, err := next(t, conn)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, f)
	}
	_ = conn.Close() // the connection drops after the second entry

	for n := 3; n < 160; n++ { // missed while disconnected: more than a page
		want = append(want, s.add(t, n))
	}
	conn, _, err = s.dialEvents(t, s.d.practitioner, got[len(got)-1].cursor)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Go(func() { // more arrive during the replay
		for n := 160; n < 220; n++ {
			env := s.add(t, n)
			mu.Lock()
			want = append(want, env)
			mu.Unlock()
		}
	})
	for len(got) < 220 {
		f, err := next(t, conn)
		if err != nil {
			t.Fatalf("after %d entries: %v", len(got), err)
		}
		got = append(got, f)
	}
	wg.Wait()
	live := s.add(t, 220)
	f, err := next(t, conn)
	if err != nil || !bytes.Equal(f.envelope, live) {
		t.Fatalf("live entry = %s, %v; want %s", f.envelope, err, live)
	}
	got = append(got, f)
	want = append(want, live)

	for i := range want {
		if !bytes.Equal(got[i].envelope, want[i]) {
			t.Fatalf("entry %d = %s, want %s", i, got[i].envelope, want[i])
		}
		if i > 0 && !streamIDAfter(got[i].cursor, got[i-1].cursor) {
			t.Fatalf("cursor %d = %s after %s, want strictly increasing", i, got[i].cursor, got[i-1].cursor)
		}
	}
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE subtype_code = 'operation' AND action = 'E' AND outcome = 0
		AND entity_reference = $1`, "Encounter/"+s.encounter); n != 2 {
		t.Errorf("audited %d subscriptions, want 2", n)
	}
}

// streamIDAfter reports whether the stream id a is after b.
func streamIDAfter(a, b string) bool {
	am, as, _ := strings.Cut(a, "-")
	bm, bs, _ := strings.Cut(b, "-")
	if len(am) != len(bm) {
		return len(am) > len(bm)
	}
	if am != bm {
		return am > bm
	}
	if len(as) != len(bs) {
		return len(as) > len(bs)
	}
	return as > bs
}

func (s streamApp) count(t *testing.T, q string, args ...any) (n int) {
	t.Helper()
	if err := s.owner.QueryRow(t.Context(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Only the encounter's practitioner follows the stream: the patient and the clinic's admin are refused, an
// account the appointment does not show gets not found; each refusal is audited.
func TestNew_RefusesTheEventStreamToAnyoneButTheEncountersPractitioner(t *testing.T) {
	s := newStreamApp(t)
	for _, tt := range []struct {
		cl     *client
		name   string
		status int
	}{
		{name: "the patient", cl: s.d.patient, status: http.StatusForbidden},
		{name: "the clinic's admin", cl: s.d.admin, status: http.StatusForbidden},
		{name: "a stranger", cl: s.stranger, status: http.StatusNotFound},
	} {
		if _, res, err := s.dialEvents(t, tt.cl, ""); err == nil || res == nil || res.StatusCode != tt.status {
			t.Errorf("%s: dial = %v, %v; want %d", tt.name, res, err, tt.status)
		}
	}
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE subtype_code = 'operation' AND action = 'E' AND outcome = 4`); n != 3 {
		t.Errorf("audited %d refused subscriptions, want 3", n)
	}
}

// A revocation ends the stream at once, mid-consultation: an entry added after it commits never reaches the
// practitioner, and an idle stream closes within RecheckEvery. A new subscription is refused.
func TestNew_EndsTheEventStreamAtOnceOnARevocation(t *testing.T) {
	for _, idle := range []bool{false, true} {
		s := newStreamApp(t)
		s.add(t, 0)
		conn, _, err := s.dialEvents(t, s.d.practitioner, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := next(t, conn); err != nil {
			t.Fatal(err)
		}
		expectHTTP(t, s.d.patient, http.MethodPost, "/consents/"+s.consent+"/revoke", http.StatusOK)
		revoked := time.Now()
		if !idle {
			s.add(t, 1)
		}
		_ = conn.SetReadDeadline(time.Now().Add(3 * transcript.RecheckEvery))
		_, msg, err := conn.ReadMessage()
		var closed *websocket.CloseError
		if !errors.As(err, &closed) || closed.Code != 4403 || closed.Text != "consent_required" {
			t.Errorf("idle %v: after the revocation read %s, %v; want close 4403 consent_required", idle, msg, err)
		}
		if took := time.Since(revoked); took > 2*transcript.RecheckEvery {
			t.Errorf("idle %v: closed %v after the revocation, want within %v", idle, took, 2*transcript.RecheckEvery)
		}
		if _, res, err := s.dialEvents(t, s.d.practitioner, ""); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
			t.Errorf("idle %v: dial after the revocation = %v, %v; want 403", idle, res, err)
		}
	}
}

// The practitioner exports the stored transcript in the order it was said; nobody else does, and not after a
// revocation. Each read is audited with the segments it returned.
func TestNew_ExportsTheTranscriptToTheEncountersPractitioner(t *testing.T) {
	s := newStreamApp(t)
	for i, at := range []string{"2030-01-07T09:01:00Z", "2030-01-07T09:00:00Z"} {
		if _, err := s.owner.Exec(t.Context(), `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, translation, ts_start, ts_end)
			VALUES (gen_random_uuid(), $1, 'patient', 'bs', $2, '', $3, $3)`, s.encounter, "Placeholder original "+itoa(i), at); err != nil {
			t.Fatal(err)
		}
	}
	path := "/encounters/" + s.encounter + "/transcript"
	body := expectHTTP(t, s.d.practitioner, http.MethodGet, path, http.StatusOK)
	var got struct{ Segments []transcript.Event }
	if err := json.Unmarshal([]byte(body), &got); err != nil || len(got.Segments) != 2 ||
		got.Segments[0].Original != "Placeholder original 1" || !got.Segments[0].IsFinal || got.Segments[1].Original != "Placeholder original 0" {
		t.Errorf("export = %s, %v; want both segments in order", body, err)
	}
	expectHTTP(t, s.d.patient, http.MethodGet, path, http.StatusForbidden, `"forbidden"`)
	expectHTTP(t, s.d.admin, http.MethodGet, path, http.StatusForbidden, `"forbidden"`)
	expectHTTP(t, s.stranger, http.MethodGet, path, http.StatusNotFound, `"not_found"`)
	expectHTTP(t, s.d.patient, http.MethodPost, "/consents/"+s.consent+"/revoke", http.StatusOK)
	expectHTTP(t, s.d.practitioner, http.MethodGet, path, http.StatusForbidden, `"consent_required"`)
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE subtype_code = 'read' AND entity_name = 'transcript_segment'
		AND entity_count = 2 AND outcome = 0 AND entity_reference = $1`, "Encounter/"+s.encounter); n != 1 {
		t.Errorf("audited %d exports of two segments, want 1", n)
	}
}
