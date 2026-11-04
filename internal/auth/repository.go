package auth

import "context"

// Repository stores accounts.
type Repository interface {
	// Create stores a and returns its new id, or ErrEmailTaken when the e-mail has an account. It leaves the
	// transaction usable either way, so the failure can be audited in it.
	Create(ctx context.Context, a Account) (string, error)
	// ByEmail returns the account with the normalised e-mail, or ErrNotFound.
	ByEmail(ctx context.Context, email string) (Account, error)
	// ByID returns the account, or ErrNotFound (also for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Account, error)
	// SetPasswordHash replaces the account's password hash.
	SetPasswordHash(ctx context.Context, id, hash string) error
}

// Hasher hashes and verifies passwords (Argon2Hasher). Both wait for a free slot and give up when ctx ends.
type Hasher interface {
	Hash(ctx context.Context, password string) (string, error)
	// Verify reports whether password matches encoded and whether encoded uses older parameters.
	Verify(ctx context.Context, password, encoded string) (ok, stale bool, err error)
}
