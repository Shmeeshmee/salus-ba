package notification_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/patient"
)

var me = notification.Caller{AccountID: patientAccount}

// A patient's settings are the defaults until they change them; each read and change is audited on their account.
func TestPreferences_AreTheDefaultsUntilThePatientChangesThem(t *testing.T) {
	f := newReminderFixture(beforeChange, afterChange, fakeRules{})
	got, err := f.svc.Preferences(t.Context(), me)
	if err != nil || !got.Channels.Push || !got.Channels.Email || got.Channels.SMS || got.Quiet == nil || *got.Quiet != (notification.QuietHours{Start: "22:00", End: "08:00"}) || got.TimeZone != "" {
		t.Fatalf("Preferences() = %+v, %v; want the defaults", got, err)
	}
	in := notification.Preferences{Channels: notification.Channels{Push: true, SMS: true}, TimeZone: "Asia/Dubai"}
	if _, err := f.svc.SetPreferences(t.Context(), me, in); err != nil {
		t.Fatal(err)
	}
	got, err = f.svc.Preferences(t.Context(), me)
	if err != nil || got.Quiet != nil || got.TimeZone != "Asia/Dubai" || !got.Channels.SMS || got.Channels.Email {
		t.Errorf("after the change: %+v, %v", got, err)
	}
	if len(f.audit.events) != 3 || f.audit.events[1].Action != auditevent.ActionUpdate || f.audit.events[1].AgentAccountID != patientAccount ||
		f.audit.events[1].EntityReference != "Account/"+patientAccount {
		t.Errorf("audit %+v, want a read, an update and a read on the account", f.audit.events)
	}
}

// Staff have no settings; SMS needs a phone in the profile; quiet hours are HH:MM with two different ends and the
// zone an IANA one. Each refusal of a patient is audited.
func TestSetPreferences_RefusesWhatTheContractDoesNot(t *testing.T) {
	sms := notification.Channels{SMS: true}
	for _, tt := range []struct {
		want    error
		profile fakeProfiles
		in      notification.Preferences
		name    string
		c       notification.Caller
	}{
		{name: "staff", c: notification.Caller{AccountID: practAccount, OrganizationID: clinicID}, want: notification.ErrForbidden},
		{name: "SMS without a phone", c: me, in: notification.Preferences{Channels: sms}, want: notification.ErrPhoneRequired},
		{name: "SMS without a profile", c: me, in: notification.Preferences{Channels: sms}, profile: fakeProfiles{err: patient.ErrNotFound}, want: notification.ErrPhoneRequired},
		{name: "quiet hours of no length", c: me, in: notification.Preferences{Quiet: &notification.QuietHours{Start: "22:00", End: "22:00"}}, want: notification.ErrInvalid},
		{name: "quiet hours not HH:MM", c: me, in: notification.Preferences{Quiet: &notification.QuietHours{Start: "7:00", End: "22:00"}}, want: notification.ErrInvalid},
		{name: "an unknown zone", c: me, in: notification.Preferences{TimeZone: "Mars/Olympus"}, want: notification.ErrInvalid},
		{name: "the local zone", c: me, in: notification.Preferences{TimeZone: "Local"}, want: notification.ErrInvalid},
	} {
		f := newReminderFixture(beforeChange, afterChange, fakeRules{})
		f.svc = notification.NewService(notification.Ports{Deliveries: f.deliveries, Profiles: tt.profile, Audit: f.audit, Tx: fakeChatTx{}, Clock: clockAt(beforeChange)})
		if _, err := f.svc.SetPreferences(t.Context(), tt.c, tt.in); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", tt.name, err, tt.want)
		}
		if len(f.deliveries.prefs) != 0 {
			t.Errorf("%s: stored %v", tt.name, f.deliveries.prefs)
		}
		if tt.c.OrganizationID == "" && (len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure) {
			t.Errorf("%s: audit %+v, want the refusal", tt.name, f.audit.events)
		}
	}
	f := newReminderFixture(beforeChange, afterChange, fakeRules{})
	if _, err := f.svc.Preferences(t.Context(), notification.Caller{AccountID: practAccount, OrganizationID: clinicID}); !errors.Is(err, notification.ErrForbidden) {
		t.Errorf("staff read settings: %v", err)
	}
}

// The admin reads their clinic's SMS usage by month in its scope, audited; other staff and patients may not; months
// are YYYY-MM, to not before from.
func TestSMSUsage_IsTheAdminsByMonth(t *testing.T) {
	f := dispatched(t, afterChange.Add(-60*time.Minute))
	if err := f.svc.SendDelivery(t.Context(), deliveryOf(t, f, notification.ChannelSMS).ID, false); err != nil {
		t.Fatal(err)
	}
	admin := notification.Caller{AccountID: clinicAdmin, OrganizationID: remindClin, Admin: true}
	got, err := f.svc.SMSUsage(t.Context(), admin, "2026-01", "2026-12")
	if err != nil || len(got) != 1 || got[0] != (notification.UsageMonth{Month: "2026-10", Messages: 1, Segments: 2}) {
		t.Fatalf("SMSUsage() = %+v, %v", got, err)
	}
	if f.repo.clinic != remindClin {
		t.Errorf("read in %q, want the admin's clinic", f.repo.clinic)
	}
	if got, err := f.svc.SMSUsage(t.Context(), admin, "2025-01", "2025-02"); err != nil || got == nil || len(got) != 0 {
		t.Errorf("no usage: %+v, %v; want an empty list", got, err)
	}
	for name, c := range map[string]notification.Caller{"staff": {AccountID: practAccount, OrganizationID: remindClin}, "a patient": me} {
		if _, err := f.svc.SMSUsage(t.Context(), c, "2026-01", "2026-12"); !errors.Is(err, notification.ErrForbidden) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, r := range [][2]string{{"2026-12", "2026-01"}, {"2026-1", "2026-02"}, {"2000-01", "2026-01"}} {
		if _, err := f.svc.SMSUsage(t.Context(), admin, r[0], r[1]); !errors.Is(err, notification.ErrInvalid) {
			t.Errorf("%v: %v, want ErrInvalid", r, err)
		}
	}
}
