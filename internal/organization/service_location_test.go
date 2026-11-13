package organization_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakePlaces holds a few of the seeded places.
type fakePlaces struct{ err error }

func (p fakePlaces) AllPlaces(context.Context) ([]terminology.Place, error) {
	return []terminology.Place{
		{ID: "BA-BIH", Kind: "entity", NameLatn: "Federacija Bosne i Hercegovine"},
		{ID: "BA-09", Kind: "region", Parent: "BA-BIH", NameLatn: "Kanton Sarajevo"},
		{ID: "BA-01", Kind: "region", Parent: "BA-BIH", NameLatn: "Unsko-sanski kanton"},
		{ID: "BA-SRP", Kind: "region", NameLatn: "Republika Srpska"},
		{ID: "Q1270190", Kind: "city", Parent: "BA-09", NameLatn: "Centar"},
		{ID: "Q14103175", Kind: "city", Parent: "BA-01", NameLatn: "Bihać"},
	}, p.err
}

func newLocationService(repo *fakeRepository, audit *fakeAuditor, at time.Time) *organization.Service {
	return organization.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: at}, fakePlaces{})
}

func located(region, city string) organization.Changes {
	return organization.Changes{Name: "Synthetic clinic A", Region: &region, City: &city}
}

// The admin sets the clinic's location (O1): a city under a leaf region, with the time it changed; the same
// location again keeps that time, so a clinic counts in a period only when it has not moved (§5).
func TestUpdate_SetsTheLocationWithTheTimeItChanged(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newLocationService(repo, audit, now).Update(t.Context(), admin, clinicA, located("BA-09", "Q1270190"), 2)
	if err != nil || got.Region != "BA-09" || got.City != "Q1270190" || !got.LocationSince.Equal(now) || got.CityName != "Centar" {
		t.Fatalf("Update() = %+v, %v; want Centar in BA-09 since now", got, err)
	}
	later := now.Add(48 * time.Hour)
	got, err = newLocationService(repo, audit, later).Update(t.Context(), admin, clinicA, located("BA-09", "Q1270190"), 3)
	if err != nil || !got.LocationSince.Equal(now) {
		t.Errorf("the same location again = %+v, %v; want it since the first time", got, err)
	}
	got, err = newLocationService(repo, audit, later).Update(t.Context(), admin, clinicA, located("BA-01", "Q14103175"), 4)
	if err != nil || got.Region != "BA-01" || !got.LocationSince.Equal(later) {
		t.Errorf("a move = %+v, %v; want Bihać since the move", got, err)
	}
	got, err = newLocationService(repo, audit, later).Update(t.Context(), admin, clinicA, located("", ""), 5)
	if err != nil || got.Region != "" || got.City != "" || !got.LocationSince.IsZero() {
		t.Errorf("clearing = %+v, %v; want no location", got, err)
	}
}

// A city outside its region, a region that is not a leaf region, or only one of the two, is refused before
// anything is accessed.
func TestUpdate_RefusesALocationThatIsNotACityInItsRegion(t *testing.T) {
	for name, ch := range map[string]organization.Changes{
		"a city of another region": located("BA-01", "Q1270190"),
		"an entity as the region":  located("BA-BIH", "BA-09"),
		"a region as the city":     located("BA-SRP", "BA-SRP"),
		"an unknown city":          located("BA-09", "Q1"),
		"no city":                  located("BA-09", ""),
		"no region":                located("", "Q1270190"),
		"the region alone":         {Name: "Synthetic clinic A", Region: new("BA-09")},
	} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newLocationService(repo, audit, now).Update(t.Context(), admin, clinicA, ch, 2)
		if !errors.Is(err, organization.ErrInvalidLocation) || len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("%s: Update() = %v after %v, want ErrInvalidLocation before any access", name, err, repo.calls)
		}
	}
	_, err := organization.NewService(newFakeRepository(), &fakeAuditor{}, fakeTransactor{}, clock.Fixed{T: now}, fakePlaces{err: errors.New("down")}).
		Update(t.Context(), admin, clinicA, located("BA-09", "Q1270190"), 2)
	if err == nil || errors.Is(err, organization.ErrInvalidLocation) {
		t.Errorf("Update() with the places down = %v, want the failure", err)
	}
}

// The clinic contributes to the platform figures until its admin opts it out (O6); a change of the choice sets
// the time it changed, the same choice keeps it.
func TestUpdate_OptsTheClinicOutWithTheTimeItChanged(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	o := repo.orgs[clinicA]
	o.AnalyticsContributes, o.ContributesSince = true, now.Add(-time.Hour)
	repo.orgs[clinicA] = o
	off, on := false, true
	got, err := newLocationService(repo, audit, now).Update(t.Context(), admin, clinicA, organization.Changes{Name: o.Name, AnalyticsContributes: &on}, 2)
	if err != nil || !got.AnalyticsContributes || !got.ContributesSince.Equal(now.Add(-time.Hour)) {
		t.Errorf("the same choice = %+v, %v; want the time kept", got, err)
	}
	got, err = newLocationService(repo, audit, now).Update(t.Context(), admin, clinicA, organization.Changes{Name: o.Name, AnalyticsContributes: &off}, 3)
	if err != nil || got.AnalyticsContributes || !got.ContributesSince.Equal(now) {
		t.Errorf("opting out = %+v, %v; want it out since now", got, err)
	}
}

// Read names the city of the location, in Latin script, for Organization.address.city.
func TestRead_NamesTheCity(t *testing.T) {
	repo := newFakeRepository()
	o := repo.orgs[clinicA]
	o.Region, o.City, o.LocationSince = "BA-09", "Q1270190", now
	repo.orgs[clinicA] = o
	got, err := newLocationService(repo, &fakeAuditor{}, now).Read(t.Context(), admin, clinicA)
	if err != nil || got.CityName != "Centar" {
		t.Errorf("Read() = %+v, %v; want the city named", got, err)
	}
}

// Contributes tells the platform view whether a clinic contributes (O6), unaudited, in the clinic's scope.
func TestContributes_ReadsTheClinicsChoice(t *testing.T) {
	repo := newFakeRepository()
	o := repo.orgs[clinicB]
	o.AnalyticsContributes = true
	repo.orgs[clinicB] = o
	svc := newLocationService(repo, &fakeAuditor{}, now)
	if got, err := svc.Contributes(t.Context(), clinicB); err != nil || !got {
		t.Errorf("Contributes(B) = %v, %v; want true", got, err)
	}
	if want := []string{"SetOrganization", "ByID"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls %v, want %v", repo.calls, want)
	}
	if got, err := svc.Contributes(t.Context(), clinicA); err != nil || got {
		t.Errorf("Contributes(A) = %v, %v; want false", got, err)
	}
	if _, err := svc.Contributes(t.Context(), ""); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Contributes('') = %v, want ErrNotFound", err)
	}
}
