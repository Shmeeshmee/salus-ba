package organization_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const patientA = "da000000-0000-4000-8000-0000000000da"

// fakeChatRepository is fakeRepository with the clinics' chat settings, by clinic, shown only for the clinic set.
type fakeChatRepository struct {
	*fakeRepository
	settings map[string]organization.ChatSettings
	putAt    time.Time
}

func newFakeChatRepository() *fakeChatRepository {
	return &fakeChatRepository{fakeRepository: newFakeRepository(), settings: map[string]organization.ChatSettings{}}
}

func (r *fakeChatRepository) ChatSettings(ctx context.Context, org string) (organization.ChatSettings, bool, error) {
	r.seen(ctx, "ChatSettings")
	if r.err != nil {
		return organization.ChatSettings{}, false, r.err
	}
	s, ok := r.settings[org]
	if org != r.clinic {
		return organization.ChatSettings{}, false, nil
	}
	return s, ok, nil
}

func (r *fakeChatRepository) PutChatSettings(ctx context.Context, org string, s organization.ChatSettings, at time.Time) error {
	r.seen(ctx, "PutChatSettings")
	if org != r.clinic {
		return errors.New("fake: outside the scope")
	}
	r.settings[org], r.putAt = s, at
	return nil
}

// fakeConsents gives the consent state of a patient with a clinic, as the clinic's scope shows it.
type fakeConsents struct {
	active map[string]bool // patient + "/" + clinic
	asked  []string
}

func (f *fakeConsents) State(_ context.Context, patientID, org string) (access.Consent, error) {
	f.asked = append(f.asked, patientID+"/"+org)
	return access.Consent{Active: f.active[patientID+"/"+org]}, nil
}

func newChat(repo *fakeChatRepository, consents *fakeConsents, audit *fakeAuditor, at time.Time) *organization.Chat {
	return organization.NewChat(organization.ChatPorts{Repo: repo, Consents: consents, Audit: audit, Tx: fakeTransactor{}, Clock: clock.Fixed{T: at}})
}

var weekdays = []organization.ChatHour{
	{Day: "mon", Start: "08:00", End: "12:00"}, {Day: "mon", Start: "13:00", End: "16:00"},
	{Day: "tue", Start: "08:00", End: "16:00"}, {Day: "sat", Start: "09:00", End: "24:00"},
}

// The admin reads the defaults until they set the clinic's own; a valid change is stored and audited as an update
// of the clinic.
func TestChat_TheAdminReadsAndSetsTheClinicsChatSettings(t *testing.T) {
	repo, audit := newFakeChatRepository(), &fakeAuditor{}
	chat := newChat(repo, &fakeConsents{}, audit, now)
	admin := organization.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}

	got, err := chat.Settings(t.Context(), admin)
	if err != nil || len(got.Hours) != 0 || got.ResponseTimeHours != 24 || got.EmailAfterMinutes != 60 {
		t.Fatalf("Settings() = %+v, %v; want the defaults", got, err)
	}
	in := organization.ChatSettings{Hours: weekdays, ResponseTimeHours: 8, EmailAfterMinutes: 30}
	got, err = chat.SetSettings(t.Context(), admin, in)
	if err != nil || !slices.Equal(got.Hours, weekdays) || got.ResponseTimeHours != 8 || got.EmailAfterMinutes != 30 {
		t.Fatalf("SetSettings() = %+v, %v", got, err)
	}
	if !slices.Equal(repo.settings[clinicA].Hours, weekdays) || !repo.putAt.Equal(now) || repo.clinic != clinicA {
		t.Errorf("stored %+v at %v in %s", repo.settings[clinicA], repo.putAt, repo.clinic)
	}
	if got, _ := chat.Settings(t.Context(), admin); got.EmailAfterMinutes != 30 {
		t.Errorf("Settings() after the change = %+v", got)
	}
	if len(audit.events) != 1 || audit.events[0].EntityReference != "Organization/"+clinicA || audit.events[0].Action != auditevent.ActionUpdate ||
		audit.events[0].AgentAccountID != adminA || audit.events[0].Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit %+v, want one update of the clinic by its admin", audit.events)
	}
	for _, call := range repo.calls {
		if len(call) > 8 && call[:8] == "outside:" {
			t.Errorf("%s outside a transaction", call)
		}
	}
}

// Only the clinic's admin reads or changes the settings; staff get forbidden, anyone else not found. A refused
// change is audited.
func TestChat_OnlyTheAdminTouchesTheSettings(t *testing.T) {
	for name, tc := range map[string]struct {
		want error
		c    organization.Caller
	}{
		"a practitioner": {organization.ErrForbidden, organization.Caller{AccountID: adminA, OrganizationID: clinicA}},
		"a patient":      {organization.ErrNotFound, organization.Caller{AccountID: adminA, PatientID: patientA}},
	} {
		repo, audit := newFakeChatRepository(), &fakeAuditor{}
		chat := newChat(repo, &fakeConsents{}, audit, now)
		if _, err := chat.Settings(t.Context(), tc.c); !errors.Is(err, tc.want) {
			t.Errorf("%s: Settings() = %v, want %v", name, err, tc.want)
		}
		if _, err := chat.SetSettings(t.Context(), tc.c, organization.ChatSettings{ResponseTimeHours: 8, EmailAfterMinutes: 30}); !errors.Is(err, tc.want) {
			t.Errorf("%s: SetSettings() = %v, want %v", name, err, tc.want)
		}
		if len(repo.settings) != 0 || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: stored %v, audited %+v; want nothing stored and the refusal audited", name, repo.settings, audit.events)
		}
	}
}

func TestChat_RefusesSettingsOutOfTheirRanges(t *testing.T) {
	admin := organization.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}
	for name, in := range map[string]organization.ChatSettings{
		"response time 0":       {ResponseTimeHours: 0, EmailAfterMinutes: 60},
		"response time 169":     {ResponseTimeHours: 169, EmailAfterMinutes: 60},
		"e-mail after 4":        {ResponseTimeHours: 24, EmailAfterMinutes: 4},
		"e-mail after a week+":  {ResponseTimeHours: 24, EmailAfterMinutes: 10081},
		"unknown day":           {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "monday", Start: "08:00", End: "16:00"}}},
		"not a time":            {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "mon", Start: "8:00", End: "16:00"}}},
		"25 o'clock":            {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "mon", Start: "08:00", End: "25:00"}}},
		"start at 24:00":        {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "mon", Start: "24:00", End: "24:00"}}},
		"ends before it starts": {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "mon", Start: "16:00", End: "08:00"}}},
		"overlap":               {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: []organization.ChatHour{{Day: "mon", Start: "08:00", End: "12:00"}, {Day: "mon", Start: "11:00", End: "13:00"}}},
		"too many":              {ResponseTimeHours: 24, EmailAfterMinutes: 60, Hours: slices.Repeat([]organization.ChatHour{{Day: "mon", Start: "08:00", End: "08:01"}}, 22)},
	} {
		repo := newFakeChatRepository()
		_, err := newChat(repo, &fakeConsents{}, &fakeAuditor{}, now).SetSettings(t.Context(), admin, in)
		if !errors.Is(err, organization.ErrInvalidChatSettings) || len(repo.settings) != 0 {
			t.Errorf("%s: SetSettings() = %v, stored %v", name, err, repo.settings)
		}
	}
}

// The chat hours go to the clinic's staff and to a patient with an active care consent with it, with whether the
// clinic is open now in its own time zone; nobody else learns anything (not found).
func TestChat_HoursGoToTheClinicsStaffAndConsentingPatientsWithOpenNowInTheClinicsZone(t *testing.T) {
	repo := newFakeChatRepository()
	repo.settings[clinicA] = organization.ChatSettings{Hours: weekdays, ResponseTimeHours: 8, EmailAfterMinutes: 30}
	consents := &fakeConsents{active: map[string]bool{patientA + "/" + clinicA: true}}
	// Monday 2026-11-16, in Sarajevo (UTC+1 in November).
	for at, open := range map[string]bool{
		"2026-11-16T06:59:00Z": false, // 07:59
		"2026-11-16T07:00:00Z": true,  // 08:00
		"2026-11-16T11:30:00Z": false, // 12:30, the break
		"2026-11-16T14:59:00Z": true,  // 15:59
		"2026-11-16T15:00:00Z": false, // 16:00
		"2026-11-17T07:00:00Z": true,  // Tuesday 08:00
		"2026-11-21T22:59:00Z": true,  // Saturday 23:59
		"2026-11-22T10:00:00Z": false, // Sunday
	} {
		when, _ := time.Parse(time.RFC3339, at)
		chat := newChat(repo, consents, &fakeAuditor{}, when)
		for _, c := range []organization.Caller{{AccountID: adminA, OrganizationID: clinicA}, {AccountID: adminA, PatientID: patientA}} {
			h, err := chat.Hours(t.Context(), c, clinicA)
			if err != nil || h.OpenNow != open || h.ResponseTimeHours != 8 || h.TimeZone != "Europe/Sarajevo" || !slices.Equal(h.Hours, weekdays) {
				t.Errorf("%s as %+v: Hours() = %+v, %v; want open %v", at, c, h, err, open)
			}
		}
	}
	chat := newChat(repo, consents, &fakeAuditor{}, now)
	for name, c := range map[string]organization.Caller{
		"another clinic's staff":         {AccountID: adminA, OrganizationID: clinicB},
		"a patient without consent":      {AccountID: adminA, PatientID: "db000000-0000-4000-8000-0000000000db"},
		"a signed-in account, no person": {AccountID: adminA},
	} {
		if _, err := chat.Hours(t.Context(), c, clinicA); !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("%s: Hours() = %v, want ErrNotFound", name, err)
		}
	}
	if consents.asked[len(consents.asked)-1] != "db000000-0000-4000-8000-0000000000db/"+clinicA {
		t.Errorf("consent asked %v", consents.asked)
	}
}

// Without hours the clinic states none: the app shows no out-of-hours notice, so it is open.
func TestChat_HoursWithoutHoursSetAreOpen(t *testing.T) {
	h, err := newChat(newFakeChatRepository(), &fakeConsents{}, &fakeAuditor{}, now).Hours(t.Context(), organization.Caller{OrganizationID: clinicA}, clinicA)
	if err != nil || !h.OpenNow || h.Hours == nil || len(h.Hours) != 0 || h.ResponseTimeHours != 24 {
		t.Errorf("Hours() = %+v, %v", h, err)
	}
}

// The notifier's delay of the e-mail fallback, in the clinic's scope, the default for a clinic without settings.
func TestChat_EmailAfterIsTheClinicsDelay(t *testing.T) {
	repo := newFakeChatRepository()
	chat := newChat(repo, &fakeConsents{}, &fakeAuditor{}, now)
	if d, err := chat.EmailAfter(t.Context(), clinicA); err != nil || d != time.Hour {
		t.Errorf("EmailAfter(default) = %v, %v", d, err)
	}
	repo.settings[clinicA] = organization.ChatSettings{ResponseTimeHours: 8, EmailAfterMinutes: 15}
	if d, err := chat.EmailAfter(t.Context(), clinicA); err != nil || d != 15*time.Minute || repo.clinic != clinicA {
		t.Errorf("EmailAfter() = %v, %v in %s", d, err, repo.clinic)
	}
}
