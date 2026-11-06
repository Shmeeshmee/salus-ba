package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// Appointments gives an e-mail job its appointment and the patient's account, audited with the job as agent
// (appointment.Service).
type Appointments interface {
	Notice(ctx context.Context, agent, org, id string) (appointment.Notice, error)
}

// Accounts gives the e-mail of an account, audited with the job as agent (auth.Service).
type Accounts interface {
	AccountEmail(ctx context.Context, agent, id string) (string, error)
}

// Questionnaires tells whether an appointment has a questionnaire for the patient to fill in, audited with the
// job as agent (questionnaireresponse.Service).
type Questionnaires interface {
	Outstanding(ctx context.Context, agent, org, appointmentID string) (bool, error)
}

// questionnaireOutstanding is COPY E1 (docs/questionnaire/COPY.md): the confirmation and the reminder add it as
// their second paragraph when a questionnaire is outstanding. Never the questionnaire's title.
const questionnaireOutstanding = "There is a questionnaire to fill in before your appointment. You can find it in the Salus app.\n\n"

// The appointment e-mails, each named by its job's kind, which is the agent of its reads. body takes extra, a
// paragraph added after the first one ("" for none).
var (
	bookedEmail = appointmentEmail{
		agent: appointment.BookedEmailArgs{}.Kind(), applies: appointment.StatusBooked, subject: "Your appointment is booked", asksQuestionnaire: true,
		body: func(clinic, when, link, extra string) string {
			return "Your appointment at " + clinic + " is booked for " + when + ".\n\n" + extra +
				"To see or cancel it, open the Salus app:\n" + link + "\n\n" +
				"You can cancel it up to 24 hours before it starts.\n"
		},
	}
	cancelledEmail = appointmentEmail{
		agent: appointment.CancelledEmailArgs{}.Kind(), applies: appointment.StatusCancelled, subject: "Your appointment is cancelled",
		body: func(clinic, when, link, _ string) string {
			return "Your appointment at " + clinic + " on " + when + " is cancelled.\n\n" +
				"To book another time, open the Salus app:\n" + link + "\n"
		},
	}
	reminderEmail = appointmentEmail{
		agent: appointment.ReminderEmailArgs{}.Kind(), applies: appointment.StatusBooked, subject: "Reminder of your appointment", asksQuestionnaire: true,
		body: func(clinic, when, link, extra string) string {
			return "This is a reminder of your appointment at " + clinic + " on " + when + ".\n\n" + extra +
				"To see it, open the Salus app:\n" + link + "\n"
		},
	}
)

// appointmentEmail is one of the appointment e-mails: sent only while the appointment is in the status it
// applies to. One that asksQuestionnaire mentions an outstanding questionnaire.
type appointmentEmail struct {
	body              func(clinic, when, link, extra string) string
	agent             string
	subject           string
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

// SendAppointmentReminder sends the reminder of the appointment id of the clinic org to its patient, 24 hours
// before the start (the appointment_reminder_email job), while it is booked.
func (s *Service) SendAppointmentReminder(ctx context.Context, id, org string) error {
	return s.sendAppointment(ctx, reminderEmail, id, org)
}

// sendAppointment sends e about the appointment id of the clinic org to its patient's account e-mail: the
// clinic's name, the start in the clinic's time zone with the zone's name, and the link to the app's
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
	if n.Appointment.Status != e.applies {
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
	extra, err := s.questionnaireMention(ctx, e, org, id)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	to, err := s.accounts.AccountEmail(ctx, e.agent, n.AccountID)
	if err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	when := n.Appointment.Start.In(loc).Format("Monday 2 January 2006, 15:04") + " (" + clinic.TimeZone + ")"
	msg := Message{To: to, Subject: e.subject, Body: e.body(clinic.Name, when, s.patientAppURL+"/appointments", extra)}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: %s: %w", e.agent, err)
	}
	return nil
}

// questionnaireMention is E1's paragraph when e asks and the appointment id of the clinic org has a
// questionnaire outstanding, else "".
func (s *Service) questionnaireMention(ctx context.Context, e appointmentEmail, org, id string) (string, error) {
	if !e.asksQuestionnaire {
		return "", nil
	}
	outstanding, err := s.questionnaires.Outstanding(ctx, e.agent, org, id)
	if err != nil || !outstanding {
		return "", err
	}
	return questionnaireOutstanding, nil
}
