package communication_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/livesocket"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

func msg(seq int64) communication.Message {
	m := sample
	m.Seq, m.SentAt = seq, start.Add(time.Duration(seq)*time.Second)
	return m
}

func frame(t *testing.T, typ string, m communication.Message) []byte {
	t.Helper()
	b, err := wire.Encode(typ, communication.FrameVersion, m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// dialChat opens the conversation's socket; the status is the handshake's answer (0 when there was none).
func dialChat(t *testing.T, srv string, header http.Header) (*websocket.Conn, int, error) {
	t.Helper()
	conn, res, err := websocket.DefaultDialer.DialContext(t.Context(), "ws"+strings.TrimPrefix(srv, "http")+convPath+"/events", header)
	status := 0
	if res != nil {
		status = res.StatusCode
		_ = res.Body.Close()
	}
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
		_ = conn.SetReadDeadline(clock.System{}.Now().Add(5 * time.Second))
	}
	return conn, status, err
}

// readFrame reads the next frame and returns its type and message number.
func readFrame(t *testing.T, conn *websocket.Conn) (string, int64) {
	t.Helper()
	_, b, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var head struct {
		Type    string `json:"type"`
		Payload struct {
			Error   string `json:"error"`
			Seq     int64  `json:"seq"`
			Message struct {
				Seq int64 `json:"seq"`
			} `json:"message"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		t.Fatalf("frame %s: %v", b, err)
	}
	switch head.Type {
	case communication.SendAckType:
		return head.Type, head.Payload.Message.Seq
	case communication.CommandErrorType:
		return head.Type + ":" + head.Payload.Error, 0
	}
	return head.Type, head.Payload.Seq
}

// The socket replays from Postgres after the client's cursor, then forwards the live frames in order: a frame it
// sent already is dropped, and after a gap the missing messages come from Postgres before the next one.
func TestSocket_ReplaysThenForwardsLiveFramesInOrderAndFillsAGap(t *testing.T) {
	t.Parallel()
	svc := &stubChats{
		sub:   communication.Subscription{From: msg(1).Cursor()},
		after: [][]communication.Message{{msg(2), msg(3)}, {msg(5)}}, // the replay, then the gap's fill
	}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	for _, want := range []int64{2, 3} {
		if typ, seq := readFrame(t, conn); typ != communication.MessageType || seq != want {
			t.Fatalf("replay: %s %d, want message %d", typ, seq, want)
		}
	}
	ch := communication.ChannelKey(sample.ConversationID)
	live.publish(ch, frame(t, communication.MessageType, msg(3))) // sent already
	live.publish(ch, frame(t, communication.MessageType, msg(4)))
	live.publish(ch, frame(t, communication.MessageType, msg(6))) // 5 was dropped by pub/sub
	for _, want := range []int64{4, 5} {
		if typ, seq := readFrame(t, conn); typ != communication.MessageType || seq != want {
			t.Fatalf("live: %s %d, want message %d", typ, seq, want)
		}
	}
	live.publish(ch, frame(t, communication.MessageType, msg(6))) // after the fill, 6 is next
	if typ, seq := readFrame(t, conn); seq != 6 {
		t.Errorf("after the gap: %s %d, want 6", typ, seq)
	}
}

// An edit goes out unless its message is before the caller's starting point (history_from_seq, SAL-28).
func TestSocket_ForwardsEditsOnlyFromTheCallersStartingPoint(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sub: communication.Subscription{From: msg(5).Cursor(), HistoryFromSeq: 3}}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	ch := communication.ChannelKey(sample.ConversationID)
	live.publish(ch, frame(t, communication.MessageEditedType, msg(2)))
	live.publish(ch, frame(t, communication.MessageEditedType, msg(4)))
	if typ, seq := readFrame(t, conn); typ != communication.MessageEditedType || seq != 4 {
		t.Errorf("edit: %s %d, want only the edit of 4", typ, seq)
	}
}

func TestSocket_SendIsAnsweredWithItsAckAndABadFrameWithAnError(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sub: communication.Subscription{}, sent: msg(1), created: true}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	send, _ := wire.Encode(communication.SendType, communication.FrameVersion, communication.SendCommand{ClientMessageID: clientID2, Body: "Synthetic placeholder"})
	for _, f := range [][]byte{send, []byte("not an envelope"), []byte(`{"type":"typing","version":1,"payload":{}}`)} {
		if err := conn.WriteMessage(websocket.TextMessage, f); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{communication.SendAckType, "command_error:invalid_message", "command_error:unknown_command"} {
		if typ, _ := readFrame(t, conn); typ != want {
			t.Errorf("answer %s, want %s", typ, want)
		}
	}
	if calls := svc.calls(); len(calls) != 1 || calls[0] != "send "+clientID2+" Synthetic placeholder" {
		t.Errorf("calls = %q", calls)
	}
	svc.mu.Lock()
	svc.err = communication.ErrConsentRequired
	svc.mu.Unlock()
	if err := conn.WriteMessage(websocket.TextMessage, send); err != nil {
		t.Fatal(err)
	}
	if typ, _ := readFrame(t, conn); typ != "command_error:consent_required" {
		t.Errorf("a refused send: %s", typ)
	}
}

// When the periodic check finds access ended, the socket closes with the reason.
func TestSocket_ClosesWithTheReasonWhenAccessEnds(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		reason string
		code   int
	}{
		{communication.ErrConsentRequired, "consent_required", livesocket.CloseForbidden},
		{communication.ErrForbidden, "forbidden", livesocket.CloseForbidden},
		{communication.ErrNotFound, "not_found", livesocket.CloseNotFound},
		{errors.New("the database is down"), "internal error", websocket.CloseInternalServerErr},
	} {
		svc := &stubChats{after: [][]communication.Message{nil}, afterErr: tt.err}
		srv := newServerEvery(t, svc, newFakeLive(), true, 20*time.Millisecond)
		conn, _, err := dialChat(t, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = conn.ReadMessage()
		var ce *websocket.CloseError
		if !errors.As(err, &ce) || ce.Code != tt.code || ce.Text != tt.reason {
			t.Errorf("%v: closed with %v, want %d %s", tt.err, err, tt.code, tt.reason)
		}
	}
}

func TestSocket_RefusesBeforeTheUpgrade(t *testing.T) {
	t.Parallel()
	srv := newServer(t, &stubChats{}, newFakeLive(), true)
	if code, body, _ := do(t, srv, http.MethodGet, convPath+"/events", "", nil); code != http.StatusBadRequest || body != `{"error":"websocket_required"}` {
		t.Errorf("no upgrade: %d %s", code, body)
	}
	if _, status, err := dialChat(t, srv.URL, http.Header{"Origin": {"https://elsewhere.example.test"}}); err == nil || status != http.StatusForbidden {
		t.Errorf("another origin: %v", err)
	}
	refused := newServer(t, &stubChats{err: communication.ErrForbidden}, newFakeLive(), true)
	if _, status, err := dialChat(t, refused.URL, nil); err == nil || status != http.StatusForbidden {
		t.Errorf("a refused subscription: %v", err)
	}
	signedOut := newServer(t, &stubChats{}, newFakeLive(), false)
	if _, status, err := dialChat(t, signedOut.URL, nil); err == nil || status != http.StatusUnauthorized {
		t.Errorf("signed out: %v", err)
	}
}

// A logout or a revoked session ends an open socket at its next check (4401), whatever the conversation's access.
func TestSocket_ClosesWhenTheSessionEnds(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sessionEnded: true}
	srv := newServerEvery(t, svc, newFakeLive(), true, 20*time.Millisecond)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = conn.ReadMessage()
	var ce *websocket.CloseError
	if !errors.As(err, &ce) || ce.Code != livesocket.CloseUnauthenticated || ce.Text != "unauthenticated" {
		t.Errorf("closed with %v, want 4401 unauthenticated", err)
	}
}

// An edit of a message the socket has not sent yet fills from Postgres first: the client never sees an edit before
// its message.
func TestSocket_AnEditOfAMessageNotSentYetFillsFirst(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sub: communication.Subscription{From: msg(1).Cursor()}, after: [][]communication.Message{nil, {msg(2)}}}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	live.publish(communication.ChannelKey(sample.ConversationID), frame(t, communication.MessageEditedType, msg(2)))
	if typ, seq := readFrame(t, conn); typ != communication.MessageType || seq != 2 {
		t.Errorf("first frame %s %d, want message 2 from the fill", typ, seq)
	}
}

// A client writing in a loop is held to the socket's limit: a burst of five, then one every 200 ms.
func TestSocket_LimitsTheClientsSends(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sent: msg(1), created: true}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	send, _ := wire.Encode(communication.SendType, communication.FrameVersion, communication.SendCommand{ClientMessageID: clientID2, Body: "Synthetic placeholder"})
	for range 8 {
		if err := conn.WriteMessage(websocket.TextMessage, send); err != nil {
			t.Fatal(err)
		}
	}
	limited := 0
	for range 8 {
		if typ, _ := readFrame(t, conn); typ == "command_error:rate_limited" {
			limited++
		}
	}
	if limited < 1 || limited > 3 {
		t.Errorf("%d of 8 sends limited, want the ones past the burst of 5", limited)
	}
}
