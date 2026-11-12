package api_test

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
)

// keys are the JSON keys v encodes to.
func keys(t *testing.T, v any) []string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(m))
}

// The chat's schemas (docs/plans/chat.md "HTTP") are what the Go types encode: every key of a value with every
// field set, and as required exactly the keys of a value with only what is always there. The clients are generated
// from them (make gen-clients).
func TestChat_SchemasMatchTheGoTypes(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	id := "00000000-0000-4000-8000-000000000001"
	full := communication.Message{SentAt: at, EditedAt: at, ID: id, ConversationID: id, SenderParticipantID: id, ClientMessageID: id,
		Body: "x", SenderRole: communication.RolePatient, Kind: communication.KindText, Seq: 1, VersionID: 2}
	system := communication.Message{SentAt: at, ID: id, ConversationID: id, ClientMessageID: id, SenderRole: communication.RoleSystem,
		Kind: communication.KindSystem, SystemCode: "participant_added", SystemArgs: map[string]string{"practitioner_role_id": id}, Seq: 2, VersionID: 1}
	part := communication.Participant{AddedAt: at, ID: id, PatientID: id, PractitionerRoleID: id, OrganizationID: id, Role: communication.RolePatient,
		Status: communication.StatusActive, Reason: communication.ReasonHandover}
	from := int64(1)
	conv := communication.Conversation{Created: at, LastSentAt: at, ID: id, OrganizationID: id, PatientID: id, PractitionerRoleID: id,
		EncounterID: id, ParentID: id, Participants: []communication.Participant{part}}
	for _, tt := range []struct {
		name          string
		full, minimal any
		also          []string // keys of the other variant (a system message's `system`)
	}{
		{"ChatMessage", full, communication.Message{}, []string{"system"}},
		{"ChatMessage", system, communication.Message{}, []string{"body", "edited_at", "sender_participant_id"}},
		{"Conversation", conv, communication.Conversation{}, nil},
		{"ConversationParticipant", part, communication.Participant{}, nil},
		{"MessageRevision", communication.Revision{}, communication.Revision{}, nil},
		{"SendAck", communication.SendAck{}, communication.SendAck{}, nil},
		{"SendMessageRequest", communication.SendCommand{}, communication.SendCommand{}, nil},
		{"ChatCommandError", communication.CommandError{ID: "x"}, communication.CommandError{}, nil},
		{"AddParticipantRequest", communication.AddParticipant{FromSeq: &from}, communication.AddParticipant{}, nil},
		{"ChatInvitation", communication.Invitation{}, communication.Invitation{}, nil},
	} {
		sc, ok := s[tt.name]
		if !ok {
			t.Errorf("%s: no schema", tt.name)
			continue
		}
		props := slices.Sorted(maps.Keys(sc.Properties))
		if want := slices.Sorted(slices.Values(append(keys(t, tt.full), tt.also...))); !slices.Equal(props, want) {
			t.Errorf("%s: properties %v, Go encodes %v", tt.name, props, want)
		}
		required := slices.Sorted(slices.Values(sc.Required))
		minimal := keys(t, tt.minimal)
		if tt.name == "ChatMessage" { // the zero message is a text: its body is present but not required of a system message
			minimal = slices.DeleteFunc(minimal, func(k string) bool { return k == "body" })
		}
		if !slices.Equal(required, minimal) {
			t.Errorf("%s: required %v, Go always encodes %v", tt.name, required, minimal)
		}
	}
	if got := s["ChatMessage"].Properties["sender_role"].Enum; !slices.Equal(got, []string{"patient", "practitioner", "system"}) {
		t.Errorf("sender_role enum = %v", got)
	}
	reasons := []string{string(communication.ReasonSecondOpinion), string(communication.ReasonSpecialistReview), string(communication.ReasonHandover)}
	for name, want := range map[string][]string{
		"ConversationParticipant.status": {string(communication.StatusActive), string(communication.StatusPendingConsent)},
		"ConversationParticipant.reason": reasons, "AddParticipantRequest.reason": reasons, "ChatInvitation.reason": reasons,
		"AddParticipantRequest.history":   {communication.HistoryFull, communication.HistoryFrom},
		"ChatInvitationDecision.decision": {string(communication.DecisionGrant), string(communication.DecisionDecline)},
	} {
		schema, prop, _ := strings.Cut(name, ".")
		if got := s[schema].Properties[prop].Enum; !slices.Equal(got, want) {
			t.Errorf("%s enum = %v, want %v", name, got, want)
		}
	}
	for frame, typ := range map[string]string{"SendAckFrame": communication.SendAckType, "ChatCommandErrorFrame": communication.CommandErrorType, "ChatCommand": communication.SendType} {
		if got := s[frame].Properties["type"].Const; got != typ {
			t.Errorf("%s type = %v, want %s", frame, got, typ)
		}
	}
	if got := s["ChatMessageFrame"].Properties["type"].Enum; !slices.Equal(got, []string{communication.MessageType, communication.MessageEditedType}) {
		t.Errorf("ChatMessageFrame type = %v", got)
	}
}
