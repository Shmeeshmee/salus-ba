package notification_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// The October 2026 change in Europe/Sarajevo is on Sunday 25 October at 03:00 local (01:00 UTC); Asia/Dubai is UTC+4
// all year. A patient abroad in Dubai, with a clinic in Sarajevo.
var (
	beforeChange = time.Date(2026, 10, 24, 19, 30, 0, 0, time.UTC) // 21:30 in Sarajevo, 23:30 in Dubai
	afterChange  = time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)   // 09:00 in Sarajevo (CET), 12:00 in Dubai
	device       = "c3000000-0000-4000-8000-0000000000c3"
)

func deliveries(f *reminderFixture) []string {
	var out []string
	for _, d := range f.deliveries.deliveries {
		out = append(out, string(d.Channel)+" "+d.DeviceID+" "+string(d.State))
	}
	return out
}

// Deliver stores an e-mail and a push per device for the default settings, and an SMS only when the patient opted in
// and the clinic offers it; each with its send job now, outside quiet hours. A second Deliver of the same source adds
// nothing.
func TestDeliver_ChoosesTheChannelsOfThePatientAndTheClinic(t *testing.T) {
	for _, tt := range []struct {
		prefs     *notification.Preferences
		name      string
		want      []string
		clinicSMS bool
	}{
		{name: "the defaults", want: []string{"email  pending", "push " + device + " pending"}},
		{name: "SMS opted in, the clinic without", prefs: &notification.Preferences{Channels: notification.Channels{Email: true, SMS: true}}, want: []string{"email  pending"}},
		{name: "SMS opted in at a clinic with SMS", prefs: &notification.Preferences{Channels: notification.Channels{SMS: true}}, clinicSMS: true, want: []string{"sms  pending"}},
		{name: "nothing on", prefs: &notification.Preferences{}, clinicSMS: true},
	} {
		f := newReminderFixture(afterChange.Add(-24*time.Hour), afterChange, fakeRules{sms: tt.clinicSMS})
		if tt.prefs != nil {
			f.deliveries.prefs[patientAccount] = *tt.prefs
		}
		for range 2 {
			if err := f.svc.Deliver(t.Context(), notification.KindReminder, reminderID, remindClin, patientAccount, notification.Content{Deadline: afterChange}); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
		}
		if got := deliveries(f); !slices.Equal(got, tt.want) {
			t.Errorf("%s: deliveries %v, want %v", tt.name, got, tt.want)
		}
		if len(f.jobs.enqueued) != len(tt.want) {
			t.Errorf("%s: jobs %v, want one per delivery", tt.name, f.jobs.enqueued)
		}
		for _, j := range f.jobs.enqueued {
			if !strings.HasSuffix(j, afterChange.Add(-24*time.Hour).Format(time.RFC3339)) {
				t.Errorf("%s: job %s, want it now", tt.name, j)
			}
		}
	}
}

// Quiet hours are the patient's, in their zone, else the clinic's: inside them a push or SMS waits until they end if
// that is still 30 minutes before the start, and is dropped otherwise; the e-mail goes at once. Across the October
// change in Sarajevo, the instants stay absolute.
func TestDeliver_HoldsPushAndSMSUntilTheQuietHoursOfThePatientsZoneEnd(t *testing.T) {
	quiet := &notification.QuietHours{Start: "22:00", End: "08:00"}
	for _, tt := range []struct {
		name     string
		zone     string
		deadline time.Time
		push     string // when the push and SMS go, "" for dropped
	}{
		// 23:30 in Dubai: quiet until 08:00 there (04:00 UTC), three and a half hours before the start.
		{name: "in Dubai, the start ahead", zone: "Asia/Dubai", deadline: afterChange, push: "2026-10-25T04:00:00Z"},
		// The quiet hours end 15 minutes before the start: too late, dropped.
		{name: "in Dubai, the start too soon", zone: "Asia/Dubai", deadline: time.Date(2026, 10, 25, 4, 15, 0, 0, time.UTC)},
		// No zone: the clinic's, 21:30 in Sarajevo, before its quiet hours.
		{name: "in the clinic's zone", deadline: afterChange, push: beforeChange.Format(time.RFC3339)},
	} {
		f := newReminderFixture(beforeChange, tt.deadline, fakeRules{sms: true})
		f.deliveries.prefs[patientAccount] = notification.Preferences{Channels: notification.Channels{Push: true, Email: true, SMS: true}, Quiet: quiet, TimeZone: tt.zone}
		if err := f.svc.Deliver(t.Context(), notification.KindReminder, reminderID, remindClin, patientAccount, notification.Content{Deadline: tt.deadline}); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		want := []string{"email d0000001-0000-4000-8000-000000000000 at " + beforeChange.Format(time.RFC3339)}
		if tt.push != "" {
			want = append(want, "push d0000002-0000-4000-8000-000000000000 at "+tt.push, "sms d0000003-0000-4000-8000-000000000000 at "+tt.push)
		}
		if !slices.Equal(f.jobs.enqueued, want) {
			t.Errorf("%s: jobs %v,\nwant %v", tt.name, f.jobs.enqueued, want)
		}
	}
}

// Quiet hours that do not wrap midnight, and the morning after the October change: the end is the next 08:00 local.
func TestDeliver_QuietHoursEndOnTheLocalClock(t *testing.T) {
	// 06:30 UTC on 25 October is 07:30 in Sarajevo after the change: quiet until 08:00 CET, 07:00 UTC.
	now := time.Date(2026, 10, 25, 6, 30, 0, 0, time.UTC)
	f := newReminderFixture(now, afterChange.Add(4*time.Hour), fakeRules{})
	f.deliveries.prefs[patientAccount] = notification.Preferences{Channels: notification.Channels{Push: true}, Quiet: &notification.QuietHours{Start: "00:30", End: "08:00"}}
	if err := f.svc.Deliver(t.Context(), notification.KindReminder, reminderID, remindClin, patientAccount, notification.Content{Deadline: afterChange.Add(4 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"push d0000001-0000-4000-8000-000000000000 at 2026-10-25T07:00:00Z"}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("jobs %v, want %v", f.jobs.enqueued, want)
	}
}

// The dispatch takes the reminder, delivers it and marks it dispatched in one transaction; a reminder no longer to be
// sent does nothing, and one of another clinic is not found.
func TestDispatchReminder_DeliversAndMarksTheReminderDispatched(t *testing.T) {
	f := newReminderFixture(afterChange.Add(-time.Hour), afterChange, fakeRules{})
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil {
		t.Fatal(err)
	}
	if f.reminders.dispatched != 1 || len(f.deliveries.deliveries) != 2 {
		t.Errorf("dispatched %d with %v, want once with the e-mail and the push", f.reminders.dispatched, deliveries(f))
	}
	f = newReminderFixture(afterChange.Add(-time.Hour), afterChange, fakeRules{})
	f.reminders.takeErr = appointment.ErrReminderGone
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil || f.reminders.dispatched != 0 || len(f.deliveries.deliveries) != 0 {
		t.Errorf("a reminder gone: %v, dispatched %d, %v; want nothing", err, f.reminders.dispatched, deliveries(f))
	}
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: clinicID}); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("another clinic's: %v, want appointment.ErrNotFound", err)
	}
}

// dispatched is a fixture whose reminder has its deliveries, the patient in Dubai with SMS on at a clinic with SMS.
func dispatched(t *testing.T, now time.Time) *reminderFixture {
	t.Helper()
	f := newReminderFixture(now, afterChange, fakeRules{sms: true})
	f.deliveries.prefs[patientAccount] = notification.Preferences{Channels: notification.Channels{Push: true, Email: true, SMS: true}, TimeZone: "Asia/Dubai"}
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil {
		t.Fatal(err)
	}
	return f
}

func deliveryOf(t *testing.T, f *reminderFixture, ch notification.Channel) notification.Delivery {
	t.Helper()
	for _, d := range f.deliveries.deliveries {
		if d.Channel == ch {
			return d
		}
	}
	t.Fatalf("no %s delivery in %v", ch, deliveries(f))
	return notification.Delivery{}
}

// The e-mail holds the clinic's name, the time in the clinic's zone and, as the patient is abroad, in theirs, a
// one-time link issued for it, and the app's link; nothing about the practitioner or the visit. Sent once.
func TestSendDelivery_TheReminderEmailHoldsBothTimesAndAOneTimeLink(t *testing.T) {
	f := dispatched(t, afterChange.Add(-24*time.Hour))
	d := deliveryOf(t, f, notification.ChannelEmail)
	for range 2 {
		if err := f.svc.SendDelivery(t.Context(), d.ID, false); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.sender.sent) != 1 {
		t.Fatalf("sent %d e-mails, want 1", len(f.sender.sent))
	}
	m := f.sender.sent[0]
	golden(t, "reminder_abroad.txt", mail(m))
	for _, want := range []string{"Synthetic clinic", "Sunday 25 October 2026, 09:00 (Europe/Sarajevo)", "Sunday 25 October 2026, 12:00 (Asia/Dubai)",
		fakeLink(1), "https://app.salus.invalid/appointments"} {
		if !strings.Contains(m.Body, want) {
			t.Errorf("the e-mail lacks %q:\n%s", want, m.Body)
		}
	}
	want := appointment.Link{AppointmentID: remindAppt, OrganizationID: remindClin, ReminderID: reminderID, Language: "en", Channel: appointment.ChannelEmail}
	if !slices.Equal(f.reminders.links, []appointment.Link{want}) {
		t.Errorf("links %+v, want %+v", f.reminders.links, want)
	}
	if got := deliveryOf(t, f, notification.ChannelEmail); got.State != notification.DeliverySent || !got.SentAt.Equal(afterChange.Add(-24*time.Hour)) {
		t.Errorf("delivery %+v, want sent", got)
	}
}

// The push is generic: no clinic, time or payload.
func TestSendDelivery_TheReminderPushIsGeneric(t *testing.T) {
	f := dispatched(t, afterChange.Add(-time.Hour))
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelPush).ID, false); err != nil {
		t.Fatal(err)
	}
	sent := f.pusher.Sent()
	if len(sent) != 1 || sent[0].Token != "synthetic-token-r" || sent[0].Title != "Salus" || sent[0].Body != "Imate podsjetnik na termin" {
		t.Fatalf("pushed %+v, want the generic reminder", sent)
	}
}

// The SMS goes to the patient's phone with the clinic, the date and time in its zone and a one-time link for the SMS,
// referenced by the delivery, and counts in the clinic's month in its own scope.
func TestSendDelivery_TheReminderSMSCountsInTheClinicsUsage(t *testing.T) {
	f := dispatched(t, afterChange.Add(-time.Hour))
	d := deliveryOf(t, f, notification.ChannelSMS)
	if err := f.svc.SendDelivery(t.Context(), d.ID, false); err != nil {
		t.Fatal(err)
	}
	sent := f.sms.Sent()
	want := "Podsjetnik: Synthetic clinic, 25.10.2026. u 09:00. Potvrdite ili otkazite: " + fakeLink(1)
	if len(sent) != 1 || sent[0].To != remindPhone || sent[0].Body != want || sent[0].Reference != d.ID {
		t.Fatalf("sent %+v, want %q to the phone", sent, want)
	}
	if n := sms.Segments(sent[0].Body); n != 1 {
		t.Errorf("the SMS takes %d parts, want 1", n)
	}
	if f.reminders.links[0].Channel != appointment.ChannelSMS {
		t.Errorf("link %+v, want one for the SMS", f.reminders.links[0])
	}
	u := f.deliveries.usage[remindClin+"2026-10"]
	if u.Messages != 1 || u.Segments != 1 || !slices.Equal(f.deliveries.usageScope, []string{remindClin}) {
		t.Errorf("usage %+v in %v, want one message of one part in the clinic's scope", u, f.deliveries.usageScope)
	}
}

// A delivery whose reminder is gone (the appointment cancelled or moved) is cancelled and nothing sent; a refusal no
// retry changes fails it; another failure is retried, and fails it on the last attempt.
func TestSendDelivery_CancelsWhatNoLongerAppliesAndFailsWhatCannotGo(t *testing.T) {
	f := dispatched(t, afterChange.Add(-time.Hour))
	f.reminders.noticeErr = appointment.ErrReminderGone
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelEmail).ID, false); err != nil {
		t.Fatal(err)
	}
	if len(f.sender.sent) != 0 || deliveryOf(t, f, notification.ChannelEmail).State != notification.DeliveryCancelled {
		t.Errorf("sent %v, delivery %v; want nothing and cancelled", f.sender.sent, deliveries(f))
	}

	f = dispatched(t, afterChange.Add(-time.Hour))
	f.sms.Unreachable(remindPhone)
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelSMS).ID, false); err != nil {
		t.Fatal(err)
	}
	if got := deliveryOf(t, f, notification.ChannelSMS); got.State != notification.DeliveryFailed || len(f.deliveries.usage) != 0 {
		t.Errorf("an unreachable number: %+v, usage %v; want failed and not counted", got, f.deliveries.usage)
	}

	f = dispatched(t, afterChange.Add(-time.Hour))
	down := errors.New("the relay is down")
	f.sender.err = down
	id := deliveryOf(t, f, notification.ChannelEmail).ID
	if err := f.svc.SendDelivery(t.Context(), id, false); !errors.Is(err, down) || deliveryOf(t, f, notification.ChannelEmail).State != notification.DeliveryPending {
		t.Errorf("a failing relay: %v, %v; want the error for a retry, still pending", err, deliveries(f))
	}
	if err := f.svc.SendDelivery(t.Context(), id, true); !errors.Is(err, down) || deliveryOf(t, f, notification.ChannelEmail).State != notification.DeliveryFailed {
		t.Errorf("the last attempt: %v, %v; want failed", err, deliveries(f))
	}
	if strings.Contains(down.Error(), "example.test") {
		t.Error("the error holds the address")
	}
}

// sendReminderEmail sends the reminder e-mail in language to sender, with or without an outstanding questionnaire, at
// the clinic's own zone (one time only), for the e-mails' golden files in every language.
func sendReminderEmail(t *testing.T, language string, sender *fakeSender, outstanding bool) {
	t.Helper()
	start := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC) // the other e-mails' appointment: 10:00 in Sarajevo
	f := newReminderFixture(start.Add(-24*time.Hour), start, fakeRules{})
	f.deliveries.prefs[patientAccount] = notification.Preferences{Channels: notification.Channels{Email: true}}
	f.q.outstanding = outstanding
	f.svc = notification.NewService(notification.Ports{
		Accounts: &fakeChatAccounts{language: map[string]string{patientAccount: language}}, Clinics: sarajevo{}, Questionnaires: f.q, Sender: sender,
		PatientAppURL: "https://patient.example.test", Deliveries: f.deliveries, DeliveryJobs: f.jobs, Devices: f.repo, Reminders: f.reminders,
		Rules: fakeRules{}, Languages: &fakeChatAccounts{}, Audit: f.audit, Tx: fakeChatTx{}, Clock: clockAt(start.Add(-24 * time.Hour)),
	})
	if err := f.svc.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: reminderID, OrganizationID: remindClin}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelEmail).ID, false); err != nil {
		t.Fatalf("%s: %v", language, err)
	}
}

// The SMS in every language fits one part with a clinic name of plain letters (the Latin-script languages drop their
// diacritics; Cyrillic and Arabic cannot), and is its golden file.
func TestReminderSMS_IsWrittenInEveryLanguage(t *testing.T) {
	for _, lang := range languages {
		f := dispatched(t, afterChange.Add(-time.Hour))
		f.svc = notification.NewService(notification.Ports{
			Accounts: &fakeChatAccounts{}, Clinics: sarajevo{}, Questionnaires: f.q, Sender: f.sender, Deliveries: f.deliveries,
			DeliveryJobs: f.jobs, Devices: f.repo, Reminders: f.reminders, Rules: fakeRules{sms: true},
			Languages: &fakeChatAccounts{language: map[string]string{patientAccount: lang}}, Phones: fakePhones{phone: remindPhone},
			SMS: f.sms, Audit: f.audit, Tx: fakeChatTx{}, Clock: clockAt(afterChange.Add(-time.Hour)),
		})
		if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelSMS).ID, false); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		body := f.sms.Sent()[0].Body
		golden(t, lang+"/reminder_sms.txt", body+"\n")
		if n := sms.Segments(body); n != 1 && lang != "sr-Cyrl" && lang != "ar" {
			t.Errorf("%s: %d parts, want 1:\n%s", lang, n, body)
		}
		if n := sms.Segments(body); n > 2 {
			t.Errorf("%s: %d parts, want at most 2", lang, n)
		}
	}
}
