package chargeitemdefinition

import (
	"context"
	"time"
)

// Repository stores the price list. Every read sees only the clinic set by SetOrganization, or the active services of
// listed clinics after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// List returns the services of the clinic org that the scope shows, by name, with their translations.
	List(ctx context.Context, org string) ([]Definition, error)
	// Create stores d with its translations in the clinic d.OrganizationID at version 1 and returns it with its id.
	Create(ctx context.Context, d Definition, at time.Time) (Definition, error)
	// Update replaces the service d.ID of the scope's clinic, at version d.VersionID, and its translations, and
	// returns it at its new version. ErrNotFound for none in the scope, ErrConflict for another version.
	Update(ctx context.Context, d Definition, at time.Time) (Definition, error)
}
