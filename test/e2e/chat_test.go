//go:build e2e

package e2e_test

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TestChat_ThePatientAndTheDoctorChat (SAL-31): after a booking, the patient opens the conversation with their
// doctor and writes; the doctor, on the conversation's socket as the web app follows it, receives the message
// live and answers on the socket; the patient reads the answer in the page and has it unread. Once the
// conversation is long, the doctor asks for the AI summary (the fake), which the patient may not.
func TestChat_ThePatientAndTheDoctorChat(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.bookFirstTime(clinic)
	id := openChat(t, p.api, clinic.id, publicRole(t, clinic.id))

	doctor := followChat(t, clinic.practitioner, id)
	p.must(http.StatusCreated, http.MethodPost, "/conversations/"+id+"/messages",
		map[string]string{"client_message_id": newUUID(t), "body": "Synthetic placeholder question"}, nil)
	if m := doctor.next(t, "message"); m.Payload.Seq != 1 || m.Payload.Body != "Synthetic placeholder question" || m.Payload.SenderRole != "patient" {
		t.Fatalf("the doctor's socket got %+v, want the patient's message 1", m.Payload)
	}

	doctor.send(t, newUUID(t), "Synthetic placeholder answer")
	got := map[string]int64{}
	for range 2 { // the ack and the live frame, in either order
		f := doctor.next(t, "")
		got[f.Type] = f.Payload.Seq
	}
	if got["send_ack"] != 0 || got["message"] != 2 {
		t.Errorf("after the doctor's send the socket got %v, want the ack and message 2", got)
	}
	var page chatPage
	p.must(http.StatusOK, http.MethodGet, "/conversations/"+id+"/messages", nil, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "Synthetic placeholder answer" || page.Messages[0].SenderRole != "practitioner" {
		t.Fatalf("the patient's page = %+v, want the answer first", page.Messages)
	}
	var list struct {
		Conversations []struct {
			ID     string `json:"id"`
			Unread int64  `json:"unread"`
		} `json:"conversations"`
	}
	p.must(http.StatusOK, http.MethodGet, "/conversations", nil, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].ID != id || list.Conversations[0].Unread != 1 {
		t.Errorf("the patient's conversations = %+v, want this one with the answer unread", list.Conversations)
	}

	for i := range 48 {
		p.must(http.StatusCreated, http.MethodPost, "/conversations/"+id+"/messages",
			map[string]string{"client_message_id": newUUID(t), "body": fmt.Sprintf("Synthetic placeholder %d", i)}, nil)
	}
	var summary struct {
		Summary     string `json:"summary"`
		UpToSeq     int64  `json:"up_to_seq"`
		AIGenerated bool   `json:"ai_generated"`
	}
	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/conversations/"+id+"/summary", map[string]string{"lang": "en"}, &summary)
	if summary.Summary != "[en] Placeholder summary of 50 messages" || summary.UpToSeq != 50 || !summary.AIGenerated {
		t.Errorf("the doctor's summary = %+v", summary)
	}
	if status, code := p.refusal(http.MethodPost, "/conversations/"+id+"/summary", nil); status != http.StatusForbidden || code != "forbidden" {
		t.Errorf("the patient's summary = %d %s, want 403 forbidden", status, code)
	}
}

// TestChat_TheDoctorAddsAColleagueAndThePatientSeesIt (SAL-31, on SAL-28's backend): the doctor invites a doctor of
// another clinic for a second opinion; the patient finds the invitation, grants it, and sees in their conversation
// who was invited and that they joined (system messages); the colleague then reads the conversation.
func TestChat_TheDoctorAddsAColleagueAndThePatientSeesIt(t *testing.T) {
	clinic, other := newBookableClinic(t), newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.bookFirstTime(clinic)
	id := openChat(t, p.api, clinic.id, publicRole(t, clinic.id))
	p.must(http.StatusCreated, http.MethodPost, "/conversations/"+id+"/messages",
		map[string]string{"client_message_id": newUUID(t), "body": "Synthetic placeholder question"}, nil)

	colleague := publicRole(t, other.id)
	var added struct {
		Status string `json:"status"`
	}
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/conversations/"+id+"/participants", map[string]string{
		"practitioner_role_id": colleague, "organization_id": other.id, "reason": "second_opinion", "history": "full",
	}, &added)
	if added.Status != "pending_consent" {
		t.Errorf("a doctor of another clinic is %q, want pending_consent", added.Status)
	}
	if got := other.practitioner.do(http.MethodGet, "/conversations/"+id+"/messages", nil, nil); got != http.StatusNotFound && got != http.StatusForbidden {
		t.Errorf("the colleague reads before the patient's consent: %d", got)
	}

	var invitations struct {
		Invitations []struct {
			ID                 string `json:"id"`
			ConversationID     string `json:"conversation_id"`
			PractitionerRoleID string `json:"practitioner_role_id"`
			Reason             string `json:"reason"`
		} `json:"invitations"`
	}
	p.must(http.StatusOK, http.MethodGet, "/me/chat-invitations", nil, &invitations)
	if len(invitations.Invitations) != 1 || invitations.Invitations[0].ConversationID != id || invitations.Invitations[0].Reason != "second_opinion" {
		t.Fatalf("the patient's invitations = %+v, want the one of this conversation", invitations.Invitations)
	}
	inv := invitations.Invitations[0]
	p.must(http.StatusOK, http.MethodPost, "/me/chat-invitations/"+inv.ID+"/decision", map[string]string{"decision": "grant"}, nil)

	var page chatPage
	p.must(http.StatusOK, http.MethodGet, "/conversations/"+id+"/messages", nil, &page)
	codes := map[string]bool{}
	for _, m := range page.Messages {
		if m.Kind == "system" && m.System.Args["practitioner_role_id"] == colleague {
			codes[m.System.Code] = true
		}
	}
	if !codes["participant_invited"] || !codes["consent_granted"] {
		t.Errorf("the patient's conversation shows %v, want the invitation and the consent", codes)
	}
	other.practitioner.must(http.StatusOK, http.MethodGet, "/conversations/"+id+"/messages", nil, &page)
	if !strings.Contains(fmt.Sprint(page.Messages), "Synthetic placeholder question") {
		t.Errorf("the colleague reads %d messages without the patient's question", len(page.Messages))
	}
}

type chatMessage struct {
	System struct {
		Args map[string]string `json:"args"`
		Code string            `json:"code"`
	} `json:"system"`
	Body       string `json:"body"`
	SenderRole string `json:"sender_role"`
	Kind       string `json:"kind"`
	Seq        int64  `json:"seq"`
}

type chatPage struct {
	Messages []chatMessage `json:"messages"`
}

// openChat opens (or finds) the patient's conversation with the clinic's role and returns its id.
func openChat(t *testing.T, patient *api, clinic, role string) string {
	t.Helper()
	var conv struct {
		ID string `json:"id"`
	}
	patient.must(http.StatusCreated, http.MethodPost, "/conversations", map[string]string{"organization_id": clinic, "practitioner_role_id": role}, &conv)
	return conv.ID
}

// chatSocket is a client of the conversation's socket (GET /conversations/{id}/events), with a bearer token and
// no Origin, as the mobile app opens it.
type chatSocket struct {
	conn *websocket.Conn
}

// chatFrame is a frame of the socket; for a send_ack, Payload holds none of a message's fields (they are under
// its message).
type chatFrame struct {
	Type    string      `json:"type"`
	Payload chatMessage `json:"payload"`
}

func followChat(t *testing.T, a *api, id string) *chatSocket {
	t.Helper()
	u := strings.Replace(a.base, "http", "ws", 1) + "/conversations/" + id + "/events"
	conn, resp, err := (&websocket.Dialer{HandshakeTimeout: 10 * time.Second}).DialContext(t.Context(), u,
		http.Header{"Authorization": {"Bearer " + a.bearer}})
	if err != nil {
		t.Fatalf("open the conversation's socket: %v", err)
	}
	_ = resp.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })
	return &chatSocket{conn: conn}
}

// next is the next frame (of the type, when typ is set; others are skipped), within 10 s.
func (s *chatSocket) next(t *testing.T, typ string) chatFrame {
	t.Helper()
	_ = s.conn.SetReadDeadline(clock.System{}.Now().Add(10 * time.Second))
	for {
		_, b, err := s.conn.ReadMessage()
		if err != nil {
			t.Fatalf("read the conversation's socket: %v", err)
		}
		var f chatFrame
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatalf("frame %s: %v", b, err)
		}
		if typ == "" || f.Type == typ {
			return f
		}
	}
}

func (s *chatSocket) send(t *testing.T, clientID, body string) {
	t.Helper()
	frame, _ := json.Marshal(map[string]any{"type": "send", "version": 1, "payload": map[string]string{"client_message_id": clientID, "body": body}})
	if err := s.conn.WriteMessage(websocket.TextMessage, frame); err != nil {
		t.Fatalf("send on the conversation's socket: %v", err)
	}
}

// newUUID is a random version-4 UUID, a client's message id.
func newUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
