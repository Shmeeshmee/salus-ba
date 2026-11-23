package signup

import "context"

// Repository stores signups (clinic_signup). Its Postgres implementation is package O1's.
type Repository interface {
	// ByTokenHash returns the signup whose verification token has the SHA-256 hash.
	ByTokenHash(ctx context.Context, hash []byte) (Signup, error)
}
