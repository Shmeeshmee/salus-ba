package consent

import (
	"context"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Repository stores consents. Row-level security shows a transaction only the consents of the patient set by
// SetPatient, or, read-only, those given to the clinic set by SetClinic.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// Grant stores an active consent of the kind of the patient with the clinic, granted at, and returns it;
	// ErrActive when one of the kind is active already, ErrClinicNotFound when the clinic does not exist (which
	// aborts the transaction).
	Grant(ctx context.Context, kind Kind, patientID, organizationID string, at time.Time) (Consent, error)
	// ByID returns the consent of either kind, or ErrNotFound (also for one outside the scope, or an id that is
	// not a UUID).
	ByID(ctx context.Context, id string) (Consent, error)
	// Revoke makes the active consent id of the kind inactive, revoked at, and returns it with the next version;
	// ErrInactive when no active consent of the kind has that id.
	Revoke(ctx context.Context, kind Kind, id string, at time.Time) (Consent, error)
	// RevokeTranscription makes the patient's active transcription consent with the clinic inactive, revoked at,
	// and returns it; ErrInactive when there is none.
	RevokeTranscription(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error)
	// ByPatient returns the patient's consents of both kinds, the latest granted first.
	ByPatient(ctx context.Context, patientID string) ([]Consent, error)
	// State returns whether the patient has an active care consent with the clinic, when the latest one was
	// revoked, and whether their transcription consent there is active, as the transaction's scope shows them.
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)

	// GrantChat stores the patient's active chat consent for g, granted at, and returns it; an error when one for the
	// conversation and role is active already (a unique violation, which aborts the transaction).
	GrantChat(ctx context.Context, patientID string, g ChatGrant, at time.Time) (Consent, error)
	// RevokeChat makes the active chat consent for the conversation and role inactive, revoked at, and returns it;
	// ErrInactive when there is none in the scope.
	RevokeChat(ctx context.Context, conversationID, roleID string, at time.Time) (Consent, error)
	// ChatGranted reports whether the scope shows an active chat consent for the conversation and role.
	ChatGranted(ctx context.Context, conversationID, roleID string) (bool, error)
}
