package appointment

import "errors"

// The errors of the appointment domain. Their text carries no id, time or personal data.
var (
	// ErrNotFound is returned for an appointment that does not exist or that the caller may not see: its
	// existence is not disclosed.
	ErrNotFound = errors.New("appointment: not found")
	// ErrForbidden is returned when the caller may see the appointment but not do what they asked (staff
	// booking, an admin marking a status), or searches outside their own.
	ErrForbidden = errors.New("appointment: forbidden")
	// ErrProfileRequired is returned when a patient without a profile books.
	ErrProfileRequired = errors.New("appointment: profile required")
	// ErrConsentRequired is returned when a patient books with a clinic that holds no active consent of
	// theirs.
	ErrConsentRequired = errors.New("appointment: consent required")
	// ErrSlotUnavailable is returned when the slot is not free, not ahead, not of a listed clinic, or its time
	// is booked with the practitioner already.
	ErrSlotUnavailable = errors.New("appointment: slot unavailable")
	// ErrCancellationWindowClosed is returned when a patient cancels less than 24 hours before the start.
	ErrCancellationWindowClosed = errors.New("appointment: cancellation window closed")
	// ErrClosed is returned when an appointment that is no longer booked is changed.
	ErrClosed = errors.New("appointment: no longer booked")
	// ErrNotStarted is returned when an appointment is marked fulfilled or no-show before its start.
	ErrNotStarted = errors.New("appointment: not started")
	// ErrInvalidStatus is returned for a status a practitioner may not set (only fulfilled and noshow).
	ErrInvalidStatus = errors.New("appointment: invalid status")
)
