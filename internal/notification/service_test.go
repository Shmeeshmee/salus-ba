package notification_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
)

const (
	inviteID = "1a000000-0000-4000-8000-000000000001"
	clinicID = "c0000000-0000-4000-8000-00000000000c"
	token    = "c2Vzc2lvbiB0b2tlbiBvZiAzMiBzeW50aGV0aWMgYnl0ZXM" // synthetic
)

type fakeInvites struct {
	err      error
	inviteID string
	org      string
	issued   int
}

func (f *fakeInvites) IssueInviteToken(_ context.Context, inviteID, org string) (string, string, error) {
	f.inviteID, f.org = inviteID, org
	f.issued++
	return "cy@example.test", token, f.err
}

// fakeClinics gives a synthetic clinic with the default language set ("" for none).
type fakeClinics struct {
	err      error
	language string
	caller   organization.Caller
}

func (f *fakeClinics) Read(_ context.Context, c organization.Caller, id string) (organization.Organization, error) {
	f.caller = c
	return organization.Organization{ID: id, Name: "Synthetic clinic", DefaultLanguage: f.language}, f.err
}

type fakeSender struct {
	err  error
	sent []notification.Message
}

func (f *fakeSender) Send(_ context.Context, m notification.Message) error {
	f.sent = append(f.sent, m)
	return f.err
}

func TestSendInvite_MailsTheLinkAndTheClinicNameAndNothingElse(t *testing.T) {
	invites, clinics, sender := &fakeInvites{}, &fakeClinics{language: "en"}, &fakeSender{}
	svc := notification.NewService(notification.Ports{Invites: invites, Clinics: clinics, Sender: sender, WebBaseURL: "https://app.example.test"})

	if err := svc.SendInvite(t.Context(), inviteID, clinicID); err != nil {
		t.Fatalf("SendInvite() error = %v", err)
	}
	if invites.inviteID != inviteID || invites.org != clinicID || clinics.caller != (organization.Caller{OrganizationID: clinicID}) {
		t.Errorf("token issued for %s in %s, clinic read as %+v", invites.inviteID, invites.org, clinics.caller)
	}
	want := notification.Message{
		To:      "cy@example.test",
		Subject: "Your invitation to Salus",
		Body: "You are invited to join Synthetic clinic on Salus.\n\n" +
			"To accept, open this link, choose a password and set up a sign-in code:\n" +
			"https://app.example.test/invite#" + token + "\n\n" +
			"The link works once and expires 72 hours after the invitation was made. If you did not expect this e-mail, you can ignore it.\n",
	}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Errorf("sent %+v,\nwant %+v", sender.sent, want)
	}
}

func TestSendInvite_StopsAtTheFirstFailureWithoutLeakingTheMessage(t *testing.T) {
	down := errors.New("down")
	for name, svc := range map[string]func() (*notification.Service, *fakeSender){
		"clinic": func() (*notification.Service, *fakeSender) {
			s := &fakeSender{}
			return notification.NewService(notification.Ports{Invites: &fakeInvites{}, Clinics: &fakeClinics{err: down}, Sender: s, WebBaseURL: "https://app.example.test"}), s
		},
		"token": func() (*notification.Service, *fakeSender) {
			s := &fakeSender{}
			return notification.NewService(notification.Ports{Invites: &fakeInvites{err: down}, Clinics: &fakeClinics{}, Sender: s, WebBaseURL: "https://app.example.test"}), s
		},
		"send": func() (*notification.Service, *fakeSender) {
			s := &fakeSender{err: down}
			return notification.NewService(notification.Ports{Invites: &fakeInvites{}, Clinics: &fakeClinics{}, Sender: s, WebBaseURL: "https://app.example.test"}), s
		},
	} {
		s, sender := svc()
		err := s.SendInvite(t.Context(), inviteID, clinicID)
		if !errors.Is(err, down) || strings.Contains(err.Error(), "example.test") || strings.Contains(err.Error(), token) {
			t.Errorf("%s failing: SendInvite() = %v, want its error without the address or the token", name, err)
		}
		if name != "send" && len(sender.sent) != 0 {
			t.Errorf("%s failing: an e-mail was sent", name)
		}
	}
}
