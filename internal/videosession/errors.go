package videosession

import "errors"

// The errors of the videosession domain. Their text carries no id, room, identity, token or name.
var (
	// ErrNotFound is returned when the caller is not a participant of the appointment, or there is no such
	// appointment: its existence is not disclosed.
	ErrNotFound = errors.New("videosession: not found")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active.
	ErrConsentRequired = errors.New("videosession: consent required")
	// ErrJoinWindowNotOpen is returned before the join window opens, 10 minutes before the start.
	ErrJoinWindowNotOpen = errors.New("videosession: join window not open")
	// ErrConsultationEnded is returned once the session ended, the window closed, or the appointment is no
	// longer booked.
	ErrConsultationEnded = errors.New("videosession: consultation ended")
)
