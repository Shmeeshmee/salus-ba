package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

func (r *fakeRepository) ByPractitioner(ctx context.Context, practitionerID string) (practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "ByPractitioner")
	if r.err != nil {
		return practitionerrole.PractitionerRole{}, r.err
	}
	for _, p := range r.roles {
		if p.PractitionerID == practitionerID && r.visible(p) {
			return p, nil
		}
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

func TestLookupAndOfPractitioner_FindTheRoleInTheClinicsScopeWithoutAudit(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		find func(s *practitionerrole.Service, ctx context.Context) (practitionerrole.PractitionerRole, error)
		call string
	}{
		"Lookup": {func(s *practitionerrole.Service, ctx context.Context) (practitionerrole.PractitionerRole, error) {
			return s.Lookup(ctx, clinicA, roleA2)
		}, "ByID"},
		"OfPractitioner": {func(s *practitionerrole.Service, ctx context.Context) (practitionerrole.PractitionerRole, error) {
			return s.OfPractitioner(ctx, clinicA, "a3000000-0000-4000-8000-0000000000a2")
		}, "ByPractitioner"},
	} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		got, err := tc.find(newService(repo, audit), t.Context())
		if err != nil || got != repo.roles[2] {
			t.Errorf("%s = %+v, %v; want role A2", name, got, err)
		}
		if !slices.Equal(repo.calls, []string{"SetOrganization", tc.call}) || repo.clinic != clinicA {
			t.Errorf("%s calls = %v in clinic %q, want the clinic's scope and the query in one transaction", name, repo.calls, repo.clinic)
		}
		if len(audit.events) != 0 {
			t.Errorf("%s events = %+v, want none: the lookup serves an access that audits itself", name, audit.events)
		}
	}
}

func TestLookupAndOfPractitioner_HideARoleOfAnotherClinicEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.leak = true
	s := newService(repo, &fakeAuditor{})
	if _, err := s.Lookup(t.Context(), clinicA, roleB); !errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("Lookup(clinic A, role B) = %v, want ErrNotFound", err)
	}
	if _, err := s.OfPractitioner(t.Context(), clinicA, "b3000000-0000-4000-8000-0000000000b1"); !errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("OfPractitioner(clinic A, practitioner of B) = %v, want ErrNotFound", err)
	}
	if _, err := s.OfPractitioner(t.Context(), clinicA, "a3000000-0000-4000-8000-0000000000ff"); !errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("OfPractitioner(an unknown practitioner) = %v, want ErrNotFound", err)
	}
}

func TestLookupAndOfPractitioner_FailOnARepositoryError(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.err = errors.New("connection lost")
	s := newService(repo, &fakeAuditor{})
	if _, err := s.Lookup(t.Context(), clinicA, roleA1); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("Lookup() = %v, want the repository's error", err)
	}
	if _, err := s.OfPractitioner(t.Context(), clinicA, "a3000000-0000-4000-8000-0000000000a1"); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("OfPractitioner() = %v, want the repository's error", err)
	}
}
