package condition

import "errors"

// ErrInvalid is returned for a Condition the model does not take: an id that is not a UUID, a code that is not a
// well-formed ICD-10 code, an unknown verification status, or no field key or recording time.
var ErrInvalid = errors.New("condition: invalid condition")

// The refusals of a read: a Condition that does not exist or that the caller's scope does not see (its existence
// is not disclosed), a caller who sees it but may not read the signed document it is part of, and the encounter's
// practitioner after the patient revoked their consent.
var (
	ErrNotFound        = errors.New("condition: not found")
	ErrForbidden       = errors.New("condition: forbidden")
	ErrConsentRequired = errors.New("condition: consent required")
)

// ErrShortKey is returned for a pseudonym key shorter than 32 bytes (nil asks for no pseudonyms at all).
var ErrShortKey = errors.New("condition: the pseudonym key is shorter than 32 bytes")
