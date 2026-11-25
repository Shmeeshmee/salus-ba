package careplan

import "errors"

// The errors of a plan. Their text carries no personal data.
var (
	// ErrInvalid is returned for a plan without a title or steps, a step with an unknown service or mode or a day
	// offset out of range, or a booking whose slots do not match the steps.
	ErrInvalid = errors.New("careplan: invalid")
	// ErrNotFound is returned for a plan, patient or quote the caller may not see.
	ErrNotFound = errors.New("careplan: not found")
	// ErrForbidden is returned when the caller's role may not do what was asked.
	ErrForbidden = errors.New("careplan: forbidden")
	// ErrNotDraft is returned for a change to the steps of a booked or ended plan, or a second booking.
	ErrNotDraft = errors.New("careplan: not a draft")
	// ErrEnded is returned for a change to a completed or revoked plan.
	ErrEnded = errors.New("careplan: ended")
	// ErrVersionConflict is returned when the plan changed since the caller read it.
	ErrVersionConflict = errors.New("careplan: version conflict")
)

// ErrSlotUnavailable is returned for a booking whose slot is taken, not ahead or not the clinic's: nothing was booked.
var ErrSlotUnavailable = errors.New("careplan: slot unavailable")

// ErrResourceBusy is returned for a booking a step of which needs a device another visit holds then: nothing was
// booked.
var ErrResourceBusy = errors.New("careplan: resource busy")
