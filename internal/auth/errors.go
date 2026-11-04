package auth

import (
	"errors"
	"time"
)

// The errors of the auth domain. Their text carries no e-mail, password or token.
var (
	// ErrInvalidEmail is returned for an e-mail that is not a plain address of at most 254 characters.
	ErrInvalidEmail = errors.New("auth: invalid e-mail")
	// ErrInvalidPassword is returned for a password shorter than 12 or longer than 128 characters.
	ErrInvalidPassword = errors.New("auth: invalid password")
	// ErrEmailTaken is returned when an account with the e-mail exists.
	ErrEmailTaken = errors.New("auth: e-mail taken")
	// ErrNotFound is returned for an account that does not exist.
	ErrNotFound = errors.New("auth: account not found")
	// ErrWrongCredentials is returned for an unknown e-mail and for a wrong password alike.
	ErrWrongCredentials = errors.New("auth: wrong e-mail or password")
	// ErrMalformedHash is returned for a stored password hash that is not an argon2id PHC string.
	ErrMalformedHash = errors.New("auth: malformed password hash")
	// ErrUnauthenticated is returned for a session token that is unknown, expired or revoked.
	ErrUnauthenticated = errors.New("auth: no valid session")
	// ErrInvalidChallenge is returned for an MFA challenge that is unknown, expired or ended by wrong codes.
	ErrInvalidChallenge = errors.New("auth: invalid MFA challenge")
	// ErrWrongCode is returned for a TOTP code that does not match or was already used.
	ErrWrongCode = errors.New("auth: wrong TOTP code")
	// ErrAlreadyEnrolled is returned when enrolling an account that has MFA set up.
	ErrAlreadyEnrolled = errors.New("auth: MFA already set up")
	// ErrNotEnrolled is returned when confirming or verifying a code for an account with no secret to check
	// it against: confirm before enrol, or verify before enrolment is confirmed.
	ErrNotEnrolled = errors.New("auth: MFA not set up")
	// ErrSealedSecret is returned for a sealed TOTP secret that does not open.
	ErrSealedSecret = errors.New("auth: sealed secret does not open")
)

// RateLimitedError is returned for a login over its limit; the password was not checked.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string { return "auth: too many login attempts" }
