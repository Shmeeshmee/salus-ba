package organization_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// The clinic's default language is a staff language, set by its admin (docs/i18n/SPEC.md §3 "The clinic's
// default language").
func TestUpdate_SetsTheDefaultLanguageToAStaffLanguageAndAuditsIt(t *testing.T) {
	for _, language := range []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"} {
		repo, audit := newFakeRepository(), &fakeAuditor{}

		got, err := newService(repo, audit).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Synthetic clinic A", DefaultLanguage: &language}, 2)

		if err != nil || got.DefaultLanguage != language || repo.orgs[clinicA].DefaultLanguage != language || got.TimeZone != "Europe/Sarajevo" {
			t.Errorf("Update(%s) = %+v, %v; want the language stored and the rest kept", language, got, err)
		}
		want := updateEvent(admin, auditevent.OutcomeSuccess)
		want.EntityReference = "Organization/" + clinicA
		if len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("Update(%s): audit events %+v, want %+v", language, audit.events, want)
		}
	}
}

func TestUpdate_KeepsTheDefaultLanguageWhenNoneIsGiven(t *testing.T) {
	repo := newFakeRepository()
	o := repo.orgs[clinicA]
	o.DefaultLanguage = "hr"
	repo.orgs[clinicA] = o

	got, err := newService(repo, &fakeAuditor{}).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Renamed"}, 2)

	if err != nil || got.DefaultLanguage != "hr" {
		t.Errorf("Update() = %+v, %v; want the default language hr kept", got, err)
	}
}

func TestUpdate_RejectsADefaultLanguageThatIsNotAStaffLanguageBeforeAnyAccess(t *testing.T) {
	for _, language := range []string{"", "de", "ar", "tr", "it", "fr", "sr", "BS", "en-GB", "sr-latn", "xx"} {
		repo, audit := newFakeRepository(), &fakeAuditor{}

		_, err := newService(repo, audit).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Renamed", DefaultLanguage: &language}, 2)

		if !errors.Is(err, organization.ErrInvalidDefaultLanguage) || len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("Update(%q) = %v, calls %v, events %d; want ErrInvalidDefaultLanguage before any access",
				language, err, repo.calls, len(audit.events))
		}
	}
}

// DefaultLanguage is the port the questionnaire domain (a new questionnaire's original language) and the
// notification domain (an e-mail to staff with no language of their own) consume.
func TestDefaultLanguage_ReadsTheClinicsLanguageThroughItsScopeWithoutAuditing(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	o := repo.orgs[clinicB]
	o.DefaultLanguage = "sr-Cyrl"
	repo.orgs[clinicB] = o

	got, err := newService(repo, audit).DefaultLanguage(t.Context(), clinicB)

	if err != nil || got != "sr-Cyrl" {
		t.Errorf("DefaultLanguage() = %q, %v; want sr-Cyrl", got, err)
	}
	if want := []string{"SetOrganization", "ByID"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicB {
		t.Errorf("calls %v with clinic %q set, want %v inside the transaction with clinic B", repo.calls, repo.clinic, want)
	}
	if len(audit.events) != 0 {
		t.Errorf("audit events %+v, want none: a clinic holds no personal data", audit.events)
	}
}

func TestDefaultLanguage_GivesNotFoundForAnUnknownOrMissingClinic(t *testing.T) {
	for _, id := range []string{"", "d0000000-0000-4000-8000-00000000000d"} {
		repo := newFakeRepository()

		_, err := newService(repo, &fakeAuditor{}).DefaultLanguage(t.Context(), id)

		if !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("DefaultLanguage(%q) = %v, want ErrNotFound", id, err)
		}
	}
}

func TestDefaultLanguage_FailsOnARepositoryError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")

	_, err := newService(repo, &fakeAuditor{}).DefaultLanguage(t.Context(), clinicA)

	if err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("DefaultLanguage() = %v, want the repository error", err)
	}
}
