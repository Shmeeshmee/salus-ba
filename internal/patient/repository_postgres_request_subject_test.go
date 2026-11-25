package patient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The patient a data request names is read by the clinic the request targets once its scope is widened to the
// patients its requests name (00079), whatever the consent: an app patient who revoked theirs and never had a visit
// there included. Another clinic, the clinic without the widening, and the widening narrowed again read nothing.
func TestRequestSubject_TheClinicsScopeWidenedToItsRequests(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	for _, q := range []string{
		"INSERT INTO organization (id, name) VALUES ('" + clinicA + "', 'Clinic A'), ('" + clinicB + "', 'Clinic B')",
		"INSERT INTO consent (patient_id, organization_id, status, date_time, revoked_at) VALUES ('" + patientA + "', '" + clinicA + "', 'inactive', now() - interval '2 days', now() - interval '1 day')",
		`INSERT INTO data_request (received_at, due_at, account_id, patient_id, organization_id, filed_by_account_id, kind)
		 VALUES (now(), now() + interval '1 month', '` + accountA + "', '" + patientA + "', '" + clinicA + "', '" + accountA + "', 'restriction')",
	} {
		if _, err := owner.Exec(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	repo := patient.NewPostgresRepository(app)
	read := func(clinic string, widen bool, narrow bool) error {
		t.Helper()
		return inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			if widen {
				if err := repo.SetDataRequests(ctx, true); err != nil {
					return err
				}
			}
			if narrow {
				if err := repo.SetDataRequests(ctx, false); err != nil {
					return err
				}
			}
			p, err := repo.ByID(ctx, patientA)
			if err == nil && (p.GivenName != "Ada" || p.FamilyName != "Example") {
				t.Errorf("ByID = %+v, want Ada Example", p)
			}
			return err
		})
	}
	if err := read(clinicA, true, false); err != nil {
		t.Errorf("clinic A, widened: %v, want the profile", err)
	}
	for name, err := range map[string]error{
		"widened in a patient's scope":  widenedAsPatient(t, app, repo),
		"clinic A without the widening": read(clinicA, false, false),
		"clinic A narrowed again":       read(clinicA, true, true),
		"clinic B, widened":             read(clinicB, true, false),
	} {
		if !errors.Is(err, patient.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
}

// The restriction is read with its time and whether the erasure made it; none is zero; another clinic sees none.
func TestRestriction_ReadsItsTimeAndMaker(t *testing.T) {
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
	repo := patient.NewPostgresRepository(app)
	get := func(clinic string) patient.Restriction {
		t.Helper()
		var got patient.Restriction
		if err := inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			var err error
			got, err = repo.Restriction(ctx, patientA, clinic)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := get(clinicA); got.Restricted() {
		t.Errorf("before: %+v, want none", got)
	}
	if _, err := owner.Exec(t.Context(), "INSERT INTO patient_restriction (restricted_at, patient_id, organization_id, restricted_by_account_id) VALUES ($1, $2, $3, $4)", now, patientA, clinicA, staffA); err != nil {
		t.Fatal(err)
	}
	if got := get(clinicA); !got.At.Equal(now) || got.ByErasure {
		t.Errorf("the clinic's: %+v, want at %v by the clinic", got, now)
	}
	if got := get(clinicB); got.Restricted() {
		t.Errorf("clinic B sees %+v, want none", got)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE patient_restriction SET restricted_by_account_id = NULL"); err != nil {
		t.Fatal(err)
	}
	if got := get(clinicA); !got.ByErasure {
		t.Errorf("the erasure's: %+v, want by the erasure", got)
	}
}

// widenedAsPatient reads patient A in clinic A's scope widened to its requests with another patient's account set too:
// the widening holds in the clinic's scope only.
func widenedAsPatient(t *testing.T, app *pgxpool.Pool, repo patient.Repository) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		if err := repo.SetAccount(ctx, accountB); err != nil {
			return err
		}
		if err := repo.SetDataRequests(ctx, true); err != nil {
			return err
		}
		_, err := repo.ByID(ctx, patientA)
		return err
	})
}
