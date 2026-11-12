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
	// ByID returns the consent of any kind (an active history consent with the visits it lists now), or ErrNotFound (also for one outside the scope, or an id that is
	// not a UUID).
	ByID(ctx context.Context, id string) (Consent, error)
	// Revoke makes the active consent id of the kind inactive, revoked at, and returns it with the next version;
	// ErrInactive when no active consent of the kind has that id.
	Revoke(ctx context.Context, kind Kind, id string, at time.Time) (Consent, error)
	// RevokeTranscription makes the patient's active transcription consent with the clinic inactive, revoked at,
	// and returns it; ErrInactive when there is none.
	RevokeTranscription(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error)
	// RevokeHistory makes the patient's active history consent with the clinic inactive, revoked at, and returns it;
	// ErrInactive when there is none.
	RevokeHistory(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error)
	// ActiveHistory returns the patient's active history consent with the clinic with the visits it lists now, or
	// ErrNotFound.
	ActiveHistory(ctx context.Context, patientID, organizationID string) (Consent, error)
	// GrantHistory stores an active history consent of the patient with the clinic, granted at, listing the
	// encounters (added at), and returns it; ErrActive when one is active already; ErrClinicNotFound for an unknown
	// clinic and ErrInvalidVisit when row-level security refuses an encounter (both abort the transaction).
	GrantHistory(ctx context.Context, patientID, organizationID string, encounterIDs []string, at time.Time) (Consent, error)
	// ChangeHistory adds the encounters add to the active history consent id at version and removes remove from it, at,
	// and returns it at the next version with the visits it lists now; ErrVersionConflict when it is not active at
	// that version, ErrInvalidVisit as GrantHistory.
	ChangeHistory(ctx context.Context, id string, version int32, add, remove []string, at time.Time) (Consent, error)
	// ByPatient returns the patient's consents of every kind, the latest granted first; an active history
	// consent with the visits it lists now.
	ByPatient(ctx context.Context, patientID string) ([]Consent, error)
	// State returns whether the patient has an active care consent with the clinic, when the latest one was
	// revoked, and whether their transcription consent there is active, as the transaction's scope shows them.
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}
