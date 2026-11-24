package datarequest

import "context"

// Repository stores requests (data_request) and their history (data_request_change). Package D1 implements it in
// repository_postgres.go.
type Repository interface {
	// SetAccount sets the patient's scope (their account) on the transaction in ctx.
	SetAccount(ctx context.Context, accountID string) error
	// SetOrganization sets the clinic's scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// Insert stores a new request with its first history entry and returns it with its id.
	Insert(ctx context.Context, r Request) (Request, error)
	// ByID returns the request id visible in the scope, or ErrNotFound.
	ByID(ctx context.Context, id string) (Request, error)
	// ForAccount returns the requests of the account, newest first.
	ForAccount(ctx context.Context, accountID string) ([]Request, error)
	// ForClinic returns the requests targeting the clinic, the open ones by due date first.
	ForClinic(ctx context.Context, organizationID string) ([]Request, error)
	// Update stores r's status, extension and answer if its version is still r.Version-1, with a history entry by
	// changedBy when the status changed; false when the version moved on.
	Update(ctx context.Context, r Request, changedBy string) (bool, error)
	// History returns the request's status changes, oldest first.
	History(ctx context.Context, id string) ([]Change, error)
}
