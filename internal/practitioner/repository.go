package practitioner

import "context"

// Repository stores practitioners. Every read sees only the clinic set by SetOrganization (row-level
// security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// ByID returns the practitioner, or ErrNotFound (also for one outside the scope, or an id that is not a
	// UUID).
	ByID(ctx context.Context, id string) (Practitioner, error)
}
