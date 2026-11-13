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
	// ErrRescheduleWindowClosed is returned when a patient reschedules inside the clinic's notice (SAL-34).
	ErrRescheduleWindowClosed = errors.New("appointment: reschedule window closed")
	// ErrRescheduleLimit is returned when the appointment was rescheduled as often as the clinic allows (SAL-34).
	ErrRescheduleLimit = errors.New("appointment: reschedule limit reached")
	// ErrNotReschedulable is returned for a reschedule the clinic's rules never allow: another role's slot, the
	// consultation's join window open, or self-service rescheduling off (SAL-34).
	ErrNotReschedulable = errors.New("appointment: not reschedulable")
	// ErrPaymentDeclined is returned when the payment of a priced booking is declined: nothing was booked.
	ErrPaymentDeclined = errors.New("appointment: payment declined")
	// ErrLinkInvalid is returned for a one-time link that is unknown, used or expired, alike (SAL-34).
	ErrLinkInvalid = errors.New("appointment: link invalid")
	// ErrReminderGone is returned for a reminder that is no longer to be sent: already dispatched, cancelled or skipped,
	// or its appointment no longer booked at the start it was scheduled for, or started (SAL-33).
	ErrReminderGone = errors.New("appointment: reminder no longer applies")
)

// ErrShortKey is returned for a pseudonym key shorter than 32 bytes (nil asks for no pseudonyms at all).
var ErrShortKey = errors.New("appointment: the pseudonym key is shorter than 32 bytes")
