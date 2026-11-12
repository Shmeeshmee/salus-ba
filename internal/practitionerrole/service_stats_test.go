package practitionerrole_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// salus_stats reads a clinic's roles with their specialty (empty for none), only that clinic's
// (docs/plans/analytics.md §3).
func TestStatsSpecialties_ReadsTheClinicsRoles(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	seed(t, owner)
	if _, err := owner.Exec(t.Context(), "UPDATE practitioner_role SET specialty_code = '394537008' WHERE id = $1", roleA1); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	got, err := practitionerrole.NewStats(practitionerrole.NewPostgresStatsRepository(pool)).Specialties(t.Context(), clinicA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[roleA1] != "394537008" || got[roleA2] != "" {
		t.Errorf("Specialties = %v", got)
	}
	if _, err := practitionerrole.NewStats(practitionerrole.NewPostgresStatsRepository(pool)).Specialties(t.Context(), "x"); err == nil {
		t.Error("an id that is not a UUID gave no error")
	}
}
