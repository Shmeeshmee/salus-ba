package practitioner

import "context"

// Repository stores practitioners. Every read sees only the clinic set by SetOrganization, or the active
// practitioners of listed clinics after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// PublicByIDs returns the active practitioners among ids that the scope shows, with their name, meta and
	// active flag only (no clinic); an id that is not a UUID matches none.
	PublicByIDs(ctx context.Context, ids []string) ([]Practitioner, error)
	// ByID returns the practitioner, or ErrNotFound (also for one outside the scope, or an id that is not a
	// UUID).
	ByID(ctx context.Context, id string) (Practitioner, error)
	// Create stores a new active practitioner of the clinic org and returns it; row-level security refuses a
	// clinic other than the one set.
	Create(ctx context.Context, org, givenName, familyName string) (Practitioner, error)
}
