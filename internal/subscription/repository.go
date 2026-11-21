package subscription

import "context"

// Repository stores memberships and their invoices. Every read sees only the clinic set by SetOrganization
// (row-level security); plans are read by everyone.
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// Plans returns the active plans, by code.
	Plans(ctx context.Context) ([]Plan, error)
}
