package transcript

import (
	"errors"
	"fmt"
)

// The errors of the transcript domain. Their text carries no id, value or personal data.
var (
	// ErrInvalid is returned for an event the contract does not allow. The error names the field, never its
	// value.
	ErrInvalid = errors.New("transcript: invalid event")
	// ErrNotFound is returned for an encounter that does not exist or that the caller may not see: its
	// existence is not disclosed.
	ErrNotFound = errors.New("transcript: not found")
	// ErrForbidden is returned to a caller who sees the encounter but may not follow or read what was said in
	// it (access.ConsultationRecord): the clinic's admin, another practitioner, the patient.
	ErrForbidden = errors.New("transcript: forbidden")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active: nothing is
	// written, followed or read after a revocation.
	ErrConsentRequired = errors.New("transcript: consent required")
	// ErrInvalidCursor is returned for a cursor that is not a stream entry's id.
	ErrInvalidCursor = errors.New("transcript: invalid cursor")
)

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }
