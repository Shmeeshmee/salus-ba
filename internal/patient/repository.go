package patient

import (
	"context"
	"time"
)

// Repository stores profiles. Every read sees only the patient whose account is set by SetAccount, or, read
// only, the patients with an active consent for the clinic set by SetClinic (row-level security).
type Repository interface {
	// SetAccount scopes the rest of the transaction in ctx to the patient of the account id.
	SetAccount(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// ByAccount returns the profile of the account, or ErrNotFound.
	ByAccount(ctx context.Context, accountID string) (Patient, error)
	// ByID returns the profile, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (Patient, error)
	// Create stores p as the profile of p.AccountID, last updated at, and returns it; ErrConflict when the
	// account has one already.
	Create(ctx context.Context, p Patient, at time.Time) (Patient, error)
	// Update replaces the fields of the profile of p.AccountID when its version is p.VersionID, and returns it
	// with the next version; ErrConflict when no profile has that version.
	Update(ctx context.Context, p Patient, at time.Time) (Patient, error)
}
