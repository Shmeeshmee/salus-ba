package datarequest

import (
	"context"
	"time"
)

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
	// ForOperator returns the requests to the account (no clinic), the open ones by deadline first: on the owner's
	// connection (salusctl), which row-level security does not hold.
	ForOperator(ctx context.Context) ([]Request, error)
	// Update stores r's status, extension and answer if its version is still r.Version-1, with a history entry by
	// changedBy when the status changed; false when the version moved on.
	Update(ctx context.Context, r Request, changedBy string) (bool, error)
	// History returns the request's status changes, oldest first.
	History(ctx context.Context, id string) ([]Change, error)
	// SetDeadlines sets the deadline job's scope on the transaction in ctx: the open requests of every target.
	SetDeadlines(ctx context.Context) error
	// OpenDeadlines returns the open requests whose deadline is before before, the earliest first.
	OpenDeadlines(ctx context.Context, before time.Time) ([]Deadline, error)
	// MarkWarned records that a warning of the open request id was queued at at, if its last mark is still seen (zero
	// for none); false when it is no longer open or another run marked it since.
	MarkWarned(ctx context.Context, id string, seen, at time.Time) (bool, error)
}

// Jobs queues the deadline warnings in the transaction in ctx (River's InsertTx).
type Jobs interface {
	EnqueueWarning(ctx context.Context, args WarningArgs) error
}
