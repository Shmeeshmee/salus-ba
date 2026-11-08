package videosession

import (
	"context"
	"time"
)

// Repository stores the sessions. Row-level security shows a transaction the sessions of the patient set by
// SetPatient, and of the clinic set by SetClinic those of its appointments with an active consent and, after
// a revocation, of those that started before it.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// Open stores s, created at s.CreatedAt, unless its appointment has a session already, and returns the
	// appointment's session and whether this call created it: two first requests at once share one.
	Open(ctx context.Context, s Session) (Session, bool, error)
	// SetRoom scopes the rest of the transaction in ctx to the session of the room (the webhook's scope).
	SetRoom(ctx context.Context, room string) error
	// SetAppointment narrows the clinic's scope set by SetClinic to the appointment id: it then also shows
	// that appointment's session once its practitioner joined, whatever the consent (the practitioner's read
	// of a consultation they were in), and the appointment itself (appointment's notice scope).
	SetAppointment(ctx context.Context, id string) error
	// ByID returns the session, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Session, error)
	// ByAppointment returns the appointment's session, or ErrNotFound (also outside the scope).
	ByAppointment(ctx context.Context, appointmentID string) (Session, error)
	// ClosingAfter returns the sessions in the scope (the patient's, set by SetPatient) of the clinic org whose
	// join window closes after after.
	ClosingAfter(ctx context.Context, org string, after time.Time) ([]Session, error)
	// ByRoom returns the session of the room, or ErrNotFound (also outside the scope).
	ByRoom(ctx context.Context, room string) (Session, error)
	// Joined records that the role joined the session id at at, updated now, unless that role's join is
	// recorded already or the session ended, and returns it and whether it changed.
	Joined(ctx context.Context, id string, role Role, at, now time.Time) (Session, bool, error)
	// Removed records, in the room's scope, why the patient was removed from the session id, at at, unless a
	// reason is recorded already.
	Removed(ctx context.Context, id string, reason Reason, at time.Time) error
	// End ends the session id at at for reason unless it has ended, and returns it and whether it changed.
	End(ctx context.Context, id string, reason EndReason, at time.Time) (Session, bool, error)
	// DeleteEndedBefore deletes the sessions that ended before before, and those never ended whose window
	// closed before it, in every clinic (the retention role), and returns how many.
	DeleteEndedBefore(ctx context.Context, before time.Time) (int64, error)
}

// Provider is the video server (LiveKit, repository_livekit.go). Errors carry no room, identity or token.
type Provider interface {
	// CreateRoom creates the room, or leaves it as it is when it exists.
	CreateRoom(ctx context.Context, r Room) error
	// Token returns a signed join token for g.
	Token(g Grant) (string, error)
	// RemoveParticipant disconnects the identity from the room; nothing when either is not there.
	RemoveParticipant(ctx context.Context, room, identity string) error
	// DeleteRoom closes the room and disconnects everyone in it; nothing when it does not exist.
	DeleteRoom(ctx context.Context, room string) error
	// Participants returns the identities of the people in the room, not an agent's; none when it does not exist.
	Participants(ctx context.Context, room string) ([]string, error)
	// ParseWebhook verifies the webhook's authorization (a JWT of our key pair carrying the body's SHA-256)
	// before it reads the body, and returns its event; ErrInvalidSignature when LiveKit did not sign it.
	ParseWebhook(body []byte, authorization string) (Event, error)
}

// Jobs enqueues the session's jobs in the transaction in ctx (River, repository_river.go).
type Jobs interface {
	// EnqueueWindowClose enqueues consultation_window_close for the session of the clinic org, to run at at.
	EnqueueWindowClose(ctx context.Context, sessionID, org string, at time.Time) error
}
