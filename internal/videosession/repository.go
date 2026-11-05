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
	// appointment's session: two first requests at once share one.
	Open(ctx context.Context, s Session) (Session, error)
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
	// Participants returns the identities in the room; none when it does not exist.
	Participants(ctx context.Context, room string) ([]string, error)
}
