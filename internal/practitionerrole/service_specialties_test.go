package practitionerrole_test

import (
	"errors"
	"maps"
	"testing"
)

// Specialties gives the specialty of each role a clinic's scope shows (its own, and those of other clinics' visits
// shared with it), for the history's items: a role it does not show is left out, one without a specialty has "".
// Unaudited, like Lookup.
func TestSpecialties_GivesTheSpecialtiesOfTheRolesTheScopeShows(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.roles[0].SpecialtyCode = "419192003"
	audit := &fakeAuditor{}
	svc := newService(repo, audit)
	got, err := svc.Specialties(t.Context(), clinicA, []string{roleA1, roleA2, roleB})
	if want := map[string]string{roleA1: "419192003", roleA2: ""}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Specialties() = %v, %v; want %v", got, err, want)
	}
	repo.leak = true // practitioner_role_shared shows clinic B's role
	got, err = svc.Specialties(t.Context(), clinicA, []string{roleB})
	if want := map[string]string{roleB: ""}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Specialties(shared) = %v, %v; want %v", got, err, want)
	}
	if repo.clinic != clinicA || len(audit.events) != 0 {
		t.Errorf("read in %q with %d events, want clinic A's scope and none", repo.clinic, len(audit.events))
	}
	repo.err = errors.New("the database is down")
	if _, err := svc.Specialties(t.Context(), clinicA, []string{roleA1}); !errors.Is(err, repo.err) {
		t.Errorf("Specialties(down) = %v, want the failure", err)
	}
}
