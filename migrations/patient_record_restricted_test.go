package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// patient_record_restricted (docs/clinic-cms/WAVE5.md S7) answers salus_app whether a record is restricted at a clinic
// with no scope set, where patient_restriction itself shows the app role nothing.
func TestPatientRecordRestricted_AnswersTheAppRoleFromAnyScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var patient, clinic, other string
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic'), ('Other clinic') RETURNING id)
		SELECT min(id::text), max(id::text) FROM o`).Scan(&clinic, &other); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO patient (given_name, family_name, birth_date, registered_by_organization_id)
		VALUES ('Ada', 'Example', '1990-01-01', $1) RETURNING id::text`, clinic).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO patient_restriction (restricted_at, patient_id, organization_id) VALUES (now(), $1, $2)`, patient, clinic); err != nil {
		t.Fatal(err)
	}
	var seen int
	var atClinic, atOther bool
	if err := app.QueryRow(ctx, `SELECT (SELECT count(*) FROM patient_restriction), patient_record_restricted($1, $2), patient_record_restricted($1, $3)`,
		patient, clinic, other).Scan(&seen, &atClinic, &atOther); err != nil {
		t.Fatal(err)
	}
	if seen != 0 || !atClinic || atOther {
		t.Errorf("rows seen %d, restricted at the clinic %v, at the other %v; want 0, true, false", seen, atClinic, atOther)
	}
}
