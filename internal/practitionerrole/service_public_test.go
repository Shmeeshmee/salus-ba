package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// fakeClinics reads a clinic as its own member would (organization.Service.Read): listed as given, every
// clinic it knows of.
type fakeClinics struct {
	err    error
	listed map[string]bool
}

func (c *fakeClinics) Read(_ context.Context, caller organization.Caller, id string) (organization.Organization, error) {
	if c.err != nil {
		return organization.Organization{}, c.err
	}
	listed, ok := c.listed[id]
	if !ok || caller.OrganizationID != id {
		return organization.Organization{}, organization.ErrNotFound
	}
	return organization.Organization{ID: id, Listed: listed}, nil
}

// fakePeople gives the practitioners among ids that are not inactive (practitioner.Service.PublicNames).
type fakePeople struct {
	err      error
	inactive map[string]bool
}

func (p *fakePeople) PublicNames(_ context.Context, ids []string) ([]practitioner.Practitioner, error) {
	if p.err != nil {
		return nil, p.err
	}
	var out []practitioner.Practitioner
	for _, id := range ids {
		if !p.inactive[id] {
			out = append(out, practitioner.Practitioner{ID: id, Active: true})
		}
	}
	return out, nil
}

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

// The service checks that the clinic is listed before it reads a role; row-level security is the second guard.
func TestPublicByOrganization_ShowsNothingOfAClinicThatIsNotListedEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	for name, clinics := range map[string]*fakeClinics{
		"unlisted": {listed: map[string]bool{clinicA: false}},
		"unknown":  {listed: map[string]bool{}},
	} {
		repo := newFakeRepository()
		repo.leak = true
		got, err := newDirectoryService(repo, &fakeAuditor{}, clinics, &fakePeople{}).PublicByOrganization(t.Context(), clinicA)
		if err != nil || len(got) != 0 {
			t.Errorf("%s: PublicByOrganization() = %+v, %v; want none", name, got, err)
		}
		if slices.Contains(repo.calls, "PublicByOrganization") {
			t.Errorf("%s: calls %v, want no role read", name, repo.calls)
		}
	}
}

func TestPublicByOrganization_KeepsOnlyRolesWhosePractitionerIsActive(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	people := &fakePeople{inactive: map[string]bool{repo.roles[2].PractitionerID: true}}
	got, err := newDirectoryService(repo, &fakeAuditor{}, &fakeClinics{listed: map[string]bool{clinicA: true}}, people).
		PublicByOrganization(t.Context(), clinicA)
	if err != nil || !slices.Equal(got, repo.roles[:1]) {
		t.Errorf("PublicByOrganization() = %+v, %v; want role A1 only", got, err)
	}
}

func TestPublicByOrganization_FailsWhenTheClinicOrPractitionersCannotBeRead(t *testing.T) {
	t.Parallel()
	broken := errors.New("synthetic failure")
	listed := map[string]bool{clinicA: true}
	for name, svc := range map[string]*practitionerrole.Service{
		"clinic":        newDirectoryService(newFakeRepository(), &fakeAuditor{}, &fakeClinics{err: broken, listed: listed}, &fakePeople{}),
		"practitioners": newDirectoryService(newFakeRepository(), &fakeAuditor{}, &fakeClinics{listed: listed}, &fakePeople{err: broken}),
	} {
		if _, err := svc.PublicByOrganization(t.Context(), clinicA); !errors.Is(err, broken) {
			t.Errorf("%s: PublicByOrganization() = %v, want the error", name, err)
		}
	}
}
