package organization_test

import (
	"errors"
	"maps"
	"testing"
)

// Names gives the names of the clinics a clinic's scope shows, its own and those of visits shared with it, for the
// history's items; a clinic it does not show is left out. Unaudited: a clinic is no personal data.
func TestNames_GivesTheNamesOfTheClinicsTheScopeShows(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	audit := &fakeAuditor{}
	svc := newService(repo, audit)
	got, err := svc.Names(t.Context(), clinicA, []string{clinicA, clinicB})
	if want := map[string]string{clinicA: "Synthetic clinic A"}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Names() = %v, %v; want %v", got, err, want)
	}
	repo.shared = []string{clinicB}
	got, err = svc.Names(t.Context(), clinicA, []string{clinicA, clinicB, clinicA})
	if want := map[string]string{clinicA: "Synthetic clinic A", clinicB: "Synthetic clinic B"}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Names(shared) = %v, %v; want %v", got, err, want)
	}
	if repo.clinic != clinicA || len(audit.events) != 0 {
		t.Errorf("read in %q with %d events, want clinic A's scope and none", repo.clinic, len(audit.events))
	}
	repo.err = errors.New("the database is down")
	if _, err := svc.Names(t.Context(), clinicA, []string{clinicA}); !errors.Is(err, repo.err) {
		t.Errorf("Names(down) = %v, want the failure", err)
	}
}
