package composition_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// SAL-39: the patient's timeline lists the signed documents of their own encounters (encounter.Service.Own), with
// each one's encounter, after the page's position, in their scope, never a draft, unaudited
// (history.Timeline audits the page it returns).
func TestPatientHistory_GivesThePatientTheirSignedDocuments(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	signed := f.signedOne(t)
	f.drafted(t) // a second draft, preliminary: never listed
	p := keyset.Page{At: now.Add(1), Ref: "Encounter/x", Limit: 7}
	got, err := f.svc.PatientHistory(t.Context(), me, "", p)
	if err != nil || len(got) != 1 || got[0].Draft.ID != signed.ID || got[0].Encounter.ID != encA {
		t.Fatalf("PatientHistory() = %+v, %v; want the signed document with its encounter", got, err)
	}
	if f.repo.history.page != p || !slices.Equal(f.repo.history.encounters, []string{encA}) {
		t.Errorf("asked %+v, want the page for encounter A", f.repo.history)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing: history.Timeline audits the page it returns", f.codes())
	}
}

// No encounter at the clinic asked (or none at all) is no document: nothing read or audited.
func TestPatientHistory_NoEncounterIsNone(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.signedOne(t)
	got, err := f.svc.PatientHistory(t.Context(), me, clinicB, keyset.Page{Limit: 7})
	if err != nil || len(got) != 0 || f.repo.history.encounters != nil {
		t.Fatalf("PatientHistory(clinic B) = %+v, %v, asked %+v; want none and nothing read", got, err, f.repo.history)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing", f.codes())
	}
}

// The timeline is the patient's: staff get ErrForbidden.
func TestPatientHistory_IsForbiddenToStaff(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.svc.PatientHistory(t.Context(), pract, "", keyset.Page{Limit: 7}); !errors.Is(err, composition.ErrForbidden) {
		t.Errorf("PatientHistory() = %v, want ErrForbidden", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing", f.codes())
	}
}

func TestPatientHistory_FailsWithTheBackend(t *testing.T) {
	t.Parallel()
	for name, fail := range map[string]func(f *fixture){
		"repository": func(f *fixture) { f.repo.err = errBackend },
		"encounters": func(f *fixture) { f.encounters.err = errBackend },
	} {
		f := newFixture(t)
		f.signedOne(t)
		fail(f)
		if _, err := f.svc.PatientHistory(t.Context(), me, "", keyset.Page{Limit: 7}); !errors.Is(err, errBackend) {
			t.Errorf("%s: PatientHistory() = %v, want its error", name, err)
		}
	}
}
