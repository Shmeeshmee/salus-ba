package migrations_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A clinic whose membership is suspended or cancelled is not in the public directory (package I of
// docs/plans/clinic-cms.md, migration 00064): neither the clinic nor its practitioners and roles, and so nothing that
// hangs off them. Its own scope still reads it: care goes on. A clinic without a membership, in a trial, active or past
// due is listed as before.
func TestDirectoryMembership_ASuspendedOrCancelledClinicIsNotInTheDirectory(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedDirectory(t, owner)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE organization SET listed = true WHERE id = $1`, clinicA)
	exec(`INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)`)
	directory := map[string]string{"app.directory": "public"}
	clinic := map[string]string{"app.organization_id": clinicA}
	tables := map[string]string{"organization": clinicA, "practitioner": seeded[clinicA].practitioner, "practitioner_role": seeded[clinicA].role}
	listed := func(want bool, when string) {
		t.Helper()
		for table, id := range tables {
			got := idsIn(t, app, directory, "SELECT id::text FROM "+table)
			if slices.Contains(got, id) != want {
				t.Errorf("%s: %s in the directory = %v, want listed %v", when, table, got, want)
			}
		}
		if got := idsIn(t, app, clinic, "SELECT id::text FROM organization"); !slices.Equal(got, []string{clinicA}) {
			t.Errorf("%s: the clinic's own scope reads %v, want the clinic", when, got)
		}
	}
	listed(true, "no membership")
	exec(`INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES ($1, 'basic', 'trial', now() + interval '30 days')`, clinicA)
	listed(true, "a trial")
	if got := idsIn(t, app, directory, "SELECT organization_id::text FROM clinic_subscription"); len(got) != 0 {
		t.Errorf("the directory reads the membership %v of a clinic in a trial, want none", got)
	}
	for _, status := range []string{"active", "past_due"} {
		exec(`UPDATE clinic_subscription SET status = $2 WHERE organization_id = $1`, clinicA, status)
		listed(true, status)
	}
	for _, status := range []string{"suspended", "cancelled"} {
		exec(`UPDATE clinic_subscription SET status = $2 WHERE organization_id = $1`, clinicA, status)
		listed(false, status)
	}
	exec(`UPDATE clinic_subscription SET status = 'active' WHERE organization_id = $1`, clinicA) // paid
	listed(true, "active again")
}
