package organization

import (
	"context"
	"time"
)

// Repository stores clinics. Every read sees only the clinic set by SetOrganization (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// ByID returns the clinic, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (Organization, error)
	// Rename sets the clinic's name at version, moving it to the next version, last updated at at; or
	// returns ErrConflict when version is not the current one.
	Rename(ctx context.Context, id, name string, version int32, at time.Time) (Organization, error)
}
