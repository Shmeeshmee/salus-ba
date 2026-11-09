package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The practitioner dismisses a suggestion over the event stream (SAL-7): the status is stored and audited as
// theirs, and the suggestion with its new status comes on the stream, to every open view. A suggestion that is
// not the encounter's is answered with a command_error and changes nothing.
func TestNew_ThePractitionerDismissesASuggestionOverTheEventStream(t *testing.T) {
	s := newStreamApp(t)
	const id = "5a000000-0000-4000-8000-000000000001"
	if _, err := s.owner.Exec(t.Context(), `INSERT INTO suggestion (id, encounter_id, type, text, confidence, source_segment_ids)
		VALUES ($1, $2, 'question', 'Placeholder question to ask', 0.5, ARRAY[gen_random_uuid()])`, id, s.encounter); err != nil {
		t.Fatal(err)
	}
	conn, _, err := s.dialEvents(t, s.d.practitioner, "")
	if err != nil {
		t.Fatal(err)
	}
	send := func(id string, status suggestion.Status) {
		t.Helper()
		b, err := wire.Encode(suggestion.StatusMessageType, suggestion.StatusMessageVersion, suggestion.StatusChange{ID: id, Status: status})
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			t.Fatal(err)
		}
	}
	send("5a000000-0000-4000-8000-000000000009", suggestion.StatusDismissed)
	_ = conn.SetReadDeadline(wallNow().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if want := `{"type":"command_error","version":1,"payload":{"command":"suggestion_status","id":"5a000000-0000-4000-8000-000000000009","error":"not_found"}}`; err != nil || string(msg) != want {
		t.Fatalf("answer %s, %v; want %s", msg, err, want)
	}

	send(id, suggestion.StatusDismissed)
	f, err := next(t, conn)
	if err != nil {
		t.Fatal(err)
	}
	env, err := wire.Decode(f.envelope)
	var e suggestion.Event
	if err != nil || env.Type != suggestion.MessageType || json.Unmarshal(env.Payload, &e) != nil ||
		e.ID != id || e.Status != suggestion.StatusDismissed || e.Origin != suggestion.OriginAI {
		t.Fatalf("stream gave %s, %v; want the suggestion dismissed", f.envelope, err)
	}
	if n := s.count(t, `SELECT count(*) FROM suggestion WHERE id = $1 AND status = 'dismissed' AND version_id = 2`, id); n != 1 {
		t.Errorf("stored dismissed: %d, want 1", n)
	}
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE entity_name = 'suggestion' AND subtype_code = 'update' AND action = 'U'
		AND agent_account_id IS NOT NULL AND entity_reference = $1 AND outcome = 0`, "Encounter/"+s.encounter); n != 1 {
		t.Errorf("audited %d status changes, want 1", n)
	}
	if n := s.count(t, `SELECT count(*) FROM audit_event WHERE entity_name = 'suggestion' AND subtype_code = 'update' AND outcome = 4`); n != 1 {
		t.Errorf("audited %d refused changes, want 1", n)
	}
}
