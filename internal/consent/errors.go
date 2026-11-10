package consent

import "errors"

// The errors of the consent domain. Their text carries no id or personal data.
var (
	// ErrNotFound is returned for a consent that does not exist or that the caller may not see: its
	// existence is not disclosed.
	ErrNotFound = errors.New("consent: not found")
	// ErrClinicNotFound is returned when a consent is granted to a clinic that does not exist.
	ErrClinicNotFound = errors.New("consent: clinic not found")
	// ErrForbidden is returned when staff grant, revoke or list consents: only the patient does.
	ErrForbidden = errors.New("consent: forbidden")
	// ErrProfileRequired is returned when a patient without a profile grants a consent: there is nothing to
	// share yet.
	ErrProfileRequired = errors.New("consent: profile required")
	// ErrActive is returned when a consent is granted to a clinic that holds an active one already.
	ErrActive = errors.New("consent: already active")
	// ErrKind is returned when a consent of an unknown kind is granted.
	ErrKind = errors.New("consent: unknown kind")
	// ErrInactive is returned when a consent that is no longer active is revoked.
	ErrInactive = errors.New("consent: not active")
)
