package invoice

import "context"

// Repository stores receipts. Every read sees only the clinic set by SetOrganization, or the patient's own after
// SetPatient (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id's own receipts.
	SetPatient(ctx context.Context, id string) error
	// ByID returns the receipt with its lines, or ErrNotFound (also outside the scope).
	ByID(ctx context.Context, id string) (Invoice, error)
}
