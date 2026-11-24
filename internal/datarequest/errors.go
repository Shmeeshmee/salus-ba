package datarequest

import "errors"

// The errors of a request. Their text carries no personal data.
var (
	// ErrInvalid is returned for an unknown kind, target, answer code or extension reason, or an answer whose code does
	// not fit its status.
	ErrInvalid = errors.New("datarequest: invalid request")
	// ErrNotFound is returned for a request the caller may not see, or a clinic or patient that is not the caller's.
	ErrNotFound = errors.New("datarequest: not found")
	// ErrForbidden is returned when staff who are not the clinic's admin answer, extend or list requests.
	ErrForbidden = errors.New("datarequest: forbidden")
	// ErrClosed is returned for a change to a request that was fulfilled, refused or withdrawn.
	ErrClosed = errors.New("datarequest: closed")
	// ErrAlreadyExtended is returned for a second extension.
	ErrAlreadyExtended = errors.New("datarequest: already extended")
	// ErrVersionConflict is returned when the request changed since the caller read it.
	ErrVersionConflict = errors.New("datarequest: version conflict")
	// ErrWrongPassword is returned when the password that confirms an account's closure is wrong.
	ErrWrongPassword = errors.New("datarequest: wrong password")
	// ErrExportInProgress is returned when an access or portability request is filed while the patient's export of an
	// earlier one is still being built (D2: one export in progress per patient).
	ErrExportInProgress = errors.New("datarequest: export in progress")
)
