package organization_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// ProfileOf shows the profile of a clinic the scope shows (the clinic set, or a listed clinic in the directory).
func (r *fakeRepository) ProfileOf(ctx context.Context, id string) (organization.Profile, error) {
	r.seen(ctx, "ProfileOf")
	if r.err != nil {
		return organization.Profile{}, r.err
	}
	o, ok := r.orgs[id]
	if !ok || (id != r.clinic && !r.listed(o)) {
		return organization.Profile{}, organization.ErrNotFound
	}
	p := r.profiles[id]
	p.VersionID = o.VersionID
	return p, nil
}

func (r *fakeRepository) PutProfile(ctx context.Context, id string, p organization.Profile, at time.Time) (organization.Profile, error) {
	r.seen(ctx, "PutProfile")
	if r.updateErr != nil {
		return organization.Profile{}, r.updateErr
	}
	o := r.orgs[id]
	if id != r.clinic || o.VersionID != p.VersionID {
		return organization.Profile{}, organization.ErrConflict
	}
	o.VersionID, o.LastUpdated = o.VersionID+1, at
	r.orgs[id] = o
	p.VersionID = o.VersionID
	if r.profiles == nil {
		r.profiles = map[string]organization.Profile{}
	}
	r.profiles[id] = p
	return p, nil
}

func (r *fakeRepository) ListedSpeaking(ctx context.Context, language string) ([]organization.Organization, error) {
	r.seen(ctx, "ListedSpeaking")
	if r.err != nil {
		return nil, r.err
	}
	var out []organization.Organization
	for _, id := range []string{clinicA, clinicB} {
		if r.directory && slices.Contains(r.profiles[id].Languages, language) {
			out = append(out, r.orgs[id])
		}
	}
	return out, nil
}

var profileA = organization.Profile{
	Languages:             []string{"bs", "de"},
	Description:           "Synthetic description",
	Translations:          []organization.Translation{{Language: "de", Description: "Synthetische Beschreibung"}},
	WelcomesInternational: true,
}

func TestSaveProfile_StoresTheAdminsProfileAtTheNextVersionAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	in := profileA
	in.Description = "  Synthetic description \n"
	got, err := newService(repo, audit).SaveProfile(t.Context(), admin, in, 2)
	want := profileA
	want.VersionID = 3
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("SaveProfile() = %+v, %v; want %+v", got, err, want)
	}
	if w := []string{"SetOrganization", "PutProfile"}; !slices.Equal(repo.calls, w) || repo.clinic != clinicA {
		t.Errorf("calls %v in clinic %q, want %v in clinic A", repo.calls, repo.clinic, w)
	}
	ev := updateEvent(admin, auditevent.OutcomeSuccess)
	ev.EntityReference = "Organization/" + clinicA
	if len(audit.events) != 1 || audit.events[0] != ev {
		t.Errorf("audit events %+v, want %+v", audit.events, ev)
	}
}

func TestSaveProfile_RefusesAndAuditsAnyoneButTheClinicsAdmin(t *testing.T) {
	for _, tt := range []struct {
		want   error
		caller organization.Caller
	}{
		{caller: practitioner, want: organization.ErrForbidden},
		{caller: patient, want: organization.ErrNotFound},
	} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).SaveProfile(t.Context(), tt.caller, profileA, 2)
		if !errors.Is(err, tt.want) || len(repo.calls) != 0 {
			t.Errorf("SaveProfile() as %+v = %v after %v, want %v without touching the repository", tt.caller, err, repo.calls, tt.want)
		}
		if w := updateEvent(tt.caller, auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != w {
			t.Errorf("audit events %+v, want %+v", audit.events, w)
		}
	}
}

func TestSaveProfile_RefusesAnInvalidProfileBeforeAnythingIsAccessed(t *testing.T) {
	long := strings.Repeat("é", 2001)
	for name, p := range map[string]organization.Profile{
		"an unknown language":      {Languages: []string{"xx"}},
		"a language twice":         {Languages: []string{"de", "de"}},
		"a description too long":   {Description: long},
		"a translation's language": {Translations: []organization.Translation{{Language: "xx", Description: "x"}}},
		"a translation twice":      {Translations: []organization.Translation{{Language: "de", Description: "x"}, {Language: "de", Description: "y"}}},
		"a blank translation":      {Translations: []organization.Translation{{Language: "de", Description: "  "}}},
		"a translation too long":   {Translations: []organization.Translation{{Language: "de", Description: long}}},
		"a translation of nothing": {Translations: []organization.Translation{{Language: "de", Description: "x"}}, Description: ""},
	} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		if _, err := newService(repo, audit).SaveProfile(t.Context(), admin, p, 2); !errors.Is(err, organization.ErrInvalidProfile) ||
			len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("%s: SaveProfile() = %v after %v with events %v, want ErrInvalidProfile untouched", name, err, repo.calls, audit.events)
		}
	}
}

func TestSaveProfile_GivesAConflictForAStaleVersionAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	if _, err := newService(repo, audit).SaveProfile(t.Context(), admin, profileA, 1); !errors.Is(err, organization.ErrConflict) {
		t.Errorf("SaveProfile() at a stale version = %v, want ErrConflict", err)
	}
	if len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("audit events %+v, want one minor failure", audit.events)
	}
}

func TestSaveProfile_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.updateErr = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).SaveProfile(t.Context(), admin, profileA, 2); err == nil || errors.Is(err, organization.ErrConflict) {
		t.Errorf("SaveProfile() with a failing write = %v, want the repository error", err)
	}
	if _, err := newService(newFakeRepository(), &fakeAuditor{err: errors.New("audit down")}).SaveProfile(t.Context(), admin, profileA, 2); err == nil {
		t.Error("SaveProfile() with a failing audit = nil, want an error")
	}
}

func TestClinicProfile_GivesStaffTheirOwnClinicsProfile(t *testing.T) {
	for _, c := range []organization.Caller{admin, practitioner} {
		repo := newFakeRepository()
		repo.profiles = map[string]organization.Profile{clinicA: profileA}
		got, err := newService(repo, &fakeAuditor{}).ClinicProfile(t.Context(), c)
		want := profileA
		want.VersionID = 2
		if err != nil || !reflect.DeepEqual(got, want) || repo.clinic != clinicA {
			t.Errorf("ClinicProfile() as %+v = %+v, %v in %q; want %+v in clinic A", c, got, err, repo.clinic, want)
		}
	}
	if _, err := newService(newFakeRepository(), &fakeAuditor{}).ClinicProfile(t.Context(), patient); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ClinicProfile() as a patient = %v, want ErrNotFound", err)
	}
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).ClinicProfile(t.Context(), admin); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ClinicProfile() with a failing read = %v, want the repository error", err)
	}
}

func TestProfile_ShowsAListedClinicsProfileInTheDirectoryOnly(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles = map[string]organization.Profile{clinicB: profileA}
	got, err := newService(repo, &fakeAuditor{}).Profile(t.Context(), clinicB, nil)
	want := profileA
	want.VersionID = 1
	if err != nil || !reflect.DeepEqual(got, want) || !repo.directory {
		t.Errorf("Profile(listed clinic B) = %+v, %v (directory %v); want %+v in the directory scope", got, err, repo.directory, want)
	}
	// Clinic A is not listed; the fake's directory shows it, the service checks again.
	repo = newFakeRepository()
	repo.shared = []string{clinicA}
	for _, id := range []string{clinicA, "c0000000-0000-4000-8000-00000000000c", ""} {
		if _, err := newService(repo, &fakeAuditor{}).Profile(t.Context(), id, nil); !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("Profile(%q) = %v, want ErrNotFound", id, err)
		}
	}
	repo = newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Profile(t.Context(), clinicB, nil); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile() with a failing read = %v, want the repository error", err)
	}
}

func TestListListedSpeaking_GivesTheListedClinicsThatSpeakTheLanguage(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles = map[string]organization.Profile{clinicA: profileA, clinicB: profileA}
	got, err := newService(repo, &fakeAuditor{}).ListListedSpeaking(t.Context(), "de")
	if err != nil || len(got) != 1 || got[0].ID != clinicB {
		t.Errorf("ListListedSpeaking(de) = %+v, %v; want listed clinic B only", got, err)
	}
	if got, err := newService(repo, &fakeAuditor{}).ListListedSpeaking(t.Context(), "xx"); !errors.Is(err, organization.ErrInvalidProfile) || got != nil {
		t.Errorf("ListListedSpeaking(xx) = %+v, %v; want ErrInvalidProfile", got, err)
	}
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).ListListedSpeaking(t.Context(), "de"); err == nil {
		t.Error("ListListedSpeaking() with a failing read = nil, want an error")
	}
}
