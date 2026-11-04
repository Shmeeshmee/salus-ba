package auth

import "errors"

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
)
