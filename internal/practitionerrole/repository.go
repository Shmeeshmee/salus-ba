package practitionerrole

import "context"

// Repository stores practitioner roles. Every read sees only the clinic set by SetOrganization (row-level
// security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// ByID returns the role, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (PractitionerRole, error)
	// ByOrganization returns the roles of the clinic org, oldest first; none for an id that is not a UUID.
	ByOrganization(ctx context.Context, org string) ([]PractitionerRole, error)
}
