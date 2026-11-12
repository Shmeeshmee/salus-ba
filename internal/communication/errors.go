package communication

import (
	"errors"
	"fmt"
)

// The errors of the chat. Their text carries no id, value or personal data.
var (
	// ErrInvalid is returned for a request the contract does not allow; the error names the field, never its value.
	ErrInvalid = errors.New("communication: invalid")
	// ErrInvalidCursor is returned for a cursor that is not one the API gave.
	ErrInvalidCursor = errors.New("communication: invalid cursor")
	// ErrNotFound is returned for a conversation or message that does not exist or that the caller may not see:
	// its existence is not disclosed.
	ErrNotFound = errors.New("communication: not found")
	// ErrForbidden is returned to a caller who may not do what they asked: an admin, a practitioner who does not
	// take part, a sender editing someone else's message.
	ErrForbidden = errors.New("communication: forbidden")
	// ErrConsentRequired is returned when the patient's care consent with the clinic is not active: nothing is
	// opened or written, and staff read nothing, after a revocation.
	ErrConsentRequired = errors.New("communication: consent required")
	// ErrVersionConflict is returned for an edit whose If-Match is not the message's current version.
	ErrVersionConflict = errors.New("communication: version conflict")
	// ErrAlreadyParticipant is returned for a practitioner added or invited who takes part, or is invited, already.
	ErrAlreadyParticipant = errors.New("communication: already a participant")
)

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }
