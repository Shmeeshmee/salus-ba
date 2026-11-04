package patient

import "errors"

// The errors of the patient domain. Their text carries no name, date or phone number.
var (
	// ErrNotFound is returned when there is no profile, or one the caller may not see: its existence is not
	// disclosed.
	ErrNotFound = errors.New("patient: not found")
	// ErrForbidden is returned when someone other than a patient asks for or saves their own profile.
	ErrForbidden = errors.New("patient: forbidden")
	// ErrConflict is returned when a save was made against a version that is not the current one, or creates
	// a profile that exists already.
	ErrConflict = errors.New("patient: version conflict")
	// ErrInvalidName is returned for a given or family name that is blank or over 100 characters.
	ErrInvalidName = errors.New("patient: invalid name")
	// ErrInvalidBirthDate is returned for a birth date that is not in the past or is before 1900-01-01.
	ErrInvalidBirthDate = errors.New("patient: invalid birth date")
	// ErrInvalidPhone is returned for a phone number that is not E.164 (+ and 8 to 15 digits).
	ErrInvalidPhone = errors.New("patient: invalid phone")
)
