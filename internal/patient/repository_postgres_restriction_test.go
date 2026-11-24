package patient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A restriction (docs/clinic-cms/WAVE5.md S7) is set once by the clinic's staff, seen only in the clinic's scope,
// hides the patient from the clinic's search, and is lifted by the clinic unless the erasure set it.
func TestRestriction_RestrictsSearchesAndLiftsInTheClinicsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A'), ($2, 'Clinic B')", clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO account (id, email, password_hash, role, organization_id)
		VALUES ($1, 'admin.a@example.test', 'h', 'practitioner', $2)`, staffA, clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	restricted := func(clinic string) bool {
		t.Helper()
		var got bool
		if err := inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			var err error
			got, err = repo.Restricted(ctx, patientA, clinic)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}
	found := func() int {
		t.Helper()
		var n int
		if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			ps, err := repo.SearchClinic(ctx, clinicA, "", patient.ClinicSearch{Name: "example"}, 50)
			n = len(ps)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if restricted(clinicA) || found() != 1 {
		t.Fatalf("before: restricted %v, found %d; want unrestricted and found", restricted(clinicA), found())
	}
	for range 2 {
		if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			return repo.Restrict(ctx, patientA, clinicA, staffA, "", now)
		}); err != nil {
			t.Fatalf("Restrict: %v", err)
		}
	}
	if !restricted(clinicA) || restricted(clinicB) || found() != 0 {
		t.Errorf("restricted at A %v, at B %v, found %d; want A only and hidden from A's search", restricted(clinicA), restricted(clinicB), found())
	}
	if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error { return repo.Unrestrict(ctx, patientA, clinicA) }); err != nil {
		t.Fatal(err)
	}
	if restricted(clinicA) || found() != 1 {
		t.Errorf("after lifting: restricted %v, found %d", restricted(clinicA), found())
	}
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		return repo.Restrict(ctx, patientA, clinicA, staffA, "f9000000-0000-4000-8000-0000000000f9", now)
	})
	if !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("Restrict for an unknown request = %v, want ErrNotFound", err)
	}
	if _, err := owner.Exec(t.Context(), "INSERT INTO patient_restriction (restricted_at, patient_id, organization_id) VALUES (now(), $1, $2)", patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error { return repo.Unrestrict(ctx, patientA, clinicA) }); err != nil {
		t.Fatal(err)
	}
	if !restricted(clinicA) {
		t.Error("the erasure's restriction was lifted by the clinic")
	}
}
