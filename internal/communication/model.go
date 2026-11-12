// Package communication is the chat between a patient and their clinic's doctors (docs/plans/chat.md, ADR 0012):
// conversations, their participants and their messages, each message a FHIR Communication. Messages are part of
// the medical record: never deleted, an edit keeps the earlier body.
package communication

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Role is who takes part in a conversation or wrote a message.
type Role string

// The roles; RoleSystem writes only system messages.
const (
	RolePatient      Role = "patient"
	RolePractitioner Role = "practitioner"
	RoleSystem       Role = "system"
)

// Kind is what a message is.
type Kind string

// The kinds: a participant's text, or a system message (who joined, who left; SAL-28) the apps render from their
// catalogs by its code.
const (
	KindText   Kind = "text"
	KindSystem Kind = "system"
)

// The limits of a message and a page.
const (
	MaxBody     = 4000 // characters, after trimming
	PageSize    = 50   // a page of messages when the client names no limit
	MaxPageSize = 100
	ListSize    = 50 // a page of conversations
)

// Conversation is a chat about one patient in one clinic, with its treating practitioner.
type Conversation struct {
	Created            time.Time `json:"created_at"`
	LastSentAt         time.Time `json:"-"` // last_sent_at, zero before the first message
	ID                 string    `json:"id"`
	OrganizationID     string    `json:"organization_id"`
	PatientID          string    `json:"patient_id"`
	PractitionerRoleID string    `json:"practitioner_role_id"` // the treating practitioner
	EncounterID        string    `json:"encounter_id,omitempty"`
	ParentID           string    `json:"parent_id,omitempty"` // a doctor-only side thread's conversation (SAL-28)
	CreatedBy          string    `json:"-"`                   // the account that opened it
	// Participants are the active ones; set by Get and Open, not by List.
	Participants []Participant `json:"participants,omitempty"`
	LastSeq      int64         `json:"last_seq"`
	// Unread is the caller's: messages after their read position.
	Unread    int64 `json:"unread"`
	StaffOnly bool  `json:"staff_only"`
}

// MarshalJSON adds last_sent_at, absent before the first message.
func (c Conversation) MarshalJSON() ([]byte, error) {
	type plain Conversation
	var last *time.Time
	if !c.LastSentAt.IsZero() {
		last = &c.LastSentAt
	}
	return json.Marshal(struct {
		LastSentAt *time.Time `json:"last_sent_at,omitempty"`
		plain
	}{last, plain(c)})
}

// Participant is the patient or a practitioner (by their role) in a conversation.
type Participant struct {
	AddedAt            time.Time `json:"added_at"`
	ID                 string    `json:"id"`
	ConversationID     string    `json:"-"`
	PatientID          string    `json:"patient_id,omitempty"`
	PractitionerRoleID string    `json:"practitioner_role_id,omitempty"`
	AddedBy            string    `json:"-"`
	Role               Role      `json:"role"`
	// HistoryFromSeq: the participant reads the messages after this number; 0 for the whole history (SAL-28).
	HistoryFromSeq int64 `json:"history_from_seq"`
	ReadSeq        int64 `json:"read_seq"` // the participant's own read position
}

// Message is a FHIR Communication: sent, sender, payload.contentString; or a system message.
type Message struct {
	SentAt              time.Time         `json:"sent_at"`
	EditedAt            time.Time         `json:"-"`
	SystemArgs          map[string]string `json:"-"` // opaque ids only, for the system message's code
	ID                  string            `json:"id"`
	ConversationID      string            `json:"conversation_id"`
	OrganizationID      string            `json:"-"`
	PatientID           string            `json:"-"`
	SenderParticipantID string            `json:"sender_participant_id,omitempty"`
	SenderAccountID     string            `json:"-"` // accountability; not shown to the other participants
	ClientMessageID     string            `json:"client_message_id"`
	Body                string            `json:"-"`
	SystemCode          string            `json:"-"`
	SenderRole          Role              `json:"sender_role"`
	Kind                Kind              `json:"kind"`
	Seq                 int64             `json:"seq"`
	VersionID           int32             `json:"version_id"`
	StaffOnly           bool              `json:"-"`
}

// systemJSON is a system message's code and arguments (SystemMessage in api/openapi.yaml).
type systemJSON struct {
	Args map[string]string `json:"args,omitempty"`
	Code string            `json:"code"`
}

// MarshalJSON is the ChatMessage schema: the message, its cursor, and its body or system part.
func (m Message) MarshalJSON() ([]byte, error) {
	type plain Message
	out := struct {
		EditedAt *time.Time  `json:"edited_at,omitempty"`
		Body     *string     `json:"body,omitempty"`
		System   *systemJSON `json:"system,omitempty"`
		Cursor   string      `json:"cursor"`
		plain
	}{Cursor: m.Cursor().String(), plain: plain(m)}
	if !m.EditedAt.IsZero() {
		out.EditedAt = &m.EditedAt
	}
	if m.Kind == KindSystem {
		out.System = &systemJSON{Code: m.SystemCode, Args: m.SystemArgs}
	} else {
		out.Body = &m.Body
	}
	return json.Marshal(out)
}

// Cursor is a message's place: its time and its number.
func (m Message) Cursor() Cursor { return Cursor{SentAt: m.SentAt, Seq: m.Seq} }

// Revision is an earlier body of an edited message.
type Revision struct {
	EditedAt  time.Time `json:"edited_at"` // when this body was written: the message's sending or an earlier edit
	Body      string    `json:"body"`
	VersionID int32     `json:"version_id"`
}

// Cursor is a place in a conversation's messages, opaque to the client: the order is by (SentAt, Seq), which is
// the order by Seq (ADR 0012). The zero Cursor is before the first message.
type Cursor struct {
	SentAt time.Time
	Seq    int64
}

// String is the cursor as the client sees it: base64url of "<unix microseconds>.<seq>".
func (c Cursor) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.SentAt.UnixMicro(), 10) + "." + strconv.FormatInt(c.Seq, 10)))
}

// ParseCursor parses a cursor from String; ErrInvalidCursor otherwise.
func ParseCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || s == "" {
		return Cursor{}, ErrInvalidCursor
	}
	us, seq, ok := strings.Cut(string(raw), ".")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	t, err1 := strconv.ParseInt(us, 10, 64)
	n, err2 := strconv.ParseInt(seq, 10, 64)
	if err1 != nil || err2 != nil || n < 0 || t < 0 {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{SentAt: time.UnixMicro(t).UTC(), Seq: n}, nil
}

// Page is a page of messages, newest first, and the cursor of the older page (nil at the start).
type Page struct {
	Next     *Cursor
	Messages []Message
}

// ListCursor is a place in the caller's conversations, ordered by their latest activity (the last message, or the
// opening), newest first.
type ListCursor struct {
	At time.Time
	ID string
}

// Open is a request to open a conversation: staff name the patient, a patient the clinic and practitioner role.
type Open struct {
	PatientID          string `json:"patient_id"`
	OrganizationID     string `json:"organization_id"`
	PractitionerRoleID string `json:"practitioner_role_id"`
	EncounterID        string `json:"encounter_id"`
}

// normalizeBody trims body and checks its length (1 to MaxBody characters).
func normalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if !utf8.ValidString(body) || body == "" || utf8.RuneCountInString(body) > MaxBody || strings.ContainsRune(body, 0) {
		return "", invalid("body")
	}
	return body, nil
}

// ChannelKey is the Redis pub/sub channel of the conversation's live frames (ADR 0012).
func ChannelKey(conversationID string) string { return "chat:" + conversationID }

// The wire frames of the conversation's socket (docs/plans/chat.md "Go").
const (
	MessageType       = "message"
	MessageEditedType = "message_edited"
	SendAckType       = "send_ack"
	SendType          = "send"
	FrameVersion      = 1
)

// SendCommand is the payload of the client's send frame.
type SendCommand struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

// SendAck is the payload of the answer to a send: the client's id and the message as stored.
type SendAck struct {
	ClientMessageID string  `json:"client_message_id"`
	Message         Message `json:"message"`
}

// validUUID reports whether s is a UUID in its canonical text form, as Postgres would take it.
func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F'):
			return false
		}
	}
	return true
}
