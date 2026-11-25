package diagnosticreport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seeded is a clinic with a practitioner's account, a patient with its active care consent and a visit, and a second
// patient without one. Synthetic.
type seeded struct {
	clinic, doctor, patient, visit, stranger, strangerVisit string
}

func seed(t *testing.T, owner *pgxpool.Pool) seeded {
	t.Helper()
	ctx := t.Context()
	var s seeded
	var role string
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id)
			SELECT gen_random_uuid()::text || '@example.test', 'h', 'practitioner', id FROM o RETURNING id)
		SELECT r.organization_id::text, r.id::text, a.id::text FROM r, a`).Scan(&s.clinic, &role, &s.doctor); err != nil {
		t.Fatal(err)
	}
	visit := func(consent bool, start string) (string, string) {
		var patient, appt string
		if err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
			p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
			ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
				SELECT id, $2, $1, gen_random_uuid(), tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes') FROM p RETURNING id, patient_id)
			SELECT patient_id::text, id::text FROM ap`, s.clinic, role, start).Scan(&patient, &appt); err != nil {
			t.Fatal(err)
		}
		if consent {
			if _, err := owner.Exec(ctx, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
				patient, s.clinic); err != nil {
				t.Fatal(err)
			}
		}
		return patient, appt
	}
	s.patient, s.visit = visit(true, "2026-11-02T09:00:00Z")
	s.stranger, s.strangerVisit = visit(false, "2026-11-02T10:00:00Z")
	return s
}

// in runs fn in a transaction of salus_app in the clinic's scope, or the patient's when patient is set.
func in(t *testing.T, app *pgxpool.Pool, repo diagnosticreport.Repository, clinic, patient string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if patient != "" {
			if err := repo.SetPatient(ctx, patient); err != nil {
				return err
			}
		} else if err := repo.SetOrganization(ctx, clinic); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// A report is stored with its values in their order, changed at its version, released once, and shown to the patient
// only once released.
func TestRepository_AttachChangeReleaseAndThePatientsRead(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seed(t, owner)
	repo := diagnosticreport.NewPostgresRepository(app)
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	draft := blood()
	draft.EffectiveDate = &day
	var stored diagnosticreport.Report
	if err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error {
		var err error
		stored, err = repo.Insert(ctx, draftReport(draft, s.clinic, s.patient, s.visit), s.doctor)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if stored.ID == "" || stored.VersionID != 1 || stored.Status != diagnosticreport.StatusPreliminary {
		t.Fatalf("Insert() = %+v", stored)
	}
	err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error {
		_, err := repo.Insert(ctx, draftReport(draft, s.clinic, s.stranger, s.strangerVisit), s.doctor)
		return err
	})
	if !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Errorf("a report for a patient without consent: %v, want ErrNotFound", err)
	}
	if err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error { return change(ctx, t, repo, s, stored.ID, day) }); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, repo, "", s.patient, func(ctx context.Context) error {
		list, err := repo.ForPatient(ctx, s.patient)
		if err != nil || len(list) != 0 {
			t.Errorf("the patient sees a preliminary report: %+v, %v", list, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error {
		if ok, err := repo.Release(ctx, stored.ID, s.doctor, 1, now); err != nil || ok {
			t.Errorf("Release() at a stale version = %v, %v", ok, err)
		}
		ok, err := repo.Release(ctx, stored.ID, s.doctor, 2, now)
		if err != nil || !ok {
			t.Fatalf("Release() = %v, %v", ok, err)
		}
		if ok, err := repo.Release(ctx, stored.ID, s.doctor, 3, now); err != nil || ok {
			t.Errorf("Release() again = %v, %v", ok, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, repo, "", s.patient, func(ctx context.Context) error {
		list, err := repo.ForPatient(ctx, s.patient)
		if err != nil || len(list) != 1 || list[0].Status != diagnosticreport.StatusFinal || list[0].ReleasedAt == nil || len(list[0].Results) != 1 {
			t.Errorf("ForPatient() = %+v, %v", list, err)
		}
		if _, err := repo.ByID(ctx, "not-a-uuid"); !errors.Is(err, diagnosticreport.ErrNotFound) {
			t.Errorf("ByID(not a uuid) = %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func draftReport(d diagnosticreport.Draft, clinic, patient, visit string) diagnosticreport.Report {
	return diagnosticreport.Report{
		OrganizationID: clinic, PatientID: patient, AppointmentID: visit, Title: d.Title, LOINC: d.LOINC,
		EffectiveDate: d.EffectiveDate, Results: d.Results,
	}
}

// change reads the stored report id, changes it at its version and once more at a stale one, and lists the visit's.
func change(ctx context.Context, t *testing.T, repo diagnosticreport.Repository, s seeded, id string, day time.Time) error {
	t.Helper()
	got, err := repo.ByID(ctx, id)
	if err != nil {
		return err
	}
	if len(got.Results) != 2 || *got.Results[0].Value != 13.5 || got.Results[0].Unit != "g/dL" || got.Results[1].ValueText != "negative" ||
		got.EffectiveDate == nil || !got.EffectiveDate.Equal(day) || got.LOINC != "58410-2" {
		t.Errorf("ByID() = %+v", got)
	}
	changed := got
	changed.Title, changed.Results, changed.VersionID = "Lipid panel", changed.Results[:1], 2
	if ok, err := repo.Update(ctx, changed); err != nil || !ok {
		t.Fatalf("Update() = %v, %v", ok, err)
	}
	if ok, err := repo.Update(ctx, changed); err != nil || ok {
		t.Errorf("Update() at a stale version = %v, %v", ok, err)
	}
	list, err := repo.ForAppointment(ctx, s.visit)
	if err != nil || len(list) != 1 || list[0].Title != "Lipid panel" || len(list[0].Results) != 1 {
		t.Errorf("ForAppointment() = %+v, %v", list, err)
	}
	return nil
}

// The clinic's policy re-checks the author on every change: once the author is no longer the clinic's practitioner, a
// change and a release are refused as forbidden, not as a fault.
func TestRepository_AReportOfAnAuthorWhoLeftTheRoleIsRefusedNotAFault(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seed(t, owner)
	repo := diagnosticreport.NewPostgresRepository(app)
	var stored diagnosticreport.Report
	if err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error {
		var err error
		stored, err = repo.Insert(ctx, draftReport(blood(), s.clinic, s.patient, s.visit), s.doctor)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE account SET role = 'reception' WHERE id = $1`, s.doctor); err != nil {
		t.Fatal(err)
	}
	stored.VersionID = 2
	err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error { _, err := repo.Update(ctx, stored); return err })
	if !errors.Is(err, diagnosticreport.ErrForbidden) {
		t.Errorf("Update() = %v, want ErrForbidden", err)
	}
}

// A new PDF moves a preliminary report to its next version, at its version only; a released one never moves.
func TestRepository_TouchMovesAPreliminaryReportOn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seed(t, owner)
	repo := diagnosticreport.NewPostgresRepository(app)
	if err := in(t, app, repo, s.clinic, "", func(ctx context.Context) error {
		r, err := repo.Insert(ctx, draftReport(blood(), s.clinic, s.patient, s.visit), s.doctor)
		if err != nil {
			return err
		}
		if ok, err := repo.Touch(ctx, r.ID, 1); err != nil || !ok {
			t.Errorf("Touch() = %v, %v", ok, err)
		}
		if ok, err := repo.Touch(ctx, r.ID, 1); err != nil || ok {
			t.Errorf("Touch() at a stale version = %v, %v", ok, err)
		}
		if got, err := repo.ByID(ctx, r.ID); err != nil || got.VersionID != 2 {
			t.Errorf("ByID() = %+v, %v", got, err)
		}
		if ok, err := repo.Release(ctx, r.ID, s.doctor, 2, now); err != nil || !ok {
			t.Fatalf("Release() = %v, %v", ok, err)
		}
		if ok, err := repo.Touch(ctx, r.ID, 3); err != nil || ok {
			t.Errorf("Touch() of a released report = %v, %v", ok, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
