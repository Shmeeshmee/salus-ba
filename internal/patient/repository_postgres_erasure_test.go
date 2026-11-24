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

// The erasure's restriction replaces one the clinic's staff set before (D3 with D4): afterwards the record is held by
// the job's restriction, which the clinic cannot lift.
func TestRestrictForErasure_ReplacesAStaffRestriction(t *testing.T) {
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
	if _, err := owner.Exec(ctx, `INSERT INTO patient_restriction (restricted_at, patient_id, organization_id, restricted_by_account_id)
		VALUES (now() - interval '3 days', $1, $2, $3)`, patientA, clinicA, accountA); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	tx := postgres.NewTransactor(app)
	err := tx.WithTx(ctx, func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.erasure", "on"); err != nil {
			return err
		}
		if err := repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return repo.RestrictForErasure(ctx, patientA, clinicA, now)
	})
	if err != nil {
		t.Fatalf("RestrictForErasure() error = %v", err)
	}
	err = tx.WithTx(ctx, func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return repo.Unrestrict(ctx, patientA, clinicA)
	})
	if err != nil {
		t.Fatalf("the clinic's lift: %v", err)
	}
	var byStaff, n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FILTER (WHERE restricted_by_account_id IS NOT NULL), count(*)
		FROM patient_restriction WHERE patient_id = $1 AND organization_id = $2`, patientA, clinicA).Scan(&byStaff, &n); err != nil || byStaff != 0 || n != 1 {
		t.Errorf("%d restrictions, %d by staff, %v; want the job's one, kept after the clinic's lift", n, byStaff, err)
	}
}

// Whether a record is restricted at a clinic reads the same from any scope (D4 with the worker and the chat's guests):
// another clinic's, a patient's, and a job's with none set, as patient_record_restricted (SECURITY DEFINER) answers it.
func TestRestricted_ReadsTheSameFromAnyScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A'), ($2, 'Clinic B')", clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO patient_restriction (restricted_at, patient_id, organization_id, restricted_by_account_id)
		VALUES (now(), $1, $2, $3)`, patientA, clinicA, accountA); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	for name, scope := range map[string]func(ctx context.Context) error{
		"another clinic": func(ctx context.Context) error { return repo.SetClinic(ctx, clinicB) },
		"the patient":    func(ctx context.Context) error { return postgres.SetLocal(ctx, "app.patient_id", patientA) },
		"a job":          func(context.Context) error { return nil },
	} {
		var atA, atB bool
		err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
			if err := scope(ctx); err != nil {
				return err
			}
			var err error
			if atA, err = repo.Restricted(ctx, patientA, clinicA); err != nil {
				return err
			}
			atB, err = repo.Restricted(ctx, patientA, clinicB)
			return err
		})
		if err != nil || !atA || atB {
			t.Errorf("%s: restricted at A %v, at B %v, %v; want A only", name, atA, atB, err)
		}
	}
}

// The clinic's restricted records are listed in its own scope only (the data export's manifest, D4).
func TestRestrictedAt_ListsTheClinicsOwnRestrictions(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A'), ($2, 'Clinic B')", clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO patient_restriction (restricted_at, patient_id, organization_id) VALUES (now(), $1, $2)`, patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	for clinic, want := range map[string]int{clinicA: 1, clinicB: 0} {
		var ids []string
		if err := inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			var err error
			ids, err = repo.RestrictedAt(ctx, clinic)
			return err
		}); err != nil || len(ids) != want || (want == 1 && ids[0] != patientA) {
			t.Errorf("clinic %s: %v, %v; want %d", clinic, ids, err, want)
		}
	}
}
