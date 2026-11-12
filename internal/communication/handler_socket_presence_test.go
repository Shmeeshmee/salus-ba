package communication_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

const partMe = "d0000001-0000-4000-8000-000000000000"

func writeFrame(t *testing.T, conn *websocket.Conn, typ string, payload any) {
	t.Helper()
	b, err := wire.Encode(typ, communication.FrameVersion, payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		t.Fatal(err)
	}
}

// Once it has replayed, the socket sends how far each participant has read (the receipts kept in Redis), then the
// live receipts and typing frames as they come (SAL-30).
func TestSocket_SendsTheKeptReceiptsThenTheLiveReceiptsAndTyping(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sub: communication.Subscription{From: msg(1).Cursor(), ParticipantID: partMe}, after: [][]communication.Message{{msg(2)}}}
	svc.sal30.receipts = []communication.Receipt{{ParticipantID: "d0000002-0000-4000-8000-000000000000", Seq: 2}}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	if typ, seq := readFrame(t, conn); typ != communication.MessageType || seq != 2 {
		t.Fatalf("replay: %s %d", typ, seq)
	}
	if typ, seq := readFrame(t, conn); typ != communication.ReceiptType || seq != 2 {
		t.Fatalf("kept receipt: %s %d", typ, seq)
	}
	ch := communication.ChannelKey(sample.ConversationID)
	receipt, _ := wire.Encode(communication.ReceiptType, communication.FrameVersion, communication.Receipt{ParticipantID: partMe, Seq: 2})
	typing, _ := wire.Encode(communication.TypingType, communication.FrameVersion, communication.Typing{ParticipantID: partMe})
	live.publish(ch, receipt)
	live.publish(ch, typing)
	for _, want := range [][]byte{receipt, typing} {
		_, got, err := conn.ReadMessage()
		if err != nil || string(got) != string(want) {
			t.Errorf("live: %s, %v; want %s", got, err, want)
		}
	}
}

// A read frame moves the caller's read position as PUT …/read does, answered with nothing (a refusal with its code);
// typing frames go to the service for the caller's participant at most once a second, answered with nothing.
func TestSocket_TakesReadAndTypingFramesWithoutAnAnswer(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sub: communication.Subscription{ParticipantID: partMe}, sent: msg(1), created: true}
	live := newFakeLive()
	srv := newServer(t, svc, live, true)
	conn, _, err := dialChat(t, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-live.ready
	writeFrame(t, conn, communication.ReadType, communication.ReadCommand{Seq: 3})
	writeFrame(t, conn, communication.TypingType, struct{}{})
	writeFrame(t, conn, communication.TypingType, struct{}{})
	writeFrame(t, conn, communication.SendType, communication.SendCommand{ClientMessageID: clientID2, Body: "Synthetic placeholder"})
	if typ, _ := readFrame(t, conn); typ != communication.SendAckType {
		t.Fatalf("the first answer: %s, want the send's ack (read and typing get none)", typ)
	}
	if calls := svc.calls(); !slices.Equal(calls, []string{"read 3", "send " + clientID2 + " Synthetic placeholder"}) {
		t.Errorf("calls %q", calls)
	}
	svc.mu.Lock()
	typing := slices.Clone(svc.sal30.typing)
	svc.err = communication.ErrInvalid
	svc.mu.Unlock()
	if !slices.Equal(typing, []string{partMe}) {
		t.Errorf("typing %v, want once for the caller's participant", typing)
	}
	writeFrame(t, conn, communication.ReadType, json.RawMessage(`{"seq":99}`))
	if typ, _ := readFrame(t, conn); typ != "command_error:invalid_request" {
		t.Errorf("a refused read: %s", typ)
	}
}
