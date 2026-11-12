package organization_test

import (
	"testing"

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
	want := []organization.Location{{ID: clinicA, TimeZone: "Europe/Sarajevo"}, {ID: clinicB, TimeZone: "Europe/Zagreb"}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Locations = %+v, want %+v", got, want)
	}
}
