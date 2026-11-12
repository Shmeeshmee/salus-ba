package organization_test

import (
	"errors"
	"maps"
	"testing"
)

// SAL-39: Names gives the name of each clinic of the patient's own timeline, listed or not, each read in its own
// scope, unaudited (a clinic holds no personal data); an unknown clinic is left out.
func TestNames_GivesEachClinicsNameUnaudited(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Names(t.Context(), []string{clinicA, clinicB, clinicA, "d0000000-0000-4000-8000-00000000000d"})
	want := map[string]string{clinicA: "Synthetic clinic A", clinicB: "Synthetic clinic B"}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("Names() = %v, %v; want %v", got, err, want)
	}
	if len(repo.calls) != 6 || len(audit.events) != 0 {
		t.Errorf("calls %v, events %d; want each clinic once in its scope, no event", repo.calls, len(audit.events))
	}
	repo.err = errors.New("the database is down")
	if _, err := newService(repo, audit).Names(t.Context(), []string{clinicA}); !errors.Is(err, repo.err) {
		t.Errorf("Names() = %v, want the backend's error", err)
	}
}
