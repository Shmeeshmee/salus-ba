package signup

import (
	"errors"
	"time"
)

// The errors of a signup. Their text carries no e-mail and no clinic name.
var (
	// ErrInvalid is returned for a signup without a plain e-mail, a clinic name or a staff language.
	ErrInvalid = errors.New("signup: invalid signup")
	// ErrInvalidToken is returned for a verification token that is unknown, replaced, used or expired.
	ErrInvalidToken = errors.New("signup: invalid token")
	// ErrNotFound is returned for a signup or a clinic that does not exist, or a caller with no clinic.
	ErrNotFound = errors.New("signup: not found")
	// ErrForbidden is returned when a member of the clinic who is not its admin saves the wizard's progress.
	ErrForbidden = errors.New("signup: forbidden")
	// ErrInvalidSetup is returned for an unknown step or state of the wizard.
	ErrInvalidSetup = errors.New("signup: invalid setup")
)

// RateLimitedError is returned for a signup or a verification over its limit; nothing was stored or checked.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string { return "signup: too many requests" }
