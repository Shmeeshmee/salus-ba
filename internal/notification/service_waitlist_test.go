package notification_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

const offerID = "0f000000-0000-4000-8000-0000000000f0"

// fakeOffers gives an offer of the state set, made at 07:55 UTC on the day of start and held until expires (08:10 UTC,
// 09:10 in Sarajevo, by default).
type fakeOffers struct {
	expires time.Time
	err     error
	state   appointment.OfferState
	asked   []string
}

func (f *fakeOffers) OfferNotice(_ context.Context, agent, org, id string) (appointment.OfferNotice, error) {
	f.asked = []string{agent, org, id}
	return appointment.OfferNotice{AccountID: patientAccount, Offer: appointment.Offer{
		ID: id, OrganizationID: org, PatientID: patientID, PractitionerRoleID: roleID, SlotID: "51000000-0000-4000-8000-000000000001",
		Start: start, End: start.Add(30 * time.Minute), Created: start.Add(-65 * time.Minute), ExpiresAt: f.expires, State: f.state,
	}}, f.err
}

type waitlistNotices struct {
	offers   *fakeOffers
	accounts *fakeAccounts
	sender   *fakeSender
	pusher   *pushtest.Sender
	repo     *fakeChatRepo
	audit    *fakeChatAudit
	svc      *notification.Waitlist
}

func newWaitlistNotices(language string) waitlistNotices {
	n := waitlistNotices{
		offers: &fakeOffers{state: appointment.OfferOpen, expires: start.Add(-50 * time.Minute)}, accounts: &fakeAccounts{language: language}, sender: &fakeSender{},
		pusher: &pushtest.Sender{}, repo: newFakeChatRepo(), audit: &fakeChatAudit{},
	}
	n.svc = notification.NewWaitlist(notification.WaitlistPorts{
		Offers: n.offers, Accounts: n.accounts, Clinics: zonedClinics{}, Repo: n.repo, Pusher: n.pusher, Sender: n.sender,
		Audit: n.audit, Tx: fakeChatTx{}, PatientAppURL: "https://patient.example.test/",
	})
	return n
}

// The offer's e-mail holds the clinic, the time offered and the hold's end in the clinic's zone and the link to the
// app, nothing else; each device gets the fixed alert in the account's language.
func TestNotifyOffer_MailsAndPushesTheOfferAndNothingElse(t *testing.T) {
	for _, language := range []string{"en", "bs", "hr", "sr-Latn", "sr-Cyrl", "de", "ar", "tr", "it", "fr"} {
		n := newWaitlistNotices(language)
		if err := n.repo.RegisterDevice(t.Context(), notification.Device{AccountID: patientAccount, Token: "synthetic-token-1", Platform: push.IOS}, chatNow); err != nil {
			t.Fatal(err)
		}
		if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); err != nil {
			t.Fatalf("%s: NotifyOffer() = %v", language, err)
		}
		if strings.Join(n.offers.asked, " ") != "waitlist_offer_notify "+clinicID+" "+offerID ||
			strings.Join(n.accounts.asked, " ") != "waitlist_offer_notify "+patientAccount {
			t.Errorf("%s: asked %v and %v, want the job as agent", language, n.offers.asked, n.accounts.asked)
		}
		if len(n.sender.sent) != 1 || n.sender.sent[0].To != "ana@example.test" {
			t.Fatalf("%s: sent %+v, want one e-mail to the patient", language, n.sender.sent)
		}
		m := n.sender.sent[0]
		golden(t, language+"/waitlist_offer.txt", mail(m))
		for _, want := range []string{"Synthetic clinic", "10:00", "(Europe/Sarajevo)", "09:10 (Europe/Sarajevo)", "https://patient.example.test/waitlist"} {
			if !strings.Contains(m.Body, want) {
				t.Errorf("%s: the body lacks %q", language, want)
			}
		}
		for _, never := range []string{offerID, patientID, roleID, "51000000", "UTC"} {
			if strings.Contains(m.Body, never) || strings.Contains(m.Subject, never) {
				t.Errorf("%s: the e-mail holds %q", language, never)
			}
		}
		sent := n.pusher.Sent()
		if len(sent) != 1 || sent[0].Title != "Salus" || sent[0].Body == "" || strings.Contains(sent[0].Body, "Synthetic clinic") {
			t.Errorf("%s: pushed %+v, want the fixed alert", language, sent)
		}
		if e := n.audit.events[len(n.audit.events)-1]; e.AgentName != "waitlist_offer_notify" || e.EntityReference != "Account/"+patientAccount {
			t.Errorf("%s: audit %+v, want the devices' read by the job", language, e)
		}
	}
}

// A closed offer sends nothing; a gone token is forgotten; failures pass through without the address or token.
func TestNotifyOffer_SendsNothingForAClosedOfferAndPassesFailuresOn(t *testing.T) {
	n := newWaitlistNotices("en")
	n.offers.state = appointment.OfferAccepted
	if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); err != nil || len(n.sender.sent) != 0 {
		t.Errorf("NotifyOffer(accepted) = %v, sent %v; want nothing", err, n.sender.sent)
	}
	n = newWaitlistNotices("en")
	n.offers.err = appointment.ErrNotFound
	if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("NotifyOffer(gone) = %v, want ErrNotFound", err)
	}
	n = newWaitlistNotices("en")
	if err := n.repo.RegisterDevice(t.Context(), notification.Device{AccountID: patientAccount, Token: "synthetic-token-1", Platform: push.Android}, chatNow); err != nil {
		t.Fatal(err)
	}
	n.pusher.Gone("synthetic-token-1")
	if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); err != nil || len(n.repo.devices) != 0 {
		t.Errorf("NotifyOffer(a gone token) = %v, devices %v; want it forgotten", err, n.repo.devices)
	}
	n = newWaitlistNotices("en")
	n.sender.err = errors.New("smtp down")
	if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); err == nil || strings.Contains(err.Error(), "ana@example.test") {
		t.Errorf("NotifyOffer(smtp down) = %v, want its error without the address", err)
	}
}

// A hold that ends after midnight in the clinic's zone gives its date too (review of PR #93, finding 7).
func TestNotifyOffer_AHoldPastMidnightGivesItsDate(t *testing.T) {
	n := newWaitlistNotices("en")
	n.offers.expires = time.Date(2030, 1, 7, 23, 10, 0, 0, time.UTC) // 00:10 on Tuesday 8 January in Sarajevo
	if err := n.svc.NotifyOffer(t.Context(), offerID, clinicID); err != nil {
		t.Fatal(err)
	}
	if body := n.sender.sent[0].Body; !strings.Contains(body, "until Tuesday 8 January 2030, 00:10 (Europe/Sarajevo)") {
		t.Errorf("the body lacks the hold's date:\n%s", body)
	}
}
