package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// analytics_left_out (00076, docs/clinic-cms/WAVE5.md D4) tells the aggregate builds a patient who objected to the
// analytics (S9) and one whose record is restricted at the clinic (S7), without salus_stats reading either table;
// salus_app may not call it.
func TestAnalyticsLeftOut_AnswersTheStatsRoleOnly(t *testing.T) {
	t.Parallel()
	owner, stats := pgtest.NewStatsDatabase(t)
	var patient, clinic, other string
	if err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic'), ('Other clinic') RETURNING id)
		SELECT min(id::text), max(id::text) FROM o`).Scan(&clinic, &other); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(t.Context(), `INSERT INTO patient (given_name, family_name, birth_date, registered_by_organization_id)
		VALUES ('Ada', 'Example', '1990-01-01', $1) RETURNING id::text`, clinic).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	leftOut := func(org string) bool {
		t.Helper()
		var out bool
		if err := stats.QueryRow(t.Context(), `SELECT analytics_left_out($1, $2)`, patient, org).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	if leftOut(clinic) {
		t.Error("left out before any objection or restriction")
	}
	exec(`INSERT INTO patient_restriction (restricted_at, patient_id, organization_id) VALUES (now(), $1, $2)`, patient, clinic)
	if !leftOut(clinic) || leftOut(other) {
		t.Errorf("restricted at the clinic: left out there %v, at another %v; want there only", leftOut(clinic), leftOut(other))
	}
	exec(`UPDATE patient SET analytics_objected_at = now() WHERE id = $1`, patient)
	if !leftOut(other) {
		t.Error("an objection leaves the patient out at every clinic")
	}
	for _, q := range []string{`SELECT analytics_objected_at FROM patient`, `SELECT patient_id FROM patient_restriction`} {
		if _, err := stats.Exec(t.Context(), q); sqlState(err) != "42501" {
			t.Errorf("%s as salus_stats = %v, want permission denied", q, err)
		}
	}
	_, app := pgtest.NewAppDatabase(t)
	if _, err := app.Exec(t.Context(), `SELECT analytics_left_out(gen_random_uuid(), gen_random_uuid())`); sqlState(err) != "42501" {
		t.Errorf("analytics_left_out as salus_app = %v, want permission denied", err)
	}
}
