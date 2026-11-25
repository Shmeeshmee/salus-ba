package careplan_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/careplan"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

const (
	clinicB  = "0a000000-0000-4000-8000-0000000000a9"
	doctorA  = "0d000000-0000-4000-8000-0000000000e1"
	patAcct  = "0d000000-0000-4000-8000-0000000000e2"
	patAcctB = "0d000000-0000-4000-8000-0000000000e3"
	patientB = "0b000000-0000-4000-8000-0000000000b9" // no consent with clinic A
)

// seedDB is clinic A and B, a practitioner of A, patient A with A's active care consent, patient B without, and A's
// two services.
func seedDB(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	for _, s := range []string{
		`INSERT INTO organization (id, name) VALUES ('` + clinicA + `', 'Synthetic clinic A'), ('` + clinicB + `', 'Synthetic clinic B')`,
		`INSERT INTO account (id, email, password_hash, role, organization_id) VALUES ('` + doctorA + `', 'doctor.a@example.test', 'h', 'practitioner', '` + clinicA + `')`,
		`INSERT INTO account (id, email, password_hash, role) VALUES ('` + patAcct + `', 'pat.a@example.test', 'h', 'patient'), ('` + patAcctB + `', 'pat.b@example.test', 'h', 'patient')`,
		`INSERT INTO patient (id, account_id, given_name, family_name, birth_date) VALUES
			('` + patientA + `', '` + patAcct + `', 'Ada', 'Example', '1990-01-01'), ('` + patientB + `', '` + patAcctB + `', 'Cy', 'Example', '1991-01-01')`,
		`INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ('` + patientA + `', '` + clinicA + `', 'active', now())`,
		`INSERT INTO charge_item_definition (id, organization_id, name, price_minor) VALUES
			('` + serviceIn + `', '` + clinicA + `', 'Placeholder examination', 5000), ('` + serviceVd + `', '` + clinicA + `', 'Placeholder follow-up', 3000)`,
	} {
		if _, err := owner.Exec(t.Context(), s); err != nil {
			t.Fatal(err)
		}
	}
}

// inScope runs fn in a transaction in the clinic's scope, or the patient's.
func inScope(t *testing.T, tr *postgres.Transactor, repo careplan.Repository, clinic, patient string, fn func(ctx context.Context) error) error {
	t.Helper()
	return tr.WithTx(t.Context(), func(ctx context.Context) error {
		if clinic != "" {
			if err := repo.SetOrganization(ctx, clinic); err != nil {
				return err
			}
		}
		if patient != "" {
			if err := repo.SetPatient(ctx, patient); err != nil {
				return err
			}
		}
		return fn(ctx)
	})
}

func TestRepository_StoresAPlanItsStepsAndItsChanges(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDB(t, owner)
	repo, tr := careplan.NewPostgresRepository(app), postgres.NewTransactor(app)
	steps := []careplan.Step{{ServiceID: serviceIn, VisitMode: "in-person"}, {ServiceID: serviceVd, VisitMode: "video", DayOffset: 14}}
	var p careplan.Plan
	if err := inScope(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		var err error
		p, err = repo.Insert(ctx, careplan.Plan{OrganizationID: clinicA, PatientID: patientA, Title: "Synthetic plan", Steps: steps}, doctorA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if p.ID == "" || p.Status != careplan.StatusDraft || p.VersionID != 1 || len(p.Steps) != 2 || p.Steps[1].Position != 2 || p.Steps[1].DayOffset != 14 {
		t.Fatalf("Insert = %+v", p)
	}
	// The steps swap places, the first is dropped and a new one added; then the plan is booked.
	p.Steps = []careplan.Step{p.Steps[1], {ServiceID: serviceIn, VisitMode: "in-person", DayOffset: 30}}
	p.Title, p.VersionID = "Changed", 2
	day, at := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC)
	if err := inScope(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		if ok, err := repo.Update(ctx, p, doctorA); !ok || err != nil {
			return errors.Join(err, errors.New("not updated"))
		}
		if ok, err := repo.Update(ctx, p, doctorA); ok || err != nil { // version 1 is gone
			return errors.Join(err, errors.New("updated at an old version"))
		}
		var err error
		p, err = repo.ByID(ctx, p.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if p.Title != "Changed" || p.VersionID != 2 || len(p.Steps) != 2 || p.Steps[0].ServiceID != serviceVd || p.Steps[1].DayOffset != 30 {
		t.Fatalf("after Update: %+v", p)
	}
	if got := mine(t, tr, repo); len(got) != 0 {
		t.Errorf("the patient sees %+v before booking, want nothing", got)
	}
	p.Status, p.StartsOn, p.BookedAt, p.VersionID = careplan.StatusActive, &day, &at, 3
	if err := inScope(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		_, err := repo.Update(ctx, p, doctorA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	got := mine(t, tr, repo)
	if len(got) != 1 || got[0].Status != careplan.StatusActive || !got[0].StartsOn.Equal(day) || len(got[0].Steps) != 2 {
		t.Errorf("the patient sees %+v, want the booked plan", got)
	}
}

func mine(t *testing.T, tr *postgres.Transactor, repo careplan.Repository) []careplan.Plan {
	t.Helper()
	var out []careplan.Plan
	if err := inScope(t, tr, repo, "", patientA, func(ctx context.Context) error {
		var err error
		out, err = repo.ForPatient(ctx, patientA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRepository_RefusesWhatRowLevelSecurityRefuses(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDB(t, owner)
	repo, tr := careplan.NewPostgresRepository(app), postgres.NewTransactor(app)
	steps := []careplan.Step{{ServiceID: serviceIn, VisitMode: "in-person"}}
	for name, tc := range map[string]struct {
		clinic string
		p      careplan.Plan
	}{
		"a patient without the clinic's consent": {clinicA, careplan.Plan{OrganizationID: clinicA, PatientID: patientB, Title: "Synthetic plan", Steps: steps}},
		"another clinic's plan":                  {clinicB, careplan.Plan{OrganizationID: clinicA, PatientID: patientA, Title: "Synthetic plan", Steps: steps}},
		"a quote that does not exist":            {clinicA, careplan.Plan{OrganizationID: clinicA, PatientID: patientA, Title: "Synthetic plan", QuoteID: patientB, Steps: steps}},
	} {
		err := inScope(t, tr, repo, tc.clinic, "", func(ctx context.Context) error {
			_, err := repo.Insert(ctx, tc.p, doctorA)
			return err
		})
		if !errors.Is(err, careplan.ErrNotFound) && !errors.Is(err, careplan.ErrInvalid) {
			t.Errorf("%s: %v, want a refusal", name, err)
		}
	}
	err := inScope(t, tr, repo, clinicA, "", func(ctx context.Context) error {
		_, err := repo.ByID(ctx, "not-a-uuid")
		return err
	})
	if !errors.Is(err, careplan.ErrNotFound) {
		t.Errorf("a malformed id: %v, want ErrNotFound", err)
	}
}
