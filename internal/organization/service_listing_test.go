package organization_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx, "SetDirectory")
	r.directory = true
	return nil
}

// Listed shows the listed clinics in the directory scope, by id; whether a clinic is listed is checked by
// the database and again by the service, so the fake shows every clinic when the scope is set.
func (r *fakeRepository) Listed(ctx context.Context) ([]organization.Organization, error) {
	r.seen(ctx, "Listed")
	if r.err != nil {
		return nil, r.err
	}
	if !r.directory {
		return nil, nil
	}
	return []organization.Organization{r.orgs[clinicA], r.orgs[clinicB]}, nil
}

func ptr[T any](v T) *T { return &v }

func TestUpdate_ListsTheClinicAndSetsItsTimeZone(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Update(t.Context(), admin, clinicA,
		organization.Changes{Name: "Synthetic clinic A", Listed: ptr(true), TimeZone: ptr("Europe/Vienna")}, 2)
	want := organization.Organization{ID: clinicA, Name: "Synthetic clinic A", TimeZone: "Europe/Vienna", VersionID: 3, Active: true,
		Listed: true, LastUpdated: now}
	if err != nil || got != want {
		t.Fatalf("Update() = %+v, %v;\nwant %+v", got, err, want)
	}
	wantEvent := updateEvent(admin, auditevent.OutcomeSuccess)
	wantEvent.EntityReference = "Organization/" + clinicA
	if !slices.Equal(audit.events, []auditevent.AuditEvent{wantEvent}) {
		t.Errorf("audit events %+v, want %+v", audit.events, wantEvent)
	}
}

func TestUpdate_KeepsTheListingAndTimeZoneThatAreNotGiven(t *testing.T) {
	repo := newFakeRepository()
	a := repo.orgs[clinicA]
	a.Listed, a.TimeZone = true, "Europe/Vienna"
	repo.orgs[clinicA] = a
	got, err := newService(repo, &fakeAuditor{}).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Renamed"}, 2)
	if err != nil || !got.Listed || got.TimeZone != "Europe/Vienna" || got.Name != "Renamed" {
		t.Errorf("Update(name only) = %+v, %v; want the listing and zone kept", got, err)
	}
	got, err = newService(repo, &fakeAuditor{}).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Renamed", Listed: ptr(false)}, 3)
	if err != nil || got.Listed || got.TimeZone != "Europe/Vienna" {
		t.Errorf("Update(unlisted) = %+v, %v; want unlisted with the zone kept", got, err)
	}
}

func TestUpdate_AcceptsAnIANATimeZone(t *testing.T) {
	for _, zone := range []string{"Europe/Sarajevo", "America/New_York", "Asia/Kolkata", "UTC"} {
		if _, err := newService(newFakeRepository(), &fakeAuditor{}).Update(t.Context(), admin, clinicA,
			organization.Changes{Name: "Renamed", TimeZone: ptr(zone)}, 2); err != nil {
			t.Errorf("Update(time zone %q) = %v, want nil", zone, err)
		}
	}
}

func TestUpdate_RejectsAnUnknownTimeZoneBeforeAnyAccess(t *testing.T) {
	for _, zone := range []string{"", "Local", "Europe/Atlantis", "europe/sarajevo", " Europe/Sarajevo", "+01:00", "../etc/passwd"} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Renamed", TimeZone: ptr(zone)}, 2)
		if !errors.Is(err, organization.ErrInvalidTimeZone) || len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("Update(time zone %q) = %v, calls %v, events %d; want ErrInvalidTimeZone before any access",
				zone, err, repo.calls, len(audit.events))
		}
	}
}

func TestListListed_GivesTheListedClinicsInTheDirectoryScopeWithoutAudit(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).ListListed(t.Context())
	// Clinic A is not listed: the service does not rest on row-level security alone.
	if err != nil || !slices.Equal(got, []organization.Organization{repo.orgs[clinicB]}) {
		t.Errorf("ListListed() = %+v, %v; want clinic B only", got, err)
	}
	if !slices.Equal(repo.calls, []string{"SetDirectory", "Listed"}) {
		t.Errorf("calls %v, want the directory scope and the query in one transaction", repo.calls)
	}
	if len(audit.events) != 0 {
		t.Errorf("audit events %+v, want none: a public read is not audited", audit.events)
	}
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, audit).ListListed(t.Context()); !errors.Is(err, repo.err) {
		t.Errorf("ListListed() with a failing query = %v, want the repository error", err)
	}
}
