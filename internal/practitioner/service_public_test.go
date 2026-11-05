package practitioner_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitioner"
)

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx, "SetDirectory")
	r.directory = true
	return nil
}

// PublicByIDs shows the practitioners of ids in the directory scope, inactive ones too when leak is set (a
// database without the policy); whether the clinic is listed is the database's part. Like the query, it
// leaves out the clinic.
func (r *fakeRepository) PublicByIDs(ctx context.Context, ids []string) ([]practitioner.Practitioner, error) {
	r.seen(ctx, "PublicByIDs")
	if r.err != nil {
		return nil, r.err
	}
	var out []practitioner.Practitioner
	for _, id := range ids {
		p, ok := r.byID[id]
		if ok && (r.leak || (r.directory && p.Active)) {
			p.OrganizationID = ""
			out = append(out, p)
		}
	}
	return out, nil
}

func TestPublicNames_GivesTheNamesInTheDirectoryScopeWithoutAudit(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).PublicNames(t.Context(), []string{practitionerA, practitionerB})
	want := []practitioner.Practitioner{
		{ID: practitionerA, GivenName: "Ada", FamilyName: "Example", VersionID: 1, Active: true, LastUpdated: now},
		{ID: practitionerB, GivenName: "Bo", FamilyName: "Example", VersionID: 1, Active: true, LastUpdated: now},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicNames() = %+v, %v;\nwant %+v", got, err, want)
	}
	if !slices.Equal(repo.calls, []string{"SetDirectory", "PublicByIDs"}) {
		t.Errorf("calls = %v, want the directory scope and the query in one transaction", repo.calls)
	}
	if len(audit.events) != 0 {
		t.Errorf("events = %+v, want none: a public read is not audited", audit.events)
	}
}

func TestPublicNames_KeepsOnlyActivePractitionersEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.leak = true
	b := repo.byID[practitionerB]
	b.Active = false
	repo.byID[practitionerB] = b
	got, err := newService(repo, &fakeAuditor{}).PublicNames(t.Context(), []string{practitionerA, practitionerB})
	if err != nil || len(got) != 1 || got[0].ID != practitionerA {
		t.Errorf("PublicNames() without the policy = %+v, %v; want only the active practitioner A", got, err)
	}
}

func TestPublicNames_AsksNothingForNoIDsAndFailsOnARepositoryError(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	if got, err := newService(repo, &fakeAuditor{}).PublicNames(t.Context(), nil); err != nil || len(got) != 0 || len(repo.calls) != 0 {
		t.Errorf("PublicNames(none) = %+v, %v after %v; want none without a query", got, err, repo.calls)
	}
	broken := errors.New("synthetic failure")
	repo.err = broken
	if _, err := newService(repo, &fakeAuditor{}).PublicNames(t.Context(), []string{practitionerA}); !errors.Is(err, broken) {
		t.Errorf("PublicNames() = %v, want the repository error", err)
	}
}
