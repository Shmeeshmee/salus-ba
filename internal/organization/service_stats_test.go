package organization_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// salus_stats reads every clinic with its time zone, by id (docs/plans/analytics.md §3).
func TestStatsLocations_ReadsEveryClinic(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	if _, err := owner.Exec(t.Context(), "UPDATE organization SET time_zone = 'Europe/Zagreb' WHERE id = $1", clinicB); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	got, err := organization.NewStats(organization.NewPostgresStatsRepository(pool)).Locations(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != clinicA || got[0].TimeZone != "Europe/Sarajevo" || got[1].ID != clinicB || got[1].TimeZone != "Europe/Zagreb" {
		t.Errorf("Locations = %+v, want A in Sarajevo's zone and B in Zagreb's", got)
	}
}

// Locations carries the location and the contribution with the times they last changed (§5 "Which clinics count").
func TestStatsLocations_CarriesTheLocationAndTheContribution(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	if _, err := owner.Exec(t.Context(), `UPDATE organization SET region_id = 'BA-09', city_id = 'Q1270190',
			location_since = '2026-09-01T00:00:00Z', analytics_contributes = false, contributes_since = '2026-10-01T00:00:00Z'
		WHERE id = $1`, clinicA); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	got, err := organization.NewStats(organization.NewPostgresStatsRepository(pool)).Locations(t.Context())
	if err != nil || len(got) != 2 {
		t.Fatalf("Locations = %+v, %v", got, err)
	}
	a := got[0]
	if a.Region != "BA-09" || a.City != "Q1270190" || a.Contributes || !a.LocationSince.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) ||
		!a.ContributesSince.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("clinic A = %+v, want Centar since September, opted out since October", a)
	}
	if b := got[1]; b.Region != "" || !b.Contributes || b.ContributesSince.IsZero() || !b.LocationSince.IsZero() {
		t.Errorf("clinic B = %+v, want no location, contributing", b)
	}
}
