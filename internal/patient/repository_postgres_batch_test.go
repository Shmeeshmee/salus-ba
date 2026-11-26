package patient_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A page of a clinic's visits reads its patients' profiles and whether each record is restricted in one statement
// each (docs/clinic-cms/load/REPORT.md), as the scope shows them: patient B, without a consent at clinic A, is not
// read; an id that is not a UUID is no row.
func TestByIDsAndRestrictedAll_ReadTheClinicsPatientsInOneStatement(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	ctx := t.Context()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organization (id, name) VALUES ($1, 'Clinic A')`, []any{clinicA}},
		{`INSERT INTO patient (id, account_id, given_name, family_name, birth_date) VALUES ($1, $2, 'Bo', 'Example', '1991-01-01')`, []any{patientB, accountB}},
		{`INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`, []any{patientA, clinicA}},
		{`INSERT INTO patient_restriction (restricted_at, patient_id, organization_id, restricted_by_account_id) VALUES (now(), $1, $2, $3)`, []any{patientA, clinicA, accountA}},
	} {
		if _, err := owner.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	repo := patient.NewPostgresRepository(app)
	var got []patient.Patient
	var restricted map[string]bool
	err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		var err error
		if got, err = repo.ByIDs(ctx, []string{patientA, patientB, "not-a-uuid"}); err != nil {
			return err
		}
		restricted, err = repo.RestrictedAll(ctx, []string{patientA, patientB, "not-a-uuid"}, clinicA)
		return err
	})
	if err != nil || len(got) != 1 || got[0].ID != patientA || got[0].GivenName != "Ada" {
		t.Fatalf("ByIDs = %+v, %v; want patient A only", got, err)
	}
	if !restricted[patientA] || restricted[patientB] || len(restricted) != 2 {
		t.Errorf("RestrictedAll = %v; want A restricted and B not", restricted)
	}
}
