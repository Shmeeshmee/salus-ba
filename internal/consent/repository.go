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
	// Grant stores an active consent of the patient with the clinic, granted at, and returns it; ErrActive
	// when one is active already, ErrClinicNotFound when the clinic does not exist (which aborts the
	// transaction).
	Grant(ctx context.Context, patientID, organizationID string, at time.Time) (Consent, error)
	// ByID returns the consent, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (Consent, error)
	// Revoke makes the active consent id inactive, revoked at, and returns it with the next version;
	// ErrInactive when no active consent has that id.
	Revoke(ctx context.Context, id string, at time.Time) (Consent, error)
	// ByPatient returns the patient's consents, the latest granted first.
	ByPatient(ctx context.Context, patientID string) ([]Consent, error)
	// State returns whether the patient has an active consent with the clinic and when the latest one was
	// revoked, as the transaction's scope shows them.
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}
