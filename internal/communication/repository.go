package communication

import (
	"context"
	"time"
)

// Repository stores conversations, participants and messages (migration 00041). Every method runs in the
// caller's transaction and scope; the row-level security of the scope decides what it sees.
type Repository interface {
	// SetPatient, SetClinic and SetConversation scope the transaction (app.patient_id, app.organization_id,
	// app.conversation_id): a conversation's messages are seen only with it set, in a scope that sees it.
	SetPatient(ctx context.Context, id string) error
	SetClinic(ctx context.Context, id string) error
	SetConversation(ctx context.Context, id string) error
	// SetPractitionerRole sets app.practitioner_role_id, the signed-in practitioner's role: as a doctor invited from
	// another clinic, they see the conversations they were granted (00044).
	SetPractitionerRole(ctx context.Context, id string) error

	// CreateConversation stores c with its participants unless the patient, role and encounter have their one
	// conversation already; it returns the stored conversation (with its participants) and whether it is new.
	CreateConversation(ctx context.Context, c Conversation, participants []Participant) (Conversation, bool, error)
	// Conversation returns the conversation, locked for the transaction when lock is set (a send's order);
	// ErrNotFound when the scope does not show it.
	Conversation(ctx context.Context, id string, lock bool) (Conversation, error)
	// Participants returns the conversation's active participants.
	Participants(ctx context.Context, conversationID string) ([]Participant, error)
	// Conversations returns a page of the conversations the patient or the practitioner role takes part in, with
	// that participant's unread count, newest activity first.
	Conversations(ctx context.Context, f ListFilter) ([]Conversation, error)

	// Messages returns up to limit messages before the cursor (nil: the newest), newest first, of numbers above
	// fromSeq.
	Messages(ctx context.Context, conversationID string, before *Cursor, fromSeq int64, limit int) ([]Message, error)
	// After returns up to limit messages after the cursor, oldest first, of numbers above fromSeq.
	After(ctx context.Context, conversationID string, after Cursor, fromSeq int64, limit int) ([]Message, error)
	// Message returns one message of the conversation; ErrNotFound otherwise.
	Message(ctx context.Context, conversationID, id string) (Message, error)
	// MessageByClientID returns the participant's message of that client id in the conversation sent since since;
	// ErrNotFound when there is none. Under the conversation's lock it decides whether a send is a repeat; since
	// bounds the partitions it reads.
	MessageByClientID(ctx context.Context, conversationID, participantID, clientMessageID string, since time.Time) (Message, error)
	// InsertMessage stores m and returns it with its id.
	InsertMessage(ctx context.Context, m Message) (Message, error)
	// Advance sets the conversation's last message number and time.
	Advance(ctx context.Context, conversationID string, seq int64, sentAt time.Time) error
	// MarkRead moves the participant's read position to seq; it never goes back.
	MarkRead(ctx context.Context, participantID string, seq int64) error
	// UpdateBody stores m's new body, version and edit time if the stored version is the one before m's
	// (the trigger keeps the earlier body); ErrVersionConflict otherwise.
	UpdateBody(ctx context.Context, m Message) error
	// Revisions returns the earlier bodies of the message, oldest first.
	Revisions(ctx context.Context, conversationID, id string) ([]Revision, error)
	// Participant returns the participant id, pending or active, where the scope shows its conversation;
	// ErrNotFound otherwise.
	Participant(ctx context.Context, id string) (Participant, error)
	// InsertParticipant stores p, who has read the history before them (read_seq is their history_from_seq), and
	// returns it with its id.
	InsertParticipant(ctx context.Context, p Participant) (Participant, error)
	// ActivateParticipant makes the pending participant id active; ErrNotFound when it is not pending or not there.
	ActivateParticipant(ctx context.Context, id string) error
	// RemoveParticipant ends the participant id at at; a practitioner leaves the conversation's side threads with it.
	// ErrNotFound when it was removed already.
	RemoveParticipant(ctx context.Context, id string, at time.Time) error
	// CreateThread stores the side thread c of its parent with its participants and returns it with them.
	CreateThread(ctx context.Context, c Conversation, participants []Participant) (Conversation, error)
	// Threads returns the side threads of the conversation the practitioner role takes part in, with that
	// participant's unread count, newest first.
	Threads(ctx context.Context, parentID, roleID string) ([]Conversation, error)
	// Invitations returns the patient's pending invitations, oldest first.
	Invitations(ctx context.Context, patientID string) ([]Invitation, error)
	// AddPartitions creates the monthly partitions of the messages from the month of from for months months,
	// skipping those that exist (communication_add_partitions).
	AddPartitions(ctx context.Context, from time.Time, months int) error
}

// ListFilter selects the caller's conversations: by PatientID for a patient, by PractitionerRoleID for a
// practitioner (both participants, active).
type ListFilter struct {
	Before             *ListCursor
	PatientID          string
	PractitionerRoleID string
	Limit              int
}

// Publisher puts a frame on a conversation's live channel (ChannelKey), to every socket of it in every api
// process (pkg/redis.Hub).
type Publisher interface {
	Publish(ctx context.Context, channel string, frame []byte) error
}
