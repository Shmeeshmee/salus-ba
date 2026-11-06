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
	// ErrAppointmentClosed is returned for a request to an appointment that is no longer booked, and for a
	// change of the answers of one (appointment_closed, as appointment answers it).
	ErrAppointmentClosed = errors.New("questionnaireresponse: appointment no longer booked")
	// ErrConsentRequired is returned when the patient's consent with the clinic is not active
	// (consent_required).
	ErrConsentRequired = errors.New("questionnaireresponse: consent required")
	// ErrSubmitted is returned for a change to a submitted response (submitted).
	ErrSubmitted = errors.New("questionnaireresponse: submitted")
	// ErrWithdrawn is returned for a change to a withdrawn response (withdrawn).
	ErrWithdrawn = errors.New("questionnaireresponse: withdrawn")
	// ErrPastDue is returned for a change to a response once its appointment has started (past_due).
	ErrPastDue = errors.New("questionnaireresponse: past due")
	// ErrConflict is returned for a change made on a version of the response that is not the current one
	// (412, version_conflict).
	ErrConflict = errors.New("questionnaireresponse: version conflict")
	// ErrAnswersInvalid is what every *AnswersError is (answers_invalid).
	ErrAnswersInvalid = errors.New("questionnaireresponse: answers invalid")
)

// AnswersError is a set of answers that breaks the rules of SPEC.md §2 "Limits" and "Patient flow": an answer
// that does not fit its item's type or limits, or, on submit, an enabled required question left unanswered.
// LinkIDs are the offending items' linkIds, in document order; an answer to no item of the version is refused
// without being named. Its text holds no linkId and no answer.
type AnswersError struct {
	LinkIDs []string
}

func (e *AnswersError) Error() string { return ErrAnswersInvalid.Error() }

// Is makes errors.Is(err, ErrAnswersInvalid) hold.
func (e *AnswersError) Is(target error) bool { return target == ErrAnswersInvalid }
