package encounter

import (
	"context"
	"time"
)

// Repository stores the encounters. Row-level security shows a transaction the encounters of the patient set
// by SetPatient, and of the clinic set by SetClinic those of its appointments with an active consent and,
// after a revocation, of those that started before it.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// Start stores e in progress unless its appointment has an encounter already, and returns the
	// appointment's encounter and whether this call created it: two starts at once give one.
	Start(ctx context.Context, e Encounter) (Encounter, bool, error)
	// Finish finishes the appointment's encounter at at when it is in progress, and returns it and whether
	// it changed; nothing when there is none or it is finished.
	Finish(ctx context.Context, appointmentID string, at time.Time) (Encounter, bool, error)
	// ByID returns the encounter, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Encounter, error)
	// Search returns the encounters of the appointment and the patient of f, when set, that the scope shows.
	Search(ctx context.Context, f Filter) ([]Encounter, error)
}
