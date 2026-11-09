package suggestion

import (
	"errors"
	"fmt"
)

// The errors of the suggestion domain. Their text carries no id, value or personal data.
var (
	// ErrInvalid is returned for an event the contract does not allow. The error names the field, never its
	// value.
	ErrInvalid = errors.New("suggestion: invalid event")
	// ErrNotFound is returned for an encounter that does not exist or that the clinic's scope does not show.
	ErrNotFound = errors.New("suggestion: not found")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active: nothing is
	// written after a revocation.
	ErrConsentRequired = errors.New("suggestion: consent required")
	// ErrForbidden is returned to a caller who sees the encounter but may not act on its suggestions: anyone but
	// its practitioner.
	ErrForbidden = errors.New("suggestion: forbidden")
)

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }
