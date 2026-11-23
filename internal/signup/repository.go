package signup

import (
	"context"
	"time"
)

// Repository stores signups (clinic_signup) and the setup wizard's progress (clinic_setup_step).
type Repository interface {
	// Insert stores a new signup with the hash of its first verification token and returns its id.
	Insert(ctx context.Context, r Request, tokenHash []byte) (string, error)
	// ByID returns the signup id, or ErrNotFound.
	ByID(ctx context.Context, id string) (Signup, error)
	// ByTokenHash returns the signup whose verification token has the SHA-256 hash, locked for the transaction in ctx,
	// or ErrNotFound.
	ByTokenHash(ctx context.Context, hash []byte) (Signup, error)
	// SetToken replaces the token hash of the unverified signup id; false when it is verified or unknown.
	SetToken(ctx context.Context, id string, tokenHash []byte) (bool, error)
	// MarkVerified records that the unverified signup id was verified at at and created its clinic; false when it was
	// verified already.
	MarkVerified(ctx context.Context, id, organizationID string, at time.Time) (bool, error)
	// Verified returns the verified signups, those waiting for approval first (the owner's connection).
	Verified(ctx context.Context) ([]Signup, error)
	// Approve records the operator's approval of the verified signup of the clinic organizationID; false when there is
	// none waiting (the owner's connection).
	Approve(ctx context.Context, organizationID string, at time.Time) (bool, error)
	// SetOrganization sets the clinic scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// Steps returns the steps of the clinic in scope that are done or skipped.
	Steps(ctx context.Context, organizationID string) (map[Step]StepState, error)
	// PutStep stores the state of a step of the clinic in scope; StepOpen removes it.
	PutStep(ctx context.Context, organizationID string, step Step, state StepState, at time.Time) error
	// DeleteUnverifiedBefore deletes the unverified signups made before before and returns how many (the retention
	// job, as salus_retention).
	DeleteUnverifiedBefore(ctx context.Context, before time.Time) (int64, error)
}

// Jobs enqueues background work in the transaction in ctx (River's InsertTx).
type Jobs interface {
	// EnqueueEmail enqueues the verification e-mail of the signup id.
	EnqueueEmail(ctx context.Context, signupID string) error
}

// Counters are fixed-window counters (Redis), for the rate limits.
type Counters interface {
	// Incr adds one to the counter key, starting a window of window when there is none, and returns the count and
	// the time left in the window.
	Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
}
