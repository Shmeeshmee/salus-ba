package notification_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/notification"
)

const deadlineRequest = "00000000-0000-4000-8000-0000000000f1"

// fakeAdmins are clinic A's admins: one in Croatian, one without a language.
type fakeAdmins struct {
	err    error
	agents []string
}

func (a *fakeAdmins) ClinicAdmins(_ context.Context, agent, org string) ([]auth.Contact, error) {
	a.agents = append(a.agents, agent)
	if a.err != nil {
		return nil, a.err
	}
	if org != clinicID {
		return nil, nil
	}
	return []auth.Contact{{Email: "admin-a@example.test", Language: "hr"}, {Email: "admin-b@example.test"}}, nil
}

func deadlineWarning(t *testing.T, org string, days int) notification.DataRequestDeadline {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	return notification.DataRequestDeadline{
		Deadline: time.Date(2026, 12, 24, 22, 59, 59, 0, time.UTC), Zone: loc,
		RequestID: deadlineRequest, OrganizationID: org, Kind: "erasure", Days: days,
	}
}

// A clinic target's warning goes to each of the clinic's admins in their language, with the link to the request; it
// names the kind, the due date and the days left, never the patient.
func TestSendDataRequestDeadline_ToEachOfTheClinicsAdminsInTheirLanguage(t *testing.T) {
	sender, admins := &fakeSender{}, &fakeAdmins{}
	svc := notification.NewService(notification.Ports{Sender: sender, Admins: admins, WebBaseURL: "https://app.example.test/"})
	if _, err := svc.SendDataRequestDeadline(t.Context(), deadlineWarning(t, clinicID, 7)); err != nil {
		t.Fatal(err)
	}
	if len(sender.sent) != 2 || sender.sent[0].To != "admin-a@example.test" || sender.sent[1].To != "admin-b@example.test" {
		t.Fatalf("sent %+v, want one to each admin", sender.sent)
	}
	if hr := sender.sent[0]; !strings.Contains(hr.Body, "Brisanje") || !strings.Contains(hr.Body, "7 dana") || !strings.Contains(hr.Body, "24. prosinca 2026.") {
		t.Errorf("the Croatian admin's e-mail:\n%s", mail(hr))
	}
	if bs := sender.sent[1]; !strings.Contains(bs.Body, "https://app.example.test/data-requests/"+deadlineRequest) || !strings.Contains(bs.Body, "Brisanje") {
		t.Errorf("the fallback language's e-mail lacks the link or the kind:\n%s", mail(bs))
	}
	if len(admins.agents) != 1 || admins.agents[0] != "data_request_deadline_warning" {
		t.Errorf("admins read as %v, want the warning's job", admins.agents)
	}
}

// An account target's warning goes to the operator in their language, pointing at salusctl; without an operator
// address it is ErrNoOperator (the dev stack's: the worker logs it).
func TestSendDataRequestDeadline_ToTheOperator(t *testing.T) {
	sender := &fakeSender{}
	svc := notification.NewService(notification.Ports{Sender: sender, Admins: &fakeAdmins{}, OperatorEmail: "operator@example.test", OperatorLanguage: "en"})
	if _, err := svc.SendDataRequestDeadline(t.Context(), deadlineWarning(t, "", 1)); err != nil {
		t.Fatal(err)
	}
	want := notification.Message{
		To:      "operator@example.test",
		Subject: "A data subject request is due in 1 day",
		Body: "A data subject request is due on Thursday 24 December 2026, 23:59 (Europe/Sarajevo), in 1 day.\n\n" +
			"Kind: Erasure\nRequest: " + deadlineRequest + "\n\n" +
			"To handle it: salusctl datarequest show " + deadlineRequest + "\n",
	}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Errorf("sent %+v,\nwant %+v", sender.sent, want)
	}
	none := notification.NewService(notification.Ports{Sender: sender, Admins: &fakeAdmins{}})
	if _, err := none.SendDataRequestDeadline(t.Context(), deadlineWarning(t, "", 1)); !errors.Is(err, notification.ErrNoOperator) {
		t.Errorf("without an operator: error = %v, want ErrNoOperator", err)
	}
}

func TestSendDataRequestDeadline_FailsWithoutLeakingTheAddress(t *testing.T) {
	down := errors.New("down")
	for name, p := range map[string]notification.Ports{
		"send":   {Sender: &fakeSender{err: down}, Admins: &fakeAdmins{}},
		"admins": {Sender: &fakeSender{}, Admins: &fakeAdmins{err: down}},
	} {
		_, err := notification.NewService(p).SendDataRequestDeadline(t.Context(), deadlineWarning(t, clinicID, 7))
		if !errors.Is(err, down) || strings.Contains(err.Error(), "example.test") {
			t.Errorf("%s fails: error = %v, want it wrapped without an address", name, err)
		}
	}
}

// pickySender cannot carry admin A's address.
type pickySender struct{ fakeSender }

func (p *pickySender) Send(ctx context.Context, m notification.Message) error {
	if m.To == "admin-a@example.test" {
		return notification.ErrRecipientNotASCII
	}
	return p.fakeSender.Send(ctx, m)
}

// An address SMTP cannot carry does not keep the other admins from their warning; a clinic without an admin gets none.
func TestSendDataRequestDeadline_SkipsAnAddressSMTPCannotCarry(t *testing.T) {
	sender := &pickySender{}
	svc := notification.NewService(notification.Ports{Sender: sender, Admins: &fakeAdmins{}})
	n, err := svc.SendDataRequestDeadline(t.Context(), deadlineWarning(t, clinicID, 7))
	if n != 1 || !errors.Is(err, notification.ErrRecipientNotASCII) || len(sender.sent) != 1 || sender.sent[0].To != "admin-b@example.test" {
		t.Errorf("sent %d (%v) to %+v, want admin B's and ErrRecipientNotASCII", n, err, sender.sent)
	}
	if n, err := svc.SendDataRequestDeadline(t.Context(), deadlineWarning(t, "c0000000-0000-4000-8000-0000000000ff", 7)); n != 0 || err != nil {
		t.Errorf("a clinic without admins: %d, %v; want none sent, no error", n, err)
	}
}

// The warning in every language, for every kind, against its golden file.
func TestSendDataRequestDeadline_InEveryLanguage(t *testing.T) {
	for _, lang := range languages {
		for _, tc := range []struct {
			kind string
			days int
		}{{"access", 7}, {"portability", 1}, {"rectification", 7}, {"restriction", 7}, {"objection", 7}, {"erasure", 1}} {
			sender := &fakeSender{}
			svc := notification.NewService(notification.Ports{Sender: sender, OperatorEmail: "operator@example.test", OperatorLanguage: lang})
			w := deadlineWarning(t, "", tc.days)
			w.Kind = tc.kind
			if _, err := svc.SendDataRequestDeadline(t.Context(), w); err != nil || len(sender.sent) != 1 {
				t.Fatalf("%s %s: SendDataRequestDeadline() = %v", lang, tc.kind, err)
			}
			if tc.kind == "access" || tc.kind == "erasure" {
				golden(t, lang+"/data_request_deadline_"+tc.kind+".txt", mail(sender.sent[0]))
			}
			m := sender.sent[0]
			if strings.Contains(m.Subject+m.Body, "{{") || strings.Contains(m.Subject+m.Body, "<no value>") || strings.Contains(m.Body, "Kind: \n") {
				t.Errorf("%s %s: the e-mail holds a template's leftover or no kind:\n%s", lang, tc.kind, mail(m))
			}
		}
	}
	// A clinic's admin gets the link instead of salusctl.
	sender := &fakeSender{}
	svc := notification.NewService(notification.Ports{Sender: sender, Admins: &fakeAdmins{}, WebBaseURL: "https://app.example.test"})
	if _, err := svc.SendDataRequestDeadline(t.Context(), deadlineWarning(t, clinicID, 7)); err != nil {
		t.Fatal(err)
	}
	golden(t, "hr/data_request_deadline_clinic.txt", mail(sender.sent[0]))
}
