package composition

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalid is returned for an event the contract does not allow. The error names the field, never its value.
var ErrInvalid = errors.New("composition: invalid event")

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }

// ErrInvalidTemplate is returned for a template the rules do not allow (InvalidTemplateError).
var ErrInvalidTemplate = errors.New("composition: invalid template")

// InvalidTemplateError is a refused template: the keys of the offending fields, in the template's order, or
// none when the template as a whole is refused (its title, language, status, type, number of fields, or a
// translation's language or title). It names keys, never a text.
type InvalidTemplateError struct {
	Keys []string
}

func (e *InvalidTemplateError) Error() string {
	if len(e.Keys) == 0 {
		return ErrInvalidTemplate.Error()
	}
	return ErrInvalidTemplate.Error() + ": " + strings.Join(e.Keys, ", ")
}

func (e *InvalidTemplateError) Unwrap() error { return ErrInvalidTemplate }

// The errors of the drafting and signing. Their text carries no id, value or personal data.
var (
	// ErrNotFound is returned for an encounter, a draft, a Composition, a DocumentReference or a Provenance
	// that does not exist or that the caller may not see: its existence is not disclosed.
	ErrNotFound = errors.New("composition: not found")
	// ErrForbidden is returned to a caller who sees the encounter but may not draft, sign or read its
	// documents (access.ConsultationRecord): the clinic's admin, another practitioner, a patient; and to a
	// patient for the templates.
	ErrForbidden = errors.New("composition: forbidden")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active: nothing is
	// drafted, signed or read after a revocation.
	ErrConsentRequired = errors.New("composition: consent required")
	// ErrTemplateNotFound is returned for a template the caller's clinic does not have.
	ErrTemplateNotFound = errors.New("composition: template not found")
	// ErrLanguageUnavailable is returned for a drafting language the template has neither as its original nor
	// as a translation.
	ErrLanguageUnavailable = errors.New("composition: language unavailable")
	// ErrVersionMismatch is returned when a write names a version other than the current one (If-Match).
	ErrVersionMismatch = errors.New("composition: version mismatch")
	// ErrNotPreliminary is returned for a sign of a draft that is no longer preliminary.
	ErrNotPreliminary = errors.New("composition: not preliminary")
	// ErrDraftingUnavailable is returned when the drafting model failed or took too long; nothing is stored.
	ErrDraftingUnavailable = errors.New("composition: drafting unavailable")
	// ErrInvalidValues is returned for a sign whose values do not match the draft's fields (FieldsError).
	ErrInvalidValues = errors.New("composition: invalid values")
	// ErrRequiredMissing is returned for a sign that leaves a required field empty (FieldsError).
	ErrRequiredMissing = errors.New("composition: required field missing")
)

// FieldsError is ErrInvalidValues or ErrRequiredMissing with the keys of the fields concerned, in the draft's
// order. It names keys, never a value.
type FieldsError struct {
	Err  error
	Keys []string
}

func (e *FieldsError) Error() string { return e.Err.Error() + ": " + strings.Join(e.Keys, ", ") }

func (e *FieldsError) Unwrap() error { return e.Err }
