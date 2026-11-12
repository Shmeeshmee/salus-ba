package organization_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// The name of a clinic, for the clinic a shared visit is marked with (SAL-42): read in the clinic's scope, unaudited.
func TestName_GivesTheClinicsNameInItsScopeWithoutAnAudit(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Name(t.Context(), clinicB)
	if err != nil || got != "Synthetic clinic B" {
		t.Errorf("Name() = %q, %v; want Synthetic clinic B", got, err)
	}
	if want := []string{"SetOrganization", "ByID"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicB || len(audit.events) != 0 {
		t.Errorf("calls %v with clinic %q and %d events, want %v with clinic B and none", repo.calls, repo.clinic, len(audit.events), want)
	}
	if _, err := newService(newFakeRepository(), audit).Name(t.Context(), "d0000000-0000-4000-8000-00000000000d"); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Name() of an unknown clinic = %v, want ErrNotFound", err)
	}
}
