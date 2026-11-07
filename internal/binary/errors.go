package binary

import "errors"

// The errors of the binary domain. Their text carries no id, title or content type. The holder's refusals
// (ErrNotFound to ErrNotAttachmentItem) are returned by the Holder as these values.
var (
	// ErrNotFound is a file or a response that does not exist or that the caller may not see, and a rejected
	// file: its existence is not disclosed (not_found).
	ErrNotFound = errors.New("binary: not found")
	// ErrForbidden is staff on an upload or a deletion: only the patient changes their answers (forbidden).
	ErrForbidden = errors.New("binary: forbidden")
	// ErrSubmitted is a change to a submitted response (submitted).
	ErrSubmitted = errors.New("binary: submitted")
	// ErrWithdrawn is a change to a withdrawn response (withdrawn).
	ErrWithdrawn = errors.New("binary: withdrawn")
	// ErrPastDue is a change once the appointment has started (past_due).
	ErrPastDue = errors.New("binary: past due")
	// ErrAppointmentClosed is a change once the appointment is no longer booked (appointment_closed).
	ErrAppointmentClosed = errors.New("binary: appointment no longer booked")
	// ErrConsentRequired is a change without the patient's active consent with the clinic (consent_required).
	ErrConsentRequired = errors.New("binary: consent required")
	// ErrNotAttachmentItem is an upload to a linkId that is no attachment item of the response's version
	// (not_attachment_item).
	ErrNotAttachmentItem = errors.New("binary: not an attachment item")
	// ErrLengthRequired is an upload without a length (length_required).
	ErrLengthRequired = errors.New("binary: length required")
	// ErrShortBody is an upload whose body ended before its declared length, or went past it
	// (invalid_request).
	ErrShortBody = errors.New("binary: body does not match its length")
	// ErrTooLarge is a file larger than its kind's limit, or before the bytes are read than every accepted
	// kind's (too_large).
	ErrTooLarge = errors.New("binary: too large")
	// ErrResponseFull is a file that would take the response's files past MaxResponseSize (response_full).
	ErrResponseFull = errors.New("binary: response full")
	// ErrTooManyFiles is a file for an item that already holds its maximum (too_many_files).
	ErrTooManyFiles = errors.New("binary: too many files")
	// ErrUnsupportedType is a file whose bytes are no type the item accepts, or an MP4 with another codec
	// (unsupported_type).
	ErrUnsupportedType = errors.New("binary: unsupported type")
	// ErrScanning is a download while the file is quarantined (scanning).
	ErrScanning = errors.New("binary: scanning")
	// ErrRangeNotSatisfiable is a range outside the file (range_not_satisfiable).
	ErrRangeNotSatisfiable = errors.New("binary: range not satisfiable")
	// ErrObjectNotFound is a Store's answer for a key that holds no object.
	ErrObjectNotFound = errors.New("binary: object not found")
)

// refusals are the errors a call answers with after auditing its refusal; anything else is a failure.
var refusals = []error{
	ErrNotFound, ErrForbidden, ErrSubmitted, ErrWithdrawn, ErrPastDue, ErrAppointmentClosed, ErrConsentRequired,
	ErrNotAttachmentItem, ErrLengthRequired, ErrShortBody, ErrTooLarge, ErrResponseFull, ErrTooManyFiles,
	ErrUnsupportedType, ErrScanning,
}

// refusal is whether err is one of the refusals.
func refusal(err error) bool {
	for _, r := range refusals {
		if errors.Is(err, r) {
			return true
		}
	}
	return false
}
