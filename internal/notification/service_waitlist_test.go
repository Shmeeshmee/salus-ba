package notification_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

const offerID = "0f000000-0000-4000-8000-0000000000f0"

// offerStart is Monday 7 January 2030, 10:00 in Sarajevo; the offer was made at 08:55 there and is held until 09:10.
var (
	offerStart = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	offerMade  = offerStart.Add(-65 * time.Minute)
)

// fakeOffers gives the offer, open and held until expires unless state says otherwise.
type fakeOffers struct {
	expires time.Time
	state   appointment.OfferState
	asked   []string
}

func (f *fakeOffers) OfferNotice(_ context.Context, agent, org, id string) (appointment.OfferNotice, error) {
	f.asked = append(f.asked, agent)
	if id != offerID {
		return appointment.OfferNotice{}, appointment.ErrNotFound
	}
	return appointment.OfferNotice{AccountID: patientAccount, Offer: appointment.Offer{
		ID: id, OrganizationID: org, PatientID: patientID, PractitionerRoleID: roleID, SlotID: "51000000-0000-4000-8000-000000000001",
		Start: offerStart, End: offerStart.Add(30 * time.Minute), Created: offerMade, ExpiresAt: f.expires, State: f.state,
	}}, nil
}

// offerFixture is the reminders' fixture with the offers, at now, for a patient writing in language who opted into
// every channel at a clinic that offers SMS.
func offerFixture(now time.Time, language string) (*reminderFixture, *fakeOffers) {
	f := newReminderFixture(now, offerStart, fakeRules{sms: true})
	offers := &fakeOffers{expires: offerMade.Add(15 * time.Minute), state: appointment.OfferOpen}
	f.deliveries.prefs[patientAccount] = notification.Preferences{Channels: notification.Channels{Push: true, Email: true, SMS: true}}
	accounts := &fakeChatAccounts{language: map[string]string{patientAccount: language}}
	f.svc = notification.NewService(notification.Ports{
		Accounts: accounts, Clinics: sarajevo{}, Questionnaires: f.q, Sender: f.sender, PatientAppURL: "https://app.salus.invalid",
		Deliveries: f.deliveries, DeliveryJobs: f.jobs, Devices: f.repo, Reminders: f.reminders, Offers: offers, Rules: fakeRules{sms: true, repo: f.repo},
		Languages: accounts, Phones: fakePhones{phone: remindPhone}, Profiles: fakeProfiles{phone: remindPhone},
		Pusher: f.pusher, SMS: f.sms, Audit: f.audit, Tx: fakeChatTx{}, Clock: clock.Fixed{T: now},
	})
	return f, offers
}

func offered() appointment.Offer {
	return appointment.Offer{ID: offerID, OrganizationID: remindClin, Created: offerMade, ExpiresAt: offerMade.Add(15 * time.Minute)}
}

// A new offer is delivered through SAL-33's deliveries: an e-mail, a push per device and the SMS the patient opted into,
// once, in the offer's transaction.
func TestNotifyOffer_StoresTheOffersDeliveries(t *testing.T) {
	f, _ := offerFixture(offerMade, "en")
	for range 2 {
		if err := f.svc.NotifyOffer(t.Context(), offered(), patientAccount); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := deliveries(f), []string{"email  pending", "push " + device + " pending", "sms  pending"}; !slices.Equal(got, want) {
		t.Errorf("deliveries %v, want %v", got, want)
	}
	for _, d := range f.deliveries.deliveries {
		if d.Kind != notification.KindWaitlistOffer || d.SourceID != offerID {
			t.Errorf("delivery %+v, want of the offer", d)
		}
	}
}

// The e-mail gives the clinic, the time offered and the hold's end in the clinic's zone and the link to the app, in the
// account's language; nothing of a practitioner, a specialty or an id.
func TestSendDelivery_TheOfferEmailInEveryLanguage(t *testing.T) {
	for _, language := range []string{"en", "bs", "hr", "sr-Latn", "sr-Cyrl", "de", "ar", "tr", "it", "fr"} {
		f, _ := offerFixture(offerMade, language)
		if err := f.svc.NotifyOffer(t.Context(), offered(), patientAccount); err != nil {
			t.Fatal(err)
		}
		if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelEmail).ID, false); err != nil {
			t.Fatalf("%s: %v", language, err)
		}
		if len(f.sender.sent) != 1 {
			t.Fatalf("%s: sent %d e-mails, want 1", language, len(f.sender.sent))
		}
		m := f.sender.sent[0]
		golden(t, language+"/waitlist_offer.txt", mail(m))
		for _, want := range []string{"Synthetic clinic", "10:00", "09:10 (Europe/Sarajevo)", "https://app.salus.invalid/waitlist"} {
			if !strings.Contains(m.Body, want) {
				t.Errorf("%s: the body lacks %q", language, want)
			}
		}
		for _, never := range []string{offerID, patientID, roleID, "51000000", "UTC"} {
			if strings.Contains(m.Body+m.Subject, never) {
				t.Errorf("%s: the e-mail holds %q", language, never)
			}
		}
	}
}

// A hold that ends after midnight in the clinic's zone gives its date too.
func TestSendDelivery_AnOfferHeldPastMidnightGivesItsDate(t *testing.T) {
	f, offers := offerFixture(offerMade, "en")
	offers.expires = time.Date(2030, 1, 7, 23, 10, 0, 0, time.UTC) // 00:10 on Tuesday 8 January in Sarajevo
	if err := f.svc.NotifyOffer(t.Context(), offered(), patientAccount); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelEmail).ID, false); err != nil {
		t.Fatal(err)
	}
	if body := f.sender.sent[0].Body; !strings.Contains(body, "until Tuesday 8 January 2030, 00:10 (Europe/Sarajevo)") {
		t.Errorf("the body lacks the hold's date:\n%s", body)
	}
}

// The push is the generic offer alert; the SMS names the clinic, the time and the hold's end in its zone and links the
// app, in one GSM-7 part for a short clinic name.
func TestSendDelivery_TheOfferPushIsGenericAndItsSMSFitsOnePart(t *testing.T) {
	f, _ := offerFixture(offerMade, "bs")
	if err := f.svc.NotifyOffer(t.Context(), offered(), patientAccount); err != nil {
		t.Fatal(err)
	}
	for _, ch := range []notification.Channel{notification.ChannelPush, notification.ChannelSMS} {
		if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, ch).ID, false); err != nil {
			t.Fatalf("%s: %v", ch, err)
		}
	}
	if sent := f.pusher.Sent(); len(sent) != 1 || sent[0].Title != "Salus" || sent[0].Body != "Termin koji ste čekali je slobodan" {
		t.Errorf("pushed %+v, want the generic offer alert", sent)
	}
	want := "Slobodan termin: Synthetic clinic, 07.01.2030. u 10:00 (Europe/Sarajevo), do 09:10. Zakazite u Salus: https://app.salus.invalid/waitlist"
	sent := f.sms.Sent()
	if len(sent) != 1 || sent[0].To != remindPhone || sent[0].Body != want {
		t.Fatalf("sent %+v, want %q", sent, want)
	}
	if n := sms.Segments(sent[0].Body); n != 1 {
		t.Errorf("the SMS takes %d parts, want 1", n)
	}
}

// An offer that closed, or whose hold ended, before its send sends nothing.
func TestSendDelivery_AnOfferNoLongerOpenSendsNothing(t *testing.T) {
	for name, change := range map[string]func(o *fakeOffers){
		"accepted":       func(o *fakeOffers) { o.state = appointment.OfferAccepted },
		"past its hold":  func(o *fakeOffers) { o.expires = offerMade },
		"unknown source": nil,
	} {
		f, offers := offerFixture(offerMade.Add(time.Minute), "en")
		o := offered()
		if change == nil {
			o.ID = "0f000000-0000-4000-8000-0000000000ff"
		} else {
			change(offers)
		}
		if err := f.svc.NotifyOffer(t.Context(), o, patientAccount); err != nil {
			t.Fatal(err)
		}
		_ = f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelEmail).ID, false)
		if len(f.sender.sent) != 0 {
			t.Errorf("%s: sent %v, want nothing", name, f.sender.sent)
		}
	}
}

// Quiet hours, for the offer to pass an entry over: in the patient's zone, else the clinic's; none set is never quiet.
func TestQuiet_IsInsideThePatientsQuietHours(t *testing.T) {
	f, _ := offerFixture(offerMade, "en")
	q := &notification.QuietHours{Start: "22:00", End: "08:00"}
	for _, tt := range []struct {
		at    time.Time
		quiet *notification.QuietHours
		zone  string
		want  bool
	}{
		{at: time.Date(2030, 1, 7, 22, 30, 0, 0, time.UTC), quiet: q, want: true},                    // 23:30 in Sarajevo
		{at: time.Date(2030, 1, 7, 12, 0, 0, 0, time.UTC), quiet: q},                                 // 13:00
		{at: time.Date(2030, 1, 7, 14, 0, 0, 0, time.UTC), quiet: q, zone: "Asia/Tokyo", want: true}, // 23:00 in Tokyo, 15:00 in Sarajevo
		{at: time.Date(2030, 1, 7, 22, 30, 0, 0, time.UTC)},
	} {
		f.deliveries.prefs[patientAccount] = notification.Preferences{Quiet: tt.quiet, TimeZone: tt.zone}
		got, err := f.svc.Quiet(t.Context(), remindClin, patientAccount, tt.at)
		if err != nil || got != tt.want {
			t.Errorf("Quiet(%v, %s) = %v, %v; want %v", tt.at, tt.zone, got, err, tt.want)
		}
	}
}
