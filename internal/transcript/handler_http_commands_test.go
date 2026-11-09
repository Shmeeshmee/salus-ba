package transcript_test

import (
	"bytes"
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// stubCommands answers every command with "reply-<n>" (n counting from 1), or nothing for a payload "silent",
// and records what it was given.
type stubCommands struct {
	got []string // "<caller account> <encounter> <type> v<version> <payload>"
	mu  sync.Mutex
}

func (s *stubCommands) Command(_ context.Context, c appointment.Caller, encounterID string, env wire.Envelope) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = append(s.got, c.AccountID+" "+encounterID+" "+env.Type+" v"+strconv.Itoa(env.Version)+" "+string(env.Payload))
	if string(env.Payload) == `"silent"` {
		return nil
	}
	return []byte("reply-" + strconv.Itoa(len(s.got)))
}

func (s *stubCommands) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.got...)
}

// A text frame from the client is a message in the wire envelope: it goes to the commands with the caller and
// the encounter, and their answer, if any, comes back on the socket, between the stream's frames. A frame that
// is no envelope is answered with a command_error and goes nowhere.
func TestEvents_HandsTheClientsMessagesToTheCommandsAndSendsTheirAnswer(t *testing.T) {
	commands := &stubCommands{}
	srv := newServerWith(t, &stubTranscripts{}, commands, signedIn)
	conn, _, err := dial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range []string{
		`{"type":"suggestion_status","version":1,"payload":"silent"}`,
		`{"type":"suggestion_status","version":1,"payload":{"id":"x"}}`,
		`not an envelope`,
	} {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"reply-2", `{"type":"command_error","version":1,"payload":{"command":"","error":"invalid_message"}}`} {
		if _, got, err := read(t, conn); err != nil || string(got) != want {
			t.Fatalf("frame = %s, %v; want %s", got, err, want)
		}
	}
	want := []string{
		pract.AccountID + " " + encounterID + ` suggestion_status v1 "silent"`,
		pract.AccountID + " " + encounterID + ` suggestion_status v1 {"id":"x"}`,
	}
	if got := commands.calls(); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("commands got %q,\nwant %q", got, want)
	}
}

// Without commands every message is answered as unknown; a frame over the read limit ends the socket.
func TestEvents_AnswersAnUnknownCommandAndDropsAnOversizedFrame(t *testing.T) {
	conn, _, err := dial(t, newServer(t, &stubTranscripts{}, signedIn), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"suggestion_status","version":1,"payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	want := `{"type":"command_error","version":1,"payload":{"command":"suggestion_status","error":"unknown_command"}}`
	if _, got, err := read(t, conn); err != nil || string(got) != want {
		t.Fatalf("frame = %s, %v; want %s", got, err, want)
	}
	if err := conn.WriteMessage(websocket.TextMessage, bytes.Repeat([]byte("a"), 4096)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := read(t, conn); err == nil {
		t.Error("the socket stayed open after an oversized frame")
	}
}

// The stream's frames and the commands' answers are written by one writer at a time (run with -race).
func TestEvents_WritesTheStreamAndTheAnswersOneAtATime(t *testing.T) {
	var script []next
	for i := range 50 {
		script = append(script, next{entries: []transcript.Entry{{ID: strconv.Itoa(i+1) + "-0", Envelope: envelope(i)}}})
	}
	srv := newServerWith(t, &stubTranscripts{script: script}, &stubCommands{}, signedIn)
	conn, _, err := dial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for range 50 {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"x","version":1,"payload":{}}`))
		}
	}()
	replies, frames := 0, 0
	for replies < 50 || frames < 100 {
		_, got, err := read(t, conn)
		if err != nil {
			t.Fatalf("after %d replies and %d frames: %v", replies, frames, err)
		}
		if bytes.HasPrefix(got, []byte("reply-")) {
			replies++
		} else {
			frames++
		}
	}
}
