package chargeitemdefinition

import "context"

// Repository stores the price list. Every read sees only the clinic set by SetOrganization, or the active services of
// listed clinics after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// List returns the services the scope shows, by name, with their translations.
	List(ctx context.Context) ([]Definition, error)
}
