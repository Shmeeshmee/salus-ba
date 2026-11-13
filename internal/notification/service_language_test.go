package notification_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
)

// languages are the tags of the enum language: every e-mail is written in each (docs/i18n/SPEC.md §3).
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

// Every e-mail kind, in every language, is its golden file testdata/<language>/<kind>.txt: subject and
// body translated, the clinic, the start in the clinic's zone with the weekday and month in that language,
// and the link, and no id.
func TestEmails_AreWrittenInEveryLanguage(t *testing.T) {
	for _, lang := range languages {
		sender := &fakeSender{}
		invite := notification.NewService(notification.Ports{
			Invites: &fakeInvites{}, Clinics: &fakeClinics{language: lang}, Sender: sender, WebBaseURL: "https://app.example.test",
		})
		if err := invite.SendInvite(t.Context(), inviteID, clinicID); err != nil {
			t.Fatalf("%s: SendInvite() error = %v", lang, err)
		}
		for _, k := range []struct {
			send        func(s *notification.Service) error
			name        string
			status      appointment.Status
			outstanding bool
		}{
			{func(s *notification.Service) error { return s.SendAppointmentBooked(t.Context(), apptID, clinicID) }, "booked", appointment.StatusBooked, false},
			{func(s *notification.Service) error { return s.SendAppointmentBooked(t.Context(), apptID, clinicID) }, "booked_questionnaire", appointment.StatusBooked, true},
			{func(s *notification.Service) error { return s.SendAppointmentCancelled(t.Context(), apptID, clinicID) }, "cancelled", appointment.StatusCancelled, false},
			{func(s *notification.Service) error { return s.SendDocumentSigned(t.Context(), apptID, clinicID) }, "signed", appointment.StatusFulfilled, false},
		} {
			s := newMailAsking(&fakeAppointments{status: k.status}, &fakeAccounts{language: lang}, zonedClinics{}, sender, &fakeQuestionnaires{outstanding: k.outstanding})
			if err := k.send(s); err != nil {
				t.Fatalf("%s %s: error = %v", lang, k.name, err)
			}
		}
		// The reminder (SAL-33) is a delivery: sent as its send job sends it, with and without a questionnaire.
		for _, outstanding := range []bool{false, true} {
			sendReminderEmail(t, lang, sender, outstanding)
		}
		names := []string{"invite", "booked", "booked_questionnaire", "cancelled", "signed", "reminder", "reminder_questionnaire"}
		if len(sender.sent) != len(names) {
			t.Fatalf("%s: sent %d e-mails, want %d", lang, len(sender.sent), len(names))
		}
		for i, m := range sender.sent {
			golden(t, lang+"/"+names[i]+".txt", mail(m))
			for _, never := range []string{apptID, accountID, inviteID, clinicID, "UTC", "09:00", "{{", "<no value>"} {
				if strings.Contains(m.Subject+m.Body, never) {
					t.Errorf("%s %s: the e-mail holds %q", lang, names[i], never)
				}
			}
			if lang != "en" && strings.Contains(m.Subject, "appointment") {
				t.Errorf("%s %s: the subject %q is English", lang, names[i], m.Subject)
			}
		}
	}
}

// An account with no language of its own gets Bosnian (patients), and so does a clinic without a default
// language (never stored: the column defaults to bs) or an unknown tag.
func TestEmails_FallBackToBosnian(t *testing.T) {
	for _, lang := range []string{"", "xx"} {
		sender := &fakeSender{}
		s := newMail(&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{language: lang}, zonedClinics{}, sender)
		if err := s.SendAppointmentBooked(t.Context(), apptID, clinicID); err != nil {
			t.Fatal(err)
		}
		golden(t, "bs/booked.txt", mail(sender.sent[0]))
		invite := notification.NewService(notification.Ports{
			Invites: &fakeInvites{}, Clinics: &fakeClinics{language: lang}, Sender: sender, WebBaseURL: "https://app.example.test",
		})
		if err := invite.SendInvite(t.Context(), inviteID, clinicID); err != nil {
			t.Fatal(err)
		}
		golden(t, "bs/invite.txt", mail(sender.sent[1]))
	}
}

// The invite goes to someone without an account yet: it is written in the clinic's default language.
func TestSendInvite_IsWrittenInTheClinicsDefaultLanguage(t *testing.T) {
	sender := &fakeSender{}
	s := notification.NewService(notification.Ports{
		Invites: &fakeInvites{}, Clinics: &fakeClinics{language: "hr"}, Sender: sender, WebBaseURL: "https://app.example.test",
	})
	if err := s.SendInvite(t.Context(), inviteID, clinicID); err != nil {
		t.Fatal(err)
	}
	golden(t, "hr/invite.txt", mail(sender.sent[0]))
}
