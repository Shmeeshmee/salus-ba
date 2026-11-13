package organization_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeAppointmentsRepository is fakeRepository with the clinics' appointment settings, shown only for the clinic set.
type fakeAppointmentsRepository struct {
	*fakeRepository
	settings map[string]organization.AppointmentSettings
	putAt    time.Time
}

func newFakeAppointmentsRepository() *fakeAppointmentsRepository {
	return &fakeAppointmentsRepository{fakeRepository: newFakeRepository(), settings: map[string]organization.AppointmentSettings{}}
}

func (r *fakeAppointmentsRepository) AppointmentSettings(ctx context.Context, org string) (organization.AppointmentSettings, bool, error) {
	r.seen(ctx, "AppointmentSettings")
	if r.err != nil {
		return organization.AppointmentSettings{}, false, r.err
	}
	if org != r.clinic {
		return organization.AppointmentSettings{}, false, nil
	}
	s, ok := r.settings[org]
	return s, ok, nil
}

func (r *fakeAppointmentsRepository) PutAppointmentSettings(ctx context.Context, org string, s organization.AppointmentSettings, at time.Time) (int, error) {
	r.seen(ctx, "PutAppointmentSettings")
	if org != r.clinic {
		return 0, errors.New("fake: outside the scope")
	}
	if s.Version != r.settings[org].Version {
		return 0, organization.ErrSettingsChanged
	}
	s.Version++
	r.settings[org], r.putAt = s, at
	return s.Version, nil
}

func newAppointments(repo *fakeAppointmentsRepository, audit *fakeAuditor) *organization.Appointments {
	return organization.NewAppointments(organization.AppointmentsPorts{Repo: repo, Audit: audit, Tx: fakeTransactor{}, Clock: clock.Fixed{T: now}, SMSAvailable: true})
}

// While the SMS engine is the fake, the paid option cannot be turned on: no usage for messages never sent (review,
// finding 9).
func TestAppointments_RefusesTheSMSOptionWhileNoProviderSends(t *testing.T) {
	repo := newFakeAppointmentsRepository()
	svc := organization.NewAppointments(organization.AppointmentsPorts{Repo: repo, Audit: &fakeAuditor{}, Tx: fakeTransactor{}, Clock: clock.Fixed{T: now}})
	if _, err := svc.SetSettings(t.Context(), adminOfA, customSet); !errors.Is(err, organization.ErrSMSUnavailable) || len(repo.settings) != 0 {
		t.Errorf("SetSettings(sms on) = %v, stored %v; want ErrSMSUnavailable and nothing", err, repo.settings)
	}
	off := customSet
	off.SMSReminders = false
	if _, err := svc.SetSettings(t.Context(), adminOfA, off); err != nil {
		t.Errorf("SetSettings(sms off) = %v", err)
	}
}

var (
	adminOfA  = organization.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}
	customSet = organization.AppointmentSettings{
		ReminderOffsetsMinutes: []int{2880, 120, 15}, SMSReminders: true, RescheduleNoticeHours: 48, RescheduleMax: 1,
		WaitlistEnabled: true, WaitlistHoldMinutes: 30, NoShowFlagThreshold: 3, NoShowFlagWindowDays: 365,
	}
)

// The admin reads the defaults (24 h and 1 h, no SMS, 24 h notice twice, no waitlist with a 15-minute hold, no flag)
// until they set the clinic's own; a change is stored and audited as an update of the clinic; others read it through
// For.
func TestAppointments_TheAdminReadsAndSetsTheClinicsSettings(t *testing.T) {
	repo, audit := newFakeAppointmentsRepository(), &fakeAuditor{}
	svc := newAppointments(repo, audit)
	got, err := svc.Settings(t.Context(), adminOfA)
	want := organization.AppointmentSettings{ReminderOffsetsMinutes: []int{1440, 60}, RescheduleNoticeHours: 24, RescheduleMax: 2,
		WaitlistHoldMinutes: 15, NoShowFlagWindowDays: 180}
	if err != nil || !same(got, want) {
		t.Fatalf("Settings() = %+v, %v; want the defaults %+v", got, err, want)
	}
	if got, err = svc.SetSettings(t.Context(), adminOfA, customSet); err != nil || !same(got, customSet) {
		t.Fatalf("SetSettings() = %+v, %v", got, err)
	}
	if got.Version != 1 {
		t.Errorf("SetSettings() version %d, want 1 after the first save", got.Version)
	}
	if !same(repo.settings[clinicA], customSet) || !repo.putAt.Equal(now) {
		t.Errorf("stored %+v at %v", repo.settings[clinicA], repo.putAt)
	}
	if got, err := svc.For(t.Context(), clinicA); err != nil || !same(got, customSet) {
		t.Errorf("For() = %+v, %v; want the stored settings", got, err)
	}
	if got, _ := svc.For(t.Context(), clinicB); !same(got, want) {
		t.Errorf("For(clinic B) = %+v, want the defaults", got)
	}
	if len(audit.events) != 1 || audit.events[0].EntityReference != "Organization/"+clinicA || audit.events[0].Action != auditevent.ActionUpdate ||
		audit.events[0].Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit %+v, want one update of the clinic", audit.events)
	}
	for _, call := range repo.calls {
		if len(call) > 8 && call[:8] == "outside:" {
			t.Errorf("%s outside a transaction", call)
		}
	}
}

func same(a, b organization.AppointmentSettings) bool {
	return slices.Equal(a.ReminderOffsetsMinutes, b.ReminderOffsetsMinutes) && a.SMSReminders == b.SMSReminders &&
		a.RescheduleNoticeHours == b.RescheduleNoticeHours && a.RescheduleMax == b.RescheduleMax && a.WaitlistEnabled == b.WaitlistEnabled &&
		a.WaitlistHoldMinutes == b.WaitlistHoldMinutes && a.NoShowFlagThreshold == b.NoShowFlagThreshold &&
		a.NoShowFlagWindowDays == b.NoShowFlagWindowDays
}

func TestAppointments_OnlyTheAdminTouchesTheSettings(t *testing.T) {
	for name, tc := range map[string]struct {
		want error
		c    organization.Caller
	}{
		"a practitioner": {organization.ErrForbidden, organization.Caller{AccountID: adminA, OrganizationID: clinicA}},
		"a patient":      {organization.ErrNotFound, organization.Caller{AccountID: adminA, PatientID: patientA}},
	} {
		repo, audit := newFakeAppointmentsRepository(), &fakeAuditor{}
		svc := newAppointments(repo, audit)
		if _, err := svc.Settings(t.Context(), tc.c); !errors.Is(err, tc.want) {
			t.Errorf("%s: Settings() = %v, want %v", name, err, tc.want)
		}
		if _, err := svc.SetSettings(t.Context(), tc.c, customSet); !errors.Is(err, tc.want) {
			t.Errorf("%s: SetSettings() = %v, want %v", name, err, tc.want)
		}
		if len(repo.settings) != 0 || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: stored %v, audited %+v; want the refusal audited only", name, repo.settings, audit.events)
		}
	}
}

func TestAppointments_RefusesSettingsOutOfTheirRanges(t *testing.T) {
	for name, change := range map[string]func(*organization.AppointmentSettings){
		"four reminders":      func(s *organization.AppointmentSettings) { s.ReminderOffsetsMinutes = []int{15, 30, 60, 120} },
		"a reminder twice":    func(s *organization.AppointmentSettings) { s.ReminderOffsetsMinutes = []int{60, 60} },
		"14 minutes before":   func(s *organization.AppointmentSettings) { s.ReminderOffsetsMinutes = []int{14} },
		"over a week before":  func(s *organization.AppointmentSettings) { s.ReminderOffsetsMinutes = []int{10081} },
		"notice -1":           func(s *organization.AppointmentSettings) { s.RescheduleNoticeHours = -1 },
		"notice 169":          func(s *organization.AppointmentSettings) { s.RescheduleNoticeHours = 169 },
		"six reschedules":     func(s *organization.AppointmentSettings) { s.RescheduleMax = 6 },
		"hold 4":              func(s *organization.AppointmentSettings) { s.WaitlistHoldMinutes = 4 },
		"hold 121":            func(s *organization.AppointmentSettings) { s.WaitlistHoldMinutes = 121 },
		"threshold 11":        func(s *organization.AppointmentSettings) { s.NoShowFlagThreshold = 11 },
		"window 29 days":      func(s *organization.AppointmentSettings) { s.NoShowFlagWindowDays = 29 },
		"window over 2 years": func(s *organization.AppointmentSettings) { s.NoShowFlagWindowDays = 731 },
	} {
		in := organization.DefaultAppointmentSettings()
		change(&in)
		repo := newFakeAppointmentsRepository()
		if _, err := newAppointments(repo, &fakeAuditor{}).SetSettings(t.Context(), adminOfA, in); !errors.Is(err, organization.ErrInvalidAppointmentSettings) || len(repo.settings) != 0 {
			t.Errorf("%s: SetSettings() = %v, stored %v; want ErrInvalidAppointmentSettings and nothing", name, err, repo.settings)
		}
	}
	none := organization.DefaultAppointmentSettings()
	none.ReminderOffsetsMinutes = []int{}
	if _, err := newAppointments(newFakeAppointmentsRepository(), &fakeAuditor{}).SetSettings(t.Context(), adminOfA, none); err != nil {
		t.Errorf("no reminders: %v, want them allowed", err)
	}
}

func TestAppointments_FailsWithTheBackend(t *testing.T) {
	repo := newFakeAppointmentsRepository()
	repo.err = errors.New("the database is down")
	if _, err := newAppointments(repo, &fakeAuditor{}).Settings(t.Context(), adminOfA); err == nil {
		t.Error("Settings() with the backend down succeeded")
	}
	audit := &fakeAuditor{err: errors.New("audit down")}
	if _, err := newAppointments(newFakeAppointmentsRepository(), audit).SetSettings(t.Context(), organization.Caller{OrganizationID: clinicA}, customSet); err == nil || errors.Is(err, organization.ErrForbidden) {
		t.Errorf("SetSettings() with the audit down = %v, want the failure", err)
	}
}

// A save names the version it read: one read at an older version is refused, stores nothing and is not audited as a
// change; the next save reads the version the last one returned.
func TestAppointments_RefusesASaveOfAnOlderVersion(t *testing.T) {
	repo, audit := newFakeAppointmentsRepository(), &fakeAuditor{}
	svc := newAppointments(repo, audit)
	read, err := svc.Settings(t.Context(), adminOfA)
	if err != nil || read.Version != 0 {
		t.Fatalf("Settings() = version %d, %v; want 0 while none is stored", read.Version, err)
	}
	first := customSet
	first.Version = read.Version
	saved, err := svc.SetSettings(t.Context(), adminOfA, first)
	if err != nil || saved.Version != 1 {
		t.Fatalf("first SetSettings() = version %d, %v", saved.Version, err)
	}
	stale := organization.DefaultAppointmentSettings()
	stale.Version = read.Version
	if _, err := svc.SetSettings(t.Context(), adminOfA, stale); !errors.Is(err, organization.ErrSettingsChanged) {
		t.Fatalf("SetSettings(read at 0) after a save = %v, want ErrSettingsChanged", err)
	}
	if got := repo.settings[clinicA]; !same(got, customSet) || got.Version != 1 {
		t.Errorf("stored %+v, want the first save's", got)
	}
	if len(audit.events) != 1 {
		t.Errorf("audit %+v, want only the first save", audit.events)
	}
	again := stale
	again.Version = saved.Version
	if got, err := svc.SetSettings(t.Context(), adminOfA, again); err != nil || got.Version != 2 {
		t.Errorf("SetSettings(read at 1) = version %d, %v; want 2", got.Version, err)
	}
}
