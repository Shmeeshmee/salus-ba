package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx, "SetDirectory")
	r.directory = true
	return nil
}

// PublicByOrganization shows the roles of org in the directory scope, all of them when leak is set (a
// database without the policy); whether the clinic is listed is the database's part.
func (r *fakeRepository) PublicByOrganization(ctx context.Context, org string) ([]practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "PublicByOrganization")
	if r.err != nil {
		return nil, r.err
	}
	var out []practitionerrole.PractitionerRole
	for _, p := range r.roles {
		if r.leak || (r.directory && p.OrganizationID == org && p.Active) {
			out = append(out, p)
		}
	}
	return out, nil
}

func TestPublicByOrganization_ListsTheActiveRolesInTheDirectoryScopeWithoutAudit(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.roles[0].SpecialtyCode = cardiology
	got, err := newService(repo, audit).PublicByOrganization(t.Context(), clinicA)
	want := []practitionerrole.PractitionerRole{repo.roles[0], repo.roles[2]}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicByOrganization(clinic A) = %+v, %v;\nwant %+v", got, err, want)
	}
	if !slices.Equal(repo.calls, []string{"SetDirectory", "PublicByOrganization"}) {
		t.Errorf("calls = %v, want the directory scope and the query in one transaction", repo.calls)
	}
	if len(audit.events) != 0 {
		t.Errorf("events = %+v, want none: a public read is not audited", audit.events)
	}
}

func TestPublicByOrganization_KeepsOnlyActiveRolesOfTheClinicEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.leak = true
	repo.roles[2].Active = false
	got, err := newService(repo, &fakeAuditor{}).PublicByOrganization(t.Context(), clinicA)
	if err != nil || !slices.Equal(got, repo.roles[:1]) {
		t.Errorf("PublicByOrganization(clinic A) without the policy = %+v, %v; want only the active role A1", got, err)
	}
}

func TestPublicByOrganization_FailsOnARepositoryError(t *testing.T) {
	t.Parallel()
	broken := errors.New("synthetic failure")
	repo := newFakeRepository()
	repo.err = broken
	if _, err := newService(repo, &fakeAuditor{}).PublicByOrganization(t.Context(), clinicA); !errors.Is(err, broken) {
		t.Errorf("PublicByOrganization() = %v, want the repository error", err)
	}
}
