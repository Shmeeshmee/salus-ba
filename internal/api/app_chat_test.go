package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The chat through api.New (docs/plans/chat.md, SAL-27): the patient opens the conversation with the practitioner
// they had an appointment with, the practitioner follows it on the socket and receives the patient's message
// live, answers on the socket, and the patient reads both in the page; a stranger and the admin see nothing, and
// after the patient's revocation the practitioner's socket closes and the page is refused.
func TestNew_ServesTheChatBetweenThePatientAndTheirPractitioner(t *testing.T) {
	c := newConsultationApp(t)
	c.book(t, wallNow().Add(-48*time.Hour))
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)

	rec := c.d.patient.do(http.MethodPost, "/conversations", `{"organization_id":"`+inviteClinic+`","practitioner_role_id":"`+c.d.roleID+`"}`)
	var conv struct {
		ID           string
		Participants []struct{ ID, Role string }
	}
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &conv) != nil || len(conv.Participants) != 2 {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	path := "/conversations/" + conv.ID
	expectHTTP(t, c.d.practitioner, http.MethodGet, path, http.StatusOK, `"id":"`+conv.ID+`"`)
	expectHTTP(t, c.stranger, http.MethodGet, path, http.StatusForbidden)
	expectHTTP(t, c.d.admin, http.MethodGet, path, http.StatusForbidden)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + path + "/events"
	socket, res, err := websocket.DefaultDialer.DialContext(t.Context(), url, http.Header{"Origin": {srv.URL}, "Authorization": {"Bearer " + c.d.practitioner.bearer}})
	if err != nil {
		t.Fatalf("dial = %v", err)
	}
	_ = res.Body.Close()
	defer func() { _ = socket.Close() }()
	_ = socket.SetReadDeadline(wallNow().Add(10 * time.Second))

	sent := wallNow()
	rec = c.d.patient.do(http.MethodPost, path+"/messages", `{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"Synthetic placeholder question"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send = %d %s", rec.Code, rec.Body)
	}
	if env := readEnvelope(t, socket); env.Type != communication.MessageType || !strings.Contains(string(env.Payload), `"seq":1`) {
		t.Fatalf("the practitioner's socket got %s %s", env.Type, env.Payload)
	}
	t.Logf("delivered in %v", wallNow().Sub(sent))
	if again := c.d.patient.do(http.MethodPost, path+"/messages", `{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"again"}`); again.Code != http.StatusOK {
		t.Errorf("a repeat = %d, want 200", again.Code)
	}

	answer, _ := wire.Encode(communication.SendType, communication.FrameVersion, communication.SendCommand{
		ClientMessageID: "c2000000-0000-4000-8000-0000000000c2", Body: "Synthetic placeholder answer",
	})
	if err := socket.WriteMessage(websocket.TextMessage, answer); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for range 2 { // the ack and the live frame, in either order
		got[readEnvelope(t, socket).Type] = true
	}
	if !got[communication.SendAckType] || !got[communication.MessageType] {
		t.Errorf("after the socket's send: %v", got)
	}
	body := expectHTTP(t, c.d.patient, http.MethodGet, path+"/messages", http.StatusOK, "Synthetic placeholder answer", "Synthetic placeholder question")
	if strings.Index(body, "answer") > strings.Index(body, "question") {
		t.Errorf("the page is not newest first: %s", body)
	}
	expectHTTP(t, c.d.patient, http.MethodGet, "/conversations", http.StatusOK, `"unread":1`, `"last_seq":2`) // the answer is unread

	expectHTTP(t, c.d.patient, http.MethodPost, "/consents/"+c.consent+"/revoke", http.StatusOK)
	_ = socket.SetReadDeadline(wallNow().Add(2 * communication.CheckEvery))
	for {
		if _, _, err := socket.ReadMessage(); err != nil {
			if ce, ok := err.(*websocket.CloseError); !ok || ce.Code != 4404 {
				t.Errorf("after the revocation the socket ended with %v, want close 4404", err)
			}
			break
		}
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, path+"/messages", http.StatusNotFound)
	expectHTTP(t, c.d.patient, http.MethodGet, path+"/messages", http.StatusOK, "Synthetic placeholder question")
	if strings.Contains(c.log.String(), "Synthetic placeholder") {
		t.Error("the log holds a message's text")
	}
}

func readEnvelope(t *testing.T, conn *websocket.Conn) wire.Envelope {
	t.Helper()
	_, b, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	env, err := wire.Decode(b)
	if err != nil {
		t.Fatalf("frame %s: %v", b, err)
	}
	return env
}
