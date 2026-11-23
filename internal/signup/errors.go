package signup

import "errors"

// The errors of a signup. Their text carries no e-mail and no clinic name.
var (
	// ErrInvalid is returned for a signup without a plain e-mail, a clinic name or a known language.
	ErrInvalid = errors.New("signup: invalid signup")
	// ErrInvalidToken is returned for a verification token that is unknown, used or expired.
	ErrInvalidToken = errors.New("signup: invalid token")
)
