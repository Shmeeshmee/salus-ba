package transcript_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// next is one scripted answer of the stub's Next.
type next struct {
	err     error
	entries []transcript.Entry
}

// stubTranscripts answers the handler with scripted results and records its calls. Once its script of Next
// answers is used up, Next waits for the request to end.
type stubTranscripts struct {
	exportErr    error
	subscribeErr error
	afters       chan string // every cursor Next was called with
	export       []transcript.Event
	script       []next
	subscribed   []string // the cursors Subscribe was called with
	mu           sync.Mutex
}

func (s *stubTranscripts) Export(context.Context, appointment.Caller, string) ([]transcript.Event, error) {
	return s.export, s.exportErr
}

func (s *stubTranscripts) Subscribe(_ context.Context, _ appointment.Caller, _, after string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribed = append(s.subscribed, after)
	if after == "" {
		after = "0"
	}
	return after, s.subscribeErr
}

func (s *stubTranscripts) Next(ctx context.Context, _ appointment.Caller, _, after string) ([]transcript.Entry, error) {
	s.afters <- after
	s.mu.Lock()
	if len(s.script) > 0 {
		n := s.script[0]
		s.script = s.script[1:]
		s.mu.Unlock()
		return n.entries, n.err
	}
	s.mu.Unlock()
	<-ctx.Done()
	return nil, ctx.Err()
}

func signedIn(context.Context) (appointment.Caller, bool, error) { return pract, true, nil }

func newServer(t *testing.T, svc *stubTranscripts, caller func(context.Context) (appointment.Caller, bool, error)) *httptest.Server {
	t.Helper()
	if svc.afters == nil {
		svc.afters = make(chan string, 100)
	}
	mux := http.NewServeMux()
	h := transcript.NewHTTPHandler(svc, caller, slog.New(slog.DiscardHandler))
	h.Routes(mux)
	h.StreamRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// dial opens the encounter's event stream on srv as the web app does: from its own origin.
func dial(t *testing.T, srv *httptest.Server, query string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/encounters/" + encounterID + "/events" + query
	conn, res, err := websocket.DefaultDialer.DialContext(t.Context(), url, http.Header{"Origin": {srv.URL}})
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
	}
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	return conn, res, err
}

// read reads the next frame within a second.
func read(t *testing.T, conn *websocket.Conn) (int, []byte, error) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	return conn.ReadMessage()
}

func cursor(id string) string {
	return `{"type":"cursor","version":1,"payload":{"id":"` + id + `"}}`
}

// Every entry goes out as two text frames: a cursor with its stream id, then its envelope's bytes unchanged.
// An entry without an envelope is not sent. The next read starts after the last entry read.
func TestEvents_ForwardsEachEnvelopeUnchangedAfterItsCursor(t *testing.T) {
	odd := []byte(`{"type":"suggestion","version":1,"payload":{"text":"spacing  kept é"}}`)
	svc := &stubTranscripts{script: []next{
		{entries: []transcript.Entry{{ID: "1-0", Envelope: envelope(1)}, {ID: "2-0"}, {ID: "3-0", Envelope: odd}}},
		{entries: nil},
		{entries: []transcript.Entry{{ID: "4-0", Envelope: envelope(4)}}},
	}}
	srv := newServer(t, svc, signedIn)
	conn, _, err := dial(t, srv, "?after=0-5")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{[]byte(cursor("1-0")), envelope(1), []byte(cursor("3-0")), odd, []byte(cursor("4-0")), envelope(4)} {
		typ, got, err := read(t, conn)
		if err != nil || typ != websocket.TextMessage || !bytes.Equal(got, want) {
			t.Fatalf("frame = %d %s, %v; want text %s", typ, got, err, want)
		}
	}
	if svc.subscribed[0] != "0-5" {
		t.Errorf("subscribed after %q, want 0-5", svc.subscribed[0])
	}
	for _, want := range []string{"0-5", "3-0", "3-0", "4-0"} {
		if got := <-svc.afters; got != want {
			t.Errorf("Next after %q, want %q", got, want)
		}
	}
}

// When access ends mid-stream the gateway closes the socket with the reason; nothing more is sent.
func TestEvents_ClosesTheSocketWhenAccessEnds(t *testing.T) {
	for _, tt := range []struct {
		err    error
		reason string
		code   int
	}{
		{err: transcript.ErrConsentRequired, code: 4403, reason: "consent_required"},
		{err: transcript.ErrForbidden, code: 4403, reason: "forbidden"},
		{err: transcript.ErrNotFound, code: 4404, reason: "not_found"},
		{err: errors.New("the stream is down"), code: websocket.CloseInternalServerErr, reason: "internal error"},
	} {
		svc := &stubTranscripts{script: []next{{entries: []transcript.Entry{{ID: "1-0", Envelope: envelope(1)}}}, {err: tt.err}}}
		conn, _, err := dial(t, newServer(t, svc, signedIn), "")
		if err != nil {
			t.Fatal(err)
		}
		for range 2 { // the cursor and the envelope of the entry before
			if _, _, err := read(t, conn); err != nil {
				t.Fatal(err)
			}
		}
		_, _, err = read(t, conn)
		var closed *websocket.CloseError
		if !errors.As(err, &closed) || closed.Code != tt.code || closed.Text != tt.reason {
			t.Errorf("%v: read = %v, want close %d %q", tt.err, err, tt.code, tt.reason)
		}
	}
}

// The upgrade is refused before anything is read or audited: without a session, without a WebSocket handshake,
// and from another origin (the session cookie is SameSite=Strict and a WebSocket cannot send X-Salus-CSRF, so
// the origin is the cross-site check). A request without Origin is no browser's, so no cross-site one.
func TestEvents_RefusesTheUpgradeBeforeSubscribing(t *testing.T) {
	svc := &stubTranscripts{}
	nobody := func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }
	srv, anon := newServer(t, svc, signedIn), newServer(t, &stubTranscripts{}, nobody)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/encounters/" + encounterID + "/events"

	for name, tt := range map[string]struct {
		header http.Header
		url    string
		status int
	}{
		"another origin":       {url: url, header: http.Header{"Origin": {"https://evil.example"}}, status: http.StatusForbidden},
		"a sibling origin":     {url: url, header: http.Header{"Origin": {"http://landing." + strings.TrimPrefix(srv.URL, "http://")}}, status: http.StatusForbidden},
		"no session":           {url: "ws" + strings.TrimPrefix(anon.URL, "http") + "/encounters/" + encounterID + "/events", header: http.Header{"Origin": {anon.URL}}, status: http.StatusUnauthorized},
		"an invalid cursor":    {url: url + "?after=$", header: http.Header{"Origin": {srv.URL}}, status: http.StatusBadRequest},
		"an unparseable start": {url: url, header: http.Header{"Origin": {"::not a url"}}, status: http.StatusForbidden},
	} {
		if name == "an invalid cursor" {
			svc.subscribeErr = transcript.ErrInvalidCursor
		}
		conn, res, err := websocket.DefaultDialer.DialContext(t.Context(), tt.url, tt.header)
		if conn != nil {
			_ = conn.Close()
		}
		if err == nil || res == nil || res.StatusCode != tt.status {
			t.Errorf("%s: dial = %v, %v; want %d", name, res, err, tt.status)
		}
		if res != nil {
			_ = res.Body.Close()
		}
		svc.subscribeErr = nil
	}
	if len(svc.subscribed) != 1 { // only the invalid cursor reached the service, which refused it
		t.Errorf("subscribed %d times, want 1", len(svc.subscribed))
	}

	res, err := srv.Client().Get(srv.URL + "/encounters/" + encounterID + "/events")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("GET without the handshake = %d, want 400", res.StatusCode)
	}
	if conn, _, err := websocket.DefaultDialer.DialContext(t.Context(), url, nil); err != nil {
		t.Errorf("dial without Origin = %v, want the upgrade", err)
	} else {
		_ = conn.Close()
	}
}

// A refused subscription is answered before the upgrade, with the reason as a JSON error.
func TestEvents_AnswersARefusedSubscriptionWithItsStatus(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{
		{err: transcript.ErrForbidden, status: http.StatusForbidden},
		{err: transcript.ErrConsentRequired, status: http.StatusForbidden},
		{err: transcript.ErrNotFound, status: http.StatusNotFound},
		{err: errors.New("the database is down"), status: http.StatusInternalServerError},
	} {
		srv := newServer(t, &stubTranscripts{subscribeErr: tt.err}, signedIn)
		_, res, err := dial(t, srv, "")
		if err == nil || res == nil || res.StatusCode != tt.status {
			t.Errorf("%v: dial = %v, %v; want %d", tt.err, res, err, tt.status)
		}
	}
}

// The socket ends when the client goes away: the next read is cancelled.
func TestEvents_EndsWhenTheClientCloses(t *testing.T) {
	svc := &stubTranscripts{}
	conn, _, err := dial(t, newServer(t, svc, signedIn), "")
	if err != nil {
		t.Fatal(err)
	}
	<-svc.afters // the gateway waits in Next
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	_ = conn.Close()
	select {
	case after := <-svc.afters:
		t.Errorf("Next called again after %q once the client closed", after)
	case <-time.After(200 * time.Millisecond):
	}
}

// The export is the stored transcript as TranscriptEvent JSON, never cached.
func TestExport_AnswersTheTranscript(t *testing.T) {
	e := valid()
	srv := newServer(t, &stubTranscripts{export: []transcript.Event{e}}, signedIn)
	res, err := srv.Client().Get(srv.URL + "/encounters/" + encounterID + "/transcript")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var body bytes.Buffer
	_, _ = body.ReadFrom(res.Body)
	want := `{"segments":[{"ts_start":"2026-11-08T09:30:00Z","ts_end":"2026-11-08T09:30:01.5Z","encounter_id":"` + encounterID +
		`","segment_id":"` + segmentID + `","speaker":"patient","lang":"bs","original":"placeholder original",` +
		`"translation":"[de] placeholder original","is_final":true}]}` + "\n"
	if res.StatusCode != http.StatusOK || body.String() != want || res.Header.Get("Cache-Control") != "no-store" ||
		res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("GET transcript = %d %s %v;\nwant 200 %s, no-store", res.StatusCode, body.String(), res.Header, want)
	}
}

func TestExport_AnswersARefusalWithItsStatus(t *testing.T) {
	nobody := func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }
	broken := func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, errBackend }
	for _, tt := range []struct {
		err    error
		caller func(context.Context) (appointment.Caller, bool, error)
		body   string
		status int
	}{
		{caller: nobody, status: http.StatusUnauthorized, body: `{"error":"unauthenticated"}`},
		{caller: broken, status: http.StatusInternalServerError, body: `{"error":"internal error"}`},
		{caller: signedIn, err: transcript.ErrForbidden, status: http.StatusForbidden, body: `{"error":"forbidden"}`},
		{caller: signedIn, err: transcript.ErrConsentRequired, status: http.StatusForbidden, body: `{"error":"consent_required"}`},
		{caller: signedIn, err: transcript.ErrNotFound, status: http.StatusNotFound, body: `{"error":"not_found"}`},
		{caller: signedIn, err: errBackend, status: http.StatusInternalServerError, body: `{"error":"internal error"}`},
	} {
		srv := newServer(t, &stubTranscripts{exportErr: tt.err}, tt.caller)
		res, err := srv.Client().Get(srv.URL + "/encounters/" + encounterID + "/transcript")
		if err != nil {
			t.Fatal(err)
		}
		var body bytes.Buffer
		_, _ = body.ReadFrom(res.Body)
		_ = res.Body.Close()
		if res.StatusCode != tt.status || strings.TrimSpace(body.String()) != tt.body {
			t.Errorf("%v: GET transcript = %d %s, want %d %s", tt.err, res.StatusCode, body.String(), tt.status, tt.body)
		}
	}
}
