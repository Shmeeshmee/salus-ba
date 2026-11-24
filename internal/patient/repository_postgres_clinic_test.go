package patient_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const clinicB = "b0000000-0000-4000-8000-00000000000b"

// inClinic runs fn in a transaction on app scoped to the clinic.
func inClinic(t *testing.T, app *pgxpool.Pool, repo patient.Repository, clinic string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinic); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func TestRegisterAndSearchClinic_GiveTheClinicItsOwnAndItsConsentedPatientsOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner) // A has a profile, without a consent with any clinic yet
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A'), ($2, 'Clinic B')", clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	var cy patient.Patient
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		cy, err = repo.Register(ctx, patient.Patient{GivenName: "Cy", FamilyName: "Examplić", BirthDate: date(1980, 3, 4), Phone: "+38761000002"}, clinicA, now)
		return err
	})
	if err != nil || cy.ID == "" || cy.AccountID != "" || cy.RegisteredBy != clinicA || cy.VersionID != 1 || !cy.LastUpdated.Equal(now) {
		t.Fatalf("Register = %+v, %v; want clinic A's profile without an account", cy, err)
	}
	search := func(clinic, role string, q patient.ClinicSearch) []string {
		t.Helper()
		var ids []string
		err := inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			found, err := repo.SearchClinic(ctx, clinic, role, q, 50)
			for _, p := range found {
				ids = append(ids, p.ID)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return ids
	}
	if got := search(clinicA, "", patient.ClinicSearch{Name: "exampl"}); len(got) != 1 || got[0] != cy.ID {
		t.Errorf("clinic A before A's consent: %v, want Cy only", got)
	}
	if got := search(clinicB, "", patient.ClinicSearch{Name: "exampl"}); len(got) != 0 {
		t.Errorf("clinic B: %v, want nobody (Cy is clinic A's)", got)
	}
	if _, err := owner.Exec(t.Context(), "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	if got := search(clinicA, "", patient.ClinicSearch{Name: "Exampl"}); len(got) != 2 || got[0] != patientA {
		t.Errorf("clinic A after A's consent: %v, want Ada Example then Cy Examplić", got)
	}
	for name, q := range map[string]patient.ClinicSearch{
		"family then given":   {Name: "examplić cy"},
		"given then family":   {Name: "Cy Ex"},
		"with the birth date": {Name: "ex", BirthDate: date(1980, 3, 4)},
	} {
		if got := search(clinicA, "", q); len(got) != 1 || got[0] != cy.ID {
			t.Errorf("%s: %v, want Cy", name, got)
		}
	}
	if got := search(clinicA, "", patient.ClinicSearch{Name: "%"}); len(got) != 0 {
		t.Errorf("a wildcard: %v, want nobody (taken literally)", got)
	}
	if got := search(clinicA, roleA, patient.ClinicSearch{Name: "ex"}); len(got) != 0 {
		t.Errorf("a practitioner without appointments: %v, want nobody", got)
	}
}

func TestCorrect_ChangesOnlyTheCurrentVersionOfAProfileTheClinicRegistered(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Clinic A'), ($2, 'Clinic B')", clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	repo := patient.NewPostgresRepository(app)
	var cy patient.Patient
	if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		cy, err = repo.Register(ctx, patient.Patient{GivenName: "Cy", FamilyName: "Example", BirthDate: date(1980, 3, 4)}, clinicA, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	correct := func(clinic string, p patient.Patient) (patient.Patient, error) {
		var out patient.Patient
		err := inClinic(t, app, repo, clinic, func(ctx context.Context) error {
			var err error
			out, err = repo.Correct(ctx, p, clinic, now.Add(time.Hour))
			return err
		})
		return out, err
	}
	in := patient.Patient{ID: cy.ID, VersionID: 1, GivenName: "Cyra", FamilyName: "Example", BirthDate: date(1980, 4, 3), Phone: "+38761000003"}
	if _, err := correct(clinicB, in); !errors.Is(err, patient.ErrConflict) {
		t.Errorf("clinic B: %v, want ErrConflict (no profile of its own)", err)
	}
	if _, err := correct(clinicA, patient.Patient{ID: patientA, VersionID: 1, GivenName: "X", FamilyName: "Y", BirthDate: date(1980, 1, 1)}); !errors.Is(err, patient.ErrConflict) {
		t.Errorf("an app patient's profile: %v, want ErrConflict", err)
	}
	got, err := correct(clinicA, in)
	if err != nil || got.GivenName != "Cyra" || got.VersionID != 2 || !got.BirthDate.Equal(date(1980, 4, 3)) || got.Phone != "+38761000003" || got.RegisteredBy != clinicA {
		t.Fatalf("Correct = %+v, %v; want the corrected profile at version 2", got, err)
	}
	if _, err := correct(clinicA, in); !errors.Is(err, patient.ErrConflict) {
		t.Errorf("a stale version: %v, want ErrConflict", err)
	}
}
