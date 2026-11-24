package notification_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
)

// profileDispatched is a fixture whose reminder of a patient without an account was dispatched at now, at a clinic
// with the SMS option.
func profileDispatched(t *testing.T, now time.Time) *reminderFixture {
	t.Helper()
	return profileDispatchedOff(t, now, nil)
}

// profileDispatchedOff is profileDispatched with the clinic's SMS option turned off when *off is true.
func profileDispatchedOff(t *testing.T, now time.Time, off *bool) *reminderFixture {
	t.Helper()
	f := newReminderFixture(now, afterChange, fakeRules{sms: true, off: off})
	f.reminders.noAccount = true
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil {
		t.Fatal(err)
	}
	return f
}

// docs/clinic-cms R3: a patient without an account gets one SMS delivery addressed to them (no account), released
// at once outside the default quiet hours in the clinic's zone; nothing is read at the dispatch (the phone is read only
// for the send). Dispatched again, nothing more.
func TestDispatchReminder_AnSMSToAPatientWithoutAnAccount(t *testing.T) {
	f := profileDispatched(t, afterChange.Add(-time.Hour))
	if f.reminders.dispatched != 1 || len(f.deliveries.deliveries) != 1 || len(f.audit.events) != 0 {
		t.Fatalf("dispatched %d with %v, audit %+v; want one SMS and no read", f.reminders.dispatched, deliveries(f), f.audit.events)
	}
	d := f.deliveries.deliveries[0]
	if d.Channel != notification.ChannelSMS || d.PatientID != patientID || d.AccountID != "" || d.Kind != notification.KindReminder || d.SourceID != reminderID {
		t.Errorf("delivery %+v, want the reminder's SMS to the patient", d)
	}
	if want := []string{"sms " + d.ID + " at 2026-10-25T07:00:00Z"}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("jobs %v, want %v", f.jobs.enqueued, want)
	}
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil || len(f.deliveries.deliveries) != 1 {
		t.Errorf("again: %v with %v, want the one SMS", err, deliveries(f))
	}
}

// The default quiet hours (22:00 to 08:00) apply in the clinic's zone: an SMS due at night waits for 08:00 there, or
// is dropped when that is less than 30 minutes before the start.
func TestDispatchReminder_ThePatientsSMSWaitsForTheClinicsMorning(t *testing.T) {
	// 22:00 UTC on 24 October is midnight in Sarajevo (CEST); 08:00 there after the change is 07:00 UTC.
	f := profileDispatched(t, time.Date(2026, 10, 24, 22, 0, 0, 0, time.UTC))
	if len(f.jobs.enqueued) != 1 || f.jobs.enqueued[0][len(f.jobs.enqueued[0])-20:] != "2026-10-25T07:00:00Z" {
		t.Errorf("jobs %v, want the SMS at 07:00 UTC", f.jobs.enqueued)
	}
	// A visit at 08:15 in Sarajevo: 08:00 is 15 minutes before it, too late.
	f = newReminderFixture(time.Date(2026, 10, 24, 22, 0, 0, 0, time.UTC), time.Date(2026, 10, 25, 7, 15, 0, 0, time.UTC), fakeRules{sms: true})
	f.reminders.noAccount = true
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil {
		t.Fatal(err)
	}
	if f.reminders.dispatched != 1 || len(f.deliveries.deliveries) != 0 {
		t.Errorf("dispatched %d with %v, want it dispatched with nothing", f.reminders.dispatched, deliveries(f))
	}
}

// The SMS goes once to the profile's phone (read for the send, audited with the job as agent) in the clinic's
// language, with the clinic and the time but no link (a one-time link acts as the patient's account, which they do
// not have), and counts in the clinic's usage.
func TestSendDelivery_ThePatientsSMSHasNoLinkAndCounts(t *testing.T) {
	f := profileDispatched(t, afterChange.Add(-time.Hour))
	d := f.deliveries.deliveries[0]
	for range 2 {
		if err := f.svc.SendDelivery(t.Context(), d.ID, false); err != nil {
			t.Fatal(err)
		}
	}
	sent := f.sms.Sent()
	want := "Podsjetnik: Synthetic clinic, 25.10.2026. u 09:00 (Europe/Sarajevo)."
	if len(sent) != 1 || sent[0].To != remindPhone || sent[0].Body != want || sent[0].Reference != d.ID {
		t.Fatalf("sent %+v, want %q once to the phone", sent, want)
	}
	if len(f.reminders.links) != 0 {
		t.Errorf("links %+v, want none", f.reminders.links)
	}
	if u := f.deliveries.usage[remindClin+"2026-10"]; u.Messages != 1 || u.Segments != 1 {
		t.Errorf("usage %+v, want one message of one part", u)
	}
	for _, e := range f.audit.events {
		if e.AgentName != "reminder_sms" {
			t.Errorf("audit %+v, want the send's job as agent", e)
		}
	}
}

// The patient's SMS is cancelled at the send when the clinic turned its SMS option off meanwhile, or the reminder is
// gone (the visit cancelled or moved): nothing is sent or counted.
func TestSendDelivery_CancelsThePatientsSMSThatNoLongerApplies(t *testing.T) {
	for name, change := range map[string]func(f *reminderFixture, off *bool){
		"the option turned off": func(_ *reminderFixture, off *bool) { *off = true },
		"the visit cancelled":   func(f *reminderFixture, _ *bool) { f.reminders.noticeErr = appointment.ErrReminderGone },
	} {
		t.Run(name, func(t *testing.T) {
			off := false
			f := profileDispatchedOff(t, afterChange.Add(-time.Hour), &off)
			change(f, &off)
			if err := f.svc.SendDelivery(t.Context(), f.deliveries.deliveries[0].ID, false); err != nil {
				t.Fatal(err)
			}
			if got := f.deliveries.deliveries[0].State; got != notification.DeliveryCancelled || len(f.sms.Sent()) != 0 || len(f.deliveries.usage) != 0 {
				t.Errorf("state %s, sent %v, usage %v; want it cancelled with nothing sent", got, f.sms.Sent(), f.deliveries.usage)
			}
		})
	}
}

// A passing failure at the send (the phone's read) is retried: the SMS is not cancelled for it.
func TestSendDelivery_RetriesThePatientsSMSOnAPassingFailure(t *testing.T) {
	off := false
	f := profileDispatchedOff(t, afterChange.Add(-time.Hour), &off)
	f.phones.err = errors.New("synthetic timeout")
	if err := f.svc.SendDelivery(t.Context(), f.deliveries.deliveries[0].ID, false); err == nil {
		t.Fatal("SendDelivery = nil, want the failure so the job retries")
	}
	if got := f.deliveries.deliveries[0].State; got == notification.DeliveryCancelled || len(f.sms.Sent()) != 0 {
		t.Errorf("state %s, sent %v; want it still pending with nothing sent", got, f.sms.Sent())
	}
}
