package questionnaireresponse

import "errors"

// The errors of the questionnaireresponse domain. Their text carries no id or personal data.
var (
	// ErrNotFound is returned for a response, an appointment or a questionnaire that does not exist or that
	// the caller may not see: its existence is not disclosed.
	ErrNotFound = errors.New("questionnaireresponse: not found")
	// ErrForbidden is returned to a caller who sees a response but may not change it (the patient, the
	// clinic's admin withdrawing one).
	ErrForbidden = errors.New("questionnaireresponse: forbidden")
	// ErrAlreadyRequested is returned when the appointment has a live request of that questionnaire
	// (already_requested).
	ErrAlreadyRequested = errors.New("questionnaireresponse: already requested")
	// ErrNotActive is returned for a questionnaire version that is a draft or retired (not_active).
	ErrNotActive = errors.New("questionnaireresponse: questionnaire not active")
	// ErrAppointmentStarted is returned for a request to an appointment that has started
	// (appointment_started).
	ErrAppointmentStarted = errors.New("questionnaireresponse: appointment started")
	// ErrAppointmentClosed is returned for a request to an appointment that is no longer booked
	// (appointment_closed, as appointment answers it).
	ErrAppointmentClosed = errors.New("questionnaireresponse: appointment no longer booked")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active
	// (consent_required).
	ErrConsentRequired = errors.New("questionnaireresponse: consent required")
	// ErrSubmitted is returned for a change to a submitted response (submitted).
	ErrSubmitted = errors.New("questionnaireresponse: submitted")
	// ErrWithdrawn is returned for a change to a withdrawn response (withdrawn).
	ErrWithdrawn = errors.New("questionnaireresponse: withdrawn")
)
