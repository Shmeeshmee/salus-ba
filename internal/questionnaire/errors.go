package questionnaire

import "errors"

// The errors of the questionnaire domain.
var (
	// ErrNotFound is returned for a questionnaire or role that does not exist or that the caller may not see.
	ErrNotFound = errors.New("questionnaire: not found")
	// ErrForbidden is returned to a caller who is not staff of a clinic.
	ErrForbidden = errors.New("questionnaire: forbidden")
	// ErrConflict is returned for an edit or publish of a version that is not the current one (412).
	ErrConflict = errors.New("questionnaire: version conflict")
	// ErrNotDraft is returned for an edit, publish or delete of a version that is published (not_draft).
	ErrNotDraft = errors.New("questionnaire: not a draft")
	// ErrNotActive is returned for a retire or revise of a version that is not active, and for a default
	// whose questionnaire has no active version (not_active).
	ErrNotActive = errors.New("questionnaire: not active")
	// ErrDraftExists is returned for a revise of a series that has a draft (draft_exists).
	ErrDraftExists = errors.New("questionnaire: draft exists")
	// ErrInvalid is what every *InvalidError is (invalid_questionnaire).
	ErrInvalid = errors.New("questionnaire: invalid questionnaire")
)

// InvalidError is a questionnaire that breaks a rule of SPEC.md §2 "Item model", "Limits" or "linkIds".
// LinkIDs are the well-formed linkIds of the offending items, in document order; none when the title, the
// description or the number of items is at fault, or a linkId is malformed. It never holds a title, a text
// or an option.
type InvalidError struct {
	LinkIDs []string
}

func (e *InvalidError) Error() string { return ErrInvalid.Error() }

// Is makes errors.Is(err, ErrInvalid) hold.
func (e *InvalidError) Is(target error) bool { return target == ErrInvalid }
