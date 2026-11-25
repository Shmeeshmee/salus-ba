package notification_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// annex is a second location of the fakes' clinic, and annexAddress its address on one line.
const (
	annex        = "a3000000-0000-4000-8000-0000000000a2"
	annexAddress = "Synthetic annex 2, 88000 Mostar"
)

// fakeSites gives a location's address on one line, as location.Service.Address; "" for one it does not know.
type fakeSites map[string]string

func (f fakeSites) Address(_ context.Context, _, id string) (string, error) { return f[id], nil }

var annexSites = fakeSites{annex: annexAddress}

// An in-person visit at a location of the clinic gives that location's address, not the clinic's registered one
// (docs/clinic-cms/WAVE6.md U2); a visit at no known location keeps the clinic's.
func TestAppointmentEmails_OfAVisitAtALocationGiveItsAddress(t *testing.T) {
	for _, tc := range []struct{ location, want, not string }{
		{annex, annexAddress, clinicAddress},
		{"", clinicAddress, annexAddress},
	} {
		appts, sender := &fakeAppointments{status: appointment.StatusBooked, mode: schedule.ModeInPerson, location: tc.location}, &fakeSender{}
		svc := notification.NewService(notification.Ports{
			Appointments: appts, Accounts: &fakeAccounts{language: "en"}, Clinics: zonedClinics{}, Sender: sender, Questionnaires: &fakeQuestionnaires{},
			WebBaseURL: "https://app.example.test", PatientAppURL: "https://patient.example.test/", Sites: annexSites,
		})
		if err := svc.SendAppointmentBooked(t.Context(), apptID, clinicID); err != nil || len(sender.sent) != 1 {
			t.Fatalf("location %q: %v, sent %d", tc.location, err, len(sender.sent))
		}
		if body := sender.sent[0].Body; !strings.Contains(body, "Address: "+tc.want) || strings.Contains(body, tc.not) {
			t.Errorf("location %q: the e-mail is\n%s\nwant the address %q", tc.location, body, tc.want)
		}
	}
}

// The reminders of an in-person visit at a location give its address, in the e-mail and the SMS.
func TestSendDelivery_TheRemindersOfAVisitAtALocationGiveItsAddress(t *testing.T) {
	f := dispatched(t, afterChange.Add(-24*time.Hour))
	f.reminders.mode, f.reminders.location = schedule.ModeInPerson, annex
	for _, ch := range []notification.Channel{notification.ChannelEmail, notification.ChannelSMS} {
		if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, ch).ID, false); err != nil {
			t.Fatalf("%s: %v", ch, err)
		}
	}
	if len(f.sender.sent) != 1 || !strings.Contains(f.sender.sent[0].Body, "Address: "+annexAddress) {
		t.Errorf("the e-mail: %+v, want the annex's address", f.sender.sent)
	}
	if len(f.sms.Sent()) != 1 || !strings.Contains(f.sms.Sent()[0].Body, "("+annexAddress+")") {
		t.Errorf("the SMS: %+v, want the annex's address", f.sms.Sent())
	}
}
