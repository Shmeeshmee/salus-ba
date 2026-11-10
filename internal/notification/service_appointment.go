package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// Appointments gives an e-mail job its appointment and the patient's account, audited with the job as agent
// (appointment.Service).
type Appointments interface {
	Notice(ctx context.Context, agent, org, id string) (appointment.Notice, error)
}

// Accounts gives the e-mail of an account and the language to write to it in ("" for none), audited with the
// job as agent (auth.Service).
type Accounts interface {
	AccountEmail(ctx context.Context, agent, id string) (email, language string, err error)
}

// Questionnaires tells whether an appointment has a questionnaire for the patient to fill in, audited with the
// job as agent (questionnaireresponse.Service).
type Questionnaires interface {
	Outstanding(ctx context.Context, agent, org, appointmentID string) (bool, error)
}

// The appointment e-mails, each named by its job's kind, which is the agent of its reads, and written by its
// template (templates/<language>/<template>.txt).
var (
	bookedEmail = appointmentEmail{
		agent: appointment.BookedEmailArgs{}.Kind(), applies: appointment.StatusBooked, template: "booked", asksQuestionnaire: true,
	}
	cancelledEmail = appointmentEmail{
		agent: appointment.CancelledEmailArgs{}.Kind(), applies: appointment.StatusCancelled, template: "cancelled",
	}
	reminderEmail = appointmentEmail{
		agent: appointment.ReminderEmailArgs{}.Kind(), applies: appointment.StatusBooked, template: "reminder", asksQuestionnaire: true,
	}
	// signedEmail applies in every status: a document is signed during the consultation or after it.
	signedEmail = appointmentEmail{agent: composition.SignedEmailArgs{}.Kind(), template: "signed"}
)

// appointmentEmail is one of the appointment e-mails: sent only while the appointment is in the status it
// applies to ("" for any). One that asksQuestionnaire mentions an outstanding questionnaire (COPY E1).
type appointmentEmail struct {
	agent             string
	template          string
	applies           appointment.Status
	asksQuestionnaire bool
}

// SendAppointmentBooked sends the confirmation of the appointment id of the clinic org to its patient (the
// appointment_booked_email job), while it is booked.
func (s *Service) SendAppointmentBooked(ctx context.Context, id, org string) error {
	return s.sendAppointment(ctx, bookedEmail, id, org)
}

// SendAppointmentCancelled sends the cancellation of the appointment id of the clinic org to its patient,
// whoever cancelled it (the appointment_cancelled_email job).
func (s *Service) SendAppointmentCancelled(ctx context.Context, id, org string) error {
	return s.sendAppointment(ctx, cancelledEmail, id, org)
}

// SendDocumentSigned tells the patient of the appointment id of the clinic org that a document of it is ready in
// the app (the document_signed_email job), whatever the appointment's status: which document, its type or its
// content is never in the e-mail (docs/consultation/SPEC.md "Patient documents").
func (s *Service) SendDocumentSigned(ctx context.Context, id, org string) error {
	return s.sendAppointment(ctx, signedEmail, id, org)
}

// SendAppointmentReminder sends the reminder of the appointment id of the clinic org to its patient, 24 hours
// before the start (the appointment_reminder_email job), while it is booked.
func (s *Service) SendAppointmentReminder(ctx context.Context, id, org string) error {
	return s.sendAppointment(ctx, reminderEmail, id, org)
}

// sendAppointment sends e about the appointment id of the clinic org to its patient's account e-mail, in the
// account's language, else Bosnian (docs/i18n/SPEC.md §3 "The account's language fields"): the clinic's name, the start in the clinic's time zone with the zone's name, and the link to the app's
// appointments; nothing about the practitioner, the specialty or the visit (docs/booking/SPEC.md "E-mail"),
// and for the confirmation and the reminder E1 when a questionnaire is outstanding, never which one
// (docs/questionnaire/SPEC.md §2 "Notifications").
// An appointment no longer in the status e applies to sends nothing. Errors carry no address or text;
// appointment.ErrNotFound passes through, for the job to cancel.
func (s *Service) sendAppointment(ctx context.Context, e appointmentEmail, id, org string) error {
	n, err := s.appointments.Notice(ctx, e.agent, org, id)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	if e.applies != "" && n.Appointment.Status != e.applies {
		return nil
	}
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return fmt.Errorf("notification: %s: the clinic's time zone: %w", e.agent, err)
	}
	outstanding, err := s.questionnaireOutstanding(ctx, e, org, id)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	to, language, err := s.accounts.AccountEmail(ctx, e.agent, n.AccountID)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	l := s.in(language)
	when, err := l.when(n.Appointment.Start.In(loc), clinic.TimeZone)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	msg, err := l.message(to, e.template, letter{Clinic: clinic.Name, When: when, Link: s.patientAppURL + "/appointments", Questionnaire: outstanding})
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	return nil
}

// questionnaireOutstanding tells whether e asks and the appointment id of the clinic org has a questionnaire
// outstanding.
func (s *Service) questionnaireOutstanding(ctx context.Context, e appointmentEmail, org, id string) (bool, error) {
	if !e.asksQuestionnaire {
		return false, nil
	}
	return s.questionnaires.Outstanding(ctx, e.agent, org, id)
}
