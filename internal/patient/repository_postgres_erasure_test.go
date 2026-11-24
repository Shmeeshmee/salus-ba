package patient_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The erasure restricts the record clinic A keeps of a due account's profile, once (docs/clinic-cms/WAVE5.md S6).
func TestRestrictForErasure_RestrictsTheDueProfileAtTheClinicOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A')", clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE account SET closed_at = now() - interval '22 days', erase_after = now() - interval '1 day'
		WHERE id = $1`, accountA); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	restrict := func() error {
		return postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
			if err := postgres.SetLocal(ctx, "app.erasure", "on"); err != nil {
				return err
			}
			if err := repo.SetClinic(ctx, clinicA); err != nil {
				return err
			}
			return repo.RestrictForErasure(ctx, patientA, clinicA, now)
		})
	}
	for range 2 {
		if err := restrict(); err != nil {
			t.Fatalf("RestrictForErasure() error = %v", err)
		}
	}
	var n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM patient_restriction WHERE patient_id = $1 AND organization_id = $2
		AND restricted_by_account_id IS NULL AND data_request_id IS NULL`, patientA, clinicA).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d restrictions, %v; want the job's one", n, err)
	}
}
