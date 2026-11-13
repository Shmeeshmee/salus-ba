package notification_test

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
)

var update = flag.Bool("update", false, "rewrite the golden e-mails in testdata/")

const (
	apptID    = "e1000000-0000-4000-8000-0000000000a1"
	accountID = "acc00000-0000-4000-8000-0000000000aa"
)

// start is Monday 7 January 2030, 10:00 in Sarajevo.
var start = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)

// fakeAppointments gives the appointment of a notice, in the status set.
type fakeAppointments struct {
	err    error
	status appointment.Status
	asked  []string // agent, clinic, appointment
}

func (f *fakeAppointments) Notice(_ context.Context, agent, org, id string) (appointment.Notice, error) {
	f.asked = []string{agent, org, id}
	if f.err != nil {
		return appointment.Notice{}, f.err
	}
	return appointment.Notice{
		Appointment: appointment.Appointment{
			ID: id, OrganizationID: org, PatientID: "aa000000-0000-4000-8000-0000000000aa", PractitionerRoleID: "a4000000-0000-4000-8000-0000000000a1",
			Start: start, End: start.Add(30 * time.Minute), Status: f.status,
		},
		AccountID: accountID,
	}, nil
}

// fakeAccounts gives the patient's synthetic address and the language set ("" for none).
type fakeAccounts struct {
	err      error
	language string
	asked    []string // agent, account
}

func (f *fakeAccounts) AccountEmail(_ context.Context, agent, id string) (string, string, error) {
	f.asked = []string{agent, id}
	return "ana@example.test", f.language, f.err
}

type zonedClinics struct{ err error }

func (c zonedClinics) Read(_ context.Context, cl organization.Caller, id string) (organization.Organization, error) {
	if c.err != nil || cl.OrganizationID != id {
		return organization.Organization{}, errors.Join(organization.ErrNotFound, c.err)
	}
	return organization.Organization{ID: id, Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo"}, nil
}

// fakeQuestionnaires tells whether a questionnaire is outstanding, as set.
type fakeQuestionnaires struct {
	err         error
	asked       []string // agent, clinic, appointment
	outstanding bool
}

func (f *fakeQuestionnaires) Outstanding(_ context.Context, agent, org, id string) (bool, error) {
	f.asked = []string{agent, org, id}
	return f.outstanding, f.err
}

func newMail(appts *fakeAppointments, accounts *fakeAccounts, clinics zonedClinics, sender *fakeSender) *notification.Service {
	return newMailAsking(appts, accounts, clinics, sender, &fakeQuestionnaires{})
}

func newMailAsking(appts *fakeAppointments, accounts *fakeAccounts, clinics zonedClinics, sender *fakeSender, q *fakeQuestionnaires) *notification.Service {
	return notification.NewService(notification.Ports{
		Appointments: appts, Accounts: accounts, Clinics: clinics, Sender: sender, Questionnaires: q,
		WebBaseURL: "https://app.example.test", PatientAppURL: "https://patient.example.test/",
	})
}

// mail is how a golden file holds an e-mail: its subject line, a blank line and its body.
func mail(m notification.Message) string {
	return "Subject: " + m.Subject + "\n\n" + m.Body
}

// golden compares body with testdata/name (en/ when name has no folder), or rewrites it with -update.
func golden(t *testing.T, name, body string) {
	t.Helper()
	if !strings.Contains(name, "/") {
		name = "en/" + name
	}
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if body != string(want) {
		t.Errorf("body of %s:\n%s\nwant:\n%s", name, body, want)
	}
}

func TestAppointmentEmails_HoldTheClinicTheLocalTimeAndTheLinkAndNothingElse(t *testing.T) {
	tests := []struct {
		send    func(s *notification.Service, ctx context.Context) error
		status  appointment.Status
		agent   string
		subject string
		golden  string
	}{
		{func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentBooked(ctx, apptID, clinicID)
		},
			appointment.StatusBooked, "appointment_booked_email", "Your appointment is booked", "booked.txt"},
		{func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentCancelled(ctx, apptID, clinicID)
		},
			appointment.StatusCancelled, "appointment_cancelled_email", "Your appointment is cancelled", "cancelled.txt"},
		{func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentReminder(ctx, apptID, clinicID)
		},
			appointment.StatusBooked, "appointment_reminder_email", "Reminder of your appointment", "reminder.txt"},
	}
	for _, tt := range tests {
		appts, accounts, sender := &fakeAppointments{status: tt.status}, &fakeAccounts{language: "en"}, &fakeSender{}
		if err := tt.send(newMail(appts, accounts, zonedClinics{}, sender), t.Context()); err != nil {
			t.Fatalf("%s: error = %v", tt.agent, err)
		}
		if want := []string{tt.agent, clinicID, apptID}; strings.Join(appts.asked, " ") != strings.Join(want, " ") ||
			strings.Join(accounts.asked, " ") != tt.agent+" "+accountID {
			t.Errorf("%s: asked for the notice %v and the account %v, want the job as agent", tt.agent, appts.asked, accounts.asked)
		}
		if len(sender.sent) != 1 || sender.sent[0].To != "ana@example.test" || sender.sent[0].Subject != tt.subject {
			t.Fatalf("%s: sent %+v, want one e-mail to the patient with subject %q", tt.agent, sender.sent, tt.subject)
		}
		body := sender.sent[0].Body
		golden(t, tt.golden, mail(sender.sent[0]))
		for _, want := range []string{"Synthetic clinic", "Monday 7 January 2030, 10:00 (Europe/Sarajevo)", "https://patient.example.test/appointments"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: the body lacks %q", tt.agent, want)
			}
		}
		for _, never := range []string{apptID, accountID, "a4000000", "aa000000", "UTC", "09:00"} {
			if strings.Contains(body, never) || strings.Contains(sender.sent[0].Subject, never) {
				t.Errorf("%s: the e-mail holds %q", tt.agent, never)
			}
		}
	}
}

// An e-mail that no longer applies is not sent: a confirmation or reminder of an appointment that is not
// booked, a cancellation of one that is not cancelled.
func TestAppointmentEmails_SendNothingThatNoLongerApplies(t *testing.T) {
	for name, tt := range map[string]struct {
		send   func(s *notification.Service, ctx context.Context) error
		status appointment.Status
	}{
		"a reminder of a cancelled appointment": {func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentReminder(ctx, apptID, clinicID)
		}, appointment.StatusCancelled},
		"a reminder of a fulfilled appointment": {func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentReminder(ctx, apptID, clinicID)
		}, appointment.StatusFulfilled},
		"a confirmation of a cancelled one": {func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentBooked(ctx, apptID, clinicID)
		}, appointment.StatusCancelled},
		"a cancellation of a booked one": {func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentCancelled(ctx, apptID, clinicID)
		}, appointment.StatusBooked},
	} {
		accounts, sender := &fakeAccounts{language: "en"}, &fakeSender{}
		if err := tt.send(newMail(&fakeAppointments{status: tt.status}, accounts, zonedClinics{}, sender), t.Context()); err != nil {
			t.Errorf("%s: error = %v, want nil", name, err)
		}
		if len(sender.sent) != 0 || accounts.asked != nil {
			t.Errorf("%s: sent %v after reading the account %v, want nothing", name, sender.sent, accounts.asked)
		}
	}
}

func TestAppointmentEmails_StopAtTheFirstFailureWithoutLeakingTheMessage(t *testing.T) {
	down := errors.New("down")
	for name, s := range map[string]struct {
		appts    *fakeAppointments
		accounts *fakeAccounts
		clinics  zonedClinics
		sender   *fakeSender
	}{
		"the appointment": {&fakeAppointments{err: down}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{}},
		"the clinic":      {&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{language: "en"}, zonedClinics{err: down}, &fakeSender{}},
		"the account":     {&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{err: down, language: "en"}, zonedClinics{}, &fakeSender{}},
		"the send":        {&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{err: down}},
	} {
		err := newMail(s.appts, s.accounts, s.clinics, s.sender).SendAppointmentBooked(t.Context(), apptID, clinicID)
		if !errors.Is(err, down) || strings.Contains(err.Error(), "ana@example.test") || strings.Contains(err.Error(), "Synthetic clinic") {
			t.Errorf("%s failing: error = %v, want it without the address or the clinic", name, err)
		}
		if name != "the send" && len(s.sender.sent) != 0 {
			t.Errorf("%s failing: an e-mail was sent", name)
		}
	}
	err := newMail(&fakeAppointments{err: appointment.ErrNotFound}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{}).SendAppointmentReminder(t.Context(), apptID, clinicID)
	if !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("an unknown appointment = %v, want appointment.ErrNotFound for the job to cancel", err)
	}
}

// e1 is COPY.md E1, the sentence the confirmation and the reminder add when a questionnaire is outstanding.
const e1 = "There is a questionnaire to fill in before your appointment. You can find it in the Salus app."

// The confirmation and the reminder add E1 after their first paragraph when the job's check finds a
// questionnaire outstanding, and leave it out otherwise; the cancellation never asks.
func TestAppointmentEmails_MentionAnOutstandingQuestionnaireAfterTheFirstParagraph(t *testing.T) {
	tests := []struct {
		send   func(s *notification.Service, ctx context.Context) error
		agent  string
		golden string
	}{
		{func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentBooked(ctx, apptID, clinicID)
		}, "appointment_booked_email", "booked"},
		{func(s *notification.Service, ctx context.Context) error {
			return s.SendAppointmentReminder(ctx, apptID, clinicID)
		}, "appointment_reminder_email", "reminder"},
	}
	for _, tt := range tests {
		for _, outstanding := range []bool{true, false} {
			q, sender := &fakeQuestionnaires{outstanding: outstanding}, &fakeSender{}
			s := newMailAsking(&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{language: "en"}, zonedClinics{}, sender, q)
			if err := tt.send(s, t.Context()); err != nil {
				t.Fatalf("%s: error = %v", tt.agent, err)
			}
			if want := tt.agent + " " + clinicID + " " + apptID; strings.Join(q.asked, " ") != want {
				t.Errorf("%s: asked %v, want %s: the job as agent", tt.agent, q.asked, want)
			}
			if len(sender.sent) != 1 {
				t.Fatalf("%s: sent %d e-mails, want 1", tt.agent, len(sender.sent))
			}
			body := sender.sent[0].Body
			if !outstanding {
				golden(t, tt.golden+".txt", mail(sender.sent[0]))
				continue
			}
			golden(t, tt.golden+"_questionnaire.txt", mail(sender.sent[0]))
			paragraphs := strings.Split(body, "\n\n")
			if len(paragraphs) < 2 || paragraphs[1] != e1 {
				t.Errorf("%s: the second paragraph is not E1:\n%s", tt.agent, body)
			}
		}
	}
	q, sender := &fakeQuestionnaires{outstanding: true}, &fakeSender{}
	if err := newMailAsking(&fakeAppointments{status: appointment.StatusCancelled}, &fakeAccounts{language: "en"}, zonedClinics{}, sender, q).
		SendAppointmentCancelled(t.Context(), apptID, clinicID); err != nil {
		t.Fatal(err)
	}
	if q.asked != nil || len(sender.sent) != 1 || strings.Contains(sender.sent[0].Body, "questionnaire") {
		t.Errorf("the cancellation asked %v and sent %+v, want no check and no mention", q.asked, sender.sent)
	}
}

// A failed check sends nothing (the job retries), and an e-mail that no longer applies does not check.
func TestAppointmentEmails_SendNothingWhenTheQuestionnaireCheckFails(t *testing.T) {
	down := errors.New("down")
	q, sender := &fakeQuestionnaires{err: down}, &fakeSender{}
	err := newMailAsking(&fakeAppointments{status: appointment.StatusBooked}, &fakeAccounts{language: "en"}, zonedClinics{}, sender, q).
		SendAppointmentBooked(t.Context(), apptID, clinicID)
	if !errors.Is(err, down) || len(sender.sent) != 0 {
		t.Errorf("error = %v, sent %d; want the check's error and nothing sent", err, len(sender.sent))
	}
	q = &fakeQuestionnaires{outstanding: true}
	if err := newMailAsking(&fakeAppointments{status: appointment.StatusCancelled}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{}, q).
		SendAppointmentReminder(t.Context(), apptID, clinicID); err != nil || q.asked != nil {
		t.Errorf("a reminder of a cancelled appointment: error = %v, asked %v; want nil and no check", err, q.asked)
	}
}
