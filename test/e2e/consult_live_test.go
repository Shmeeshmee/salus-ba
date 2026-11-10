//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The live half of the consultation pipeline's journeys (SAL-10, consult_pipeline_test.go): the scripted
// conversations, a participant's voice, the practitioner's event stream, and the latency report.

// conversation is a scripted conversation of test/testdata/consult, which make e2e's fake engines also play
// (FAKE_ENGINE_SCRIPT, deployments/docker-compose.e2e.yml): each line is a tone the fake speech engine says as its
// text, in its speaker's language; the fake model translates it, suggests from it and fills fields from it.
type conversation struct {
	path  string
	Lines []struct {
		Translations map[string]string
		Speaker      string
		Lang         string
		Text         string
		Tone         int
	}
	Suggestions []struct{ After, Type, Text string }
	Fields      []struct {
		Key, Label, From, Value string
		Required                bool
	}
}

// loadConversation reads the conversation of a patient speaking language with a Bosnian practitioner.
func loadConversation(t *testing.T, language string) conversation {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "testdata", "consult", language+".json"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c := conversation{path: path}
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	// The journeys pin the first suggestion and dismiss the second, and expect a field filled as drafted, one to
	// type over and one never covered.
	if len(c.Suggestions) != 2 || len(c.Fields) != 3 || c.Fields[0].From == "" || c.Fields[1].From == "" || c.Fields[2].From != "" {
		t.Fatalf("%s: want 2 suggestions and 3 fields, the first two filled from a line and the third never", path)
	}
	return c
}

// recordConversation adds the conversation's texts to the values LiveKit's log must not hold: LiveKit carries the
// tones, never a word of the transcript.
func recordConversation(t *testing.T, c conversation) {
	t.Helper()
	for _, l := range c.Lines {
		appendLiveKitCheck(t, "absent", "transcript line", l.Text)
		for _, tr := range l.Translations {
			appendLiveKitCheck(t, "absent", "translation", tr)
		}
	}
}

// firstHalf is the partial result the fake speech engine gives of a line (stttest): its first half of words.
func firstHalf(text string) string {
	words := strings.Fields(text)
	return strings.Join(words[:(len(words)+1)/2], " ")
}

// lineTone is how long a line's tone plays.
const lineTone = 1200 * time.Millisecond

// voice is a participant's microphone: silence, and a tone when it says a line, in G.711 µ-law 20 ms frames.
type voice struct {
	say chan toneRequest
}

type toneRequest struct {
	said chan [2]time.Time // the times of the first and the last frame of the tone
	hz   int
}

// speak publishes the room's microphone track, silent until a line is said, until the test ends.
func speak(t *testing.T, room *lksdk.Room) *voice {
	t.Helper()
	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Source: livekit.TrackSource_MICROPHONE}); err != nil {
		t.Fatalf("publish the microphone: %v", err)
	}
	v := &voice{say: make(chan toneRequest)}
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go v.play(ctx, track)
	return v
}

// play writes a frame every 20 ms: the tone asked for, for lineTone, else silence.
func (v *voice) play(ctx context.Context, track *lksdk.LocalSampleTrack) {
	const rate, frame = 8000, 160
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	var req toneRequest
	var first time.Time
	left, n := 0, 0
	for {
		if left == 0 {
			select {
			case req = <-v.say:
				left, first = int(lineTone/(20*time.Millisecond)), time.Time{}
			default:
			}
		}
		data := make([]byte, frame)
		for i := range data {
			s := 0.0
			if left > 0 {
				s = 8000 * math.Sin(2*math.Pi*float64(req.hz)*float64(n*frame+i)/rate)
			}
			data[i] = mulaw(int16(s))
		}
		n++
		if err := track.WriteSample(media.Sample{Data: data, Duration: 20 * time.Millisecond}, nil); err != nil {
			return
		}
		if left > 0 {
			now := clock.System{}.Now()
			if first.IsZero() {
				first = now
			}
			if left--; left == 0 {
				req.said <- [2]time.Time{first, now}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// line says the tone and returns when its first and last frames were sent.
func (v *voice) line(t *testing.T, hz int) (on, off time.Time) {
	t.Helper()
	said := make(chan [2]time.Time, 1)
	select {
	case v.say <- toneRequest{hz: hz, said: said}:
	case <-time.After(10 * time.Second):
		t.Fatal("the voice does not speak")
	}
	select {
	case at := <-said:
		return at[0], at[1]
	case <-time.After(10 * time.Second):
		t.Fatal("the voice did not finish its line")
	}
	return time.Time{}, time.Time{}
}

// arrival is one message of the event stream, with when it arrived.
type arrival struct {
	at      time.Time
	Type    string
	Payload json.RawMessage
}

// transcriptLine is the part of a transcript event the journeys read.
type transcriptLine struct {
	SegmentID   string `json:"segment_id"`
	Speaker     string `json:"speaker"`
	Lang        string `json:"lang"`
	Original    string `json:"original"`
	Translation string `json:"translation"`
	IsFinal     bool   `json:"is_final"`
}

// suggestionEvent is the part of a suggestion event the journeys read.
type suggestionEvent struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Text             string   `json:"text"`
	Status           string   `json:"status"`
	Origin           string   `json:"origin"`
	SourceSegmentIDs []string `json:"source_segment_ids"`
}

// events is a practitioner's view of the encounter's event stream (GET /encounters/{id}/events, the socket the
// web app opens), on which it keeps every message with its arrival time.
type events struct {
	conn    *websocket.Conn
	changed chan struct{} // a message arrived, or the socket closed
	closed  chan struct{}
	got     []arrival
	mu      sync.Mutex
	code    int // the close code, once closed
}

// openEvents opens the stream as a, with its bearer token, as the API's own origin would (no Origin header).
func openEvents(t *testing.T, a *api, encounterID string) *events {
	t.Helper()
	u := strings.Replace(a.base, "http", "ws", 1) + "/encounters/" + encounterID + "/events"
	conn, resp, err := (&websocket.Dialer{HandshakeTimeout: 10 * time.Second}).DialContext(t.Context(), u,
		http.Header{"Authorization": {"Bearer " + a.bearer}})
	if err != nil {
		t.Fatalf("open the event stream: %v", err)
	}
	_ = resp.Body.Close()
	e := &events{conn: conn, changed: make(chan struct{}, 1), closed: make(chan struct{})}
	t.Cleanup(func() { _ = conn.Close() })
	go e.read()
	return e
}

func (e *events) read() {
	defer close(e.closed)
	for {
		_, b, err := e.conn.ReadMessage()
		at := clock.System{}.Now()
		if err != nil {
			e.mu.Lock()
			if ce, ok := err.(*websocket.CloseError); ok {
				e.code = ce.Code
			}
			e.mu.Unlock()
			e.notify()
			return
		}
		var m arrival
		if json.Unmarshal(b, &m) != nil || m.Type == "cursor" {
			continue
		}
		m.at = at
		e.mu.Lock()
		e.got = append(e.got, m)
		e.mu.Unlock()
		e.notify()
	}
}

func (e *events) notify() {
	select {
	case e.changed <- struct{}{}:
	default:
	}
}

// wait returns the first message that matches, waiting up to timeout for it.
func (e *events) wait(t *testing.T, what string, timeout time.Duration, match func(arrival) bool) arrival {
	t.Helper()
	deadline := time.After(timeout)
	for {
		e.mu.Lock()
		i := slices.IndexFunc(e.got, match)
		var m arrival
		if i >= 0 {
			m = e.got[i]
		}
		e.mu.Unlock()
		if i >= 0 {
			return m
		}
		select {
		case <-e.changed:
		case <-e.closed:
			t.Fatalf("%s: the event stream closed first (code %d)", what, e.closeCode())
		case <-deadline:
			t.Fatalf("%s: not on the event stream within %v", what, timeout)
		}
	}
}

// transcript waits for the transcript event that matches.
func (e *events) transcript(t *testing.T, what string, match func(transcriptLine) bool) (transcriptLine, time.Time) {
	t.Helper()
	var line transcriptLine
	m := e.wait(t, what, 30*time.Second, func(m arrival) bool {
		return m.Type == "transcript" && json.Unmarshal(m.Payload, &line) == nil && match(line)
	})
	_ = json.Unmarshal(m.Payload, &line)
	return line, m.at
}

// suggestion waits for the suggestion with the text and status.
func (e *events) suggestion(t *testing.T, text, status string) suggestionEvent {
	t.Helper()
	var s suggestionEvent
	m := e.wait(t, fmt.Sprintf("the %s suggestion %q", status, text), 60*time.Second, func(m arrival) bool {
		return m.Type == "suggestion" && json.Unmarshal(m.Payload, &s) == nil && s.Text == text && s.Status == status
	})
	_ = json.Unmarshal(m.Payload, &s)
	return s
}

// send writes a command frame (ConsultCommand).
func (e *events) send(t *testing.T, typ string, payload any) {
	t.Helper()
	if err := e.conn.WriteJSON(map[string]any{"type": typ, "version": 1, "payload": payload}); err != nil {
		t.Fatalf("send %s: %v", typ, err)
	}
}

func (e *events) closeCode() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.code
}
