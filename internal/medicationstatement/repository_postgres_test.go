package medicationstatement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// visit is a clinic with a practitioner role, a patient with the care consent there, and an encounter. Synthetic.
type visit struct {
	clinic, role, patient, encounter, term string
}

func seedVisit(t *testing.T, owner *pgxpool.Pool) visit {
	t.Helper()
	ctx := t.Context()
	var v visit
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		SELECT organization_id::text, id::text FROM r`).Scan(&v.clinic, &v.role)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', now() FROM p),
		ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT id, $2, $1, gen_random_uuid(), tstzrange('2026-11-02T09:00:00Z', '2026-11-02T09:30:00Z') FROM p RETURNING *),
		e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM ap RETURNING id, patient_id)
		SELECT patient_id::text, id::text FROM e`, v.clinic, v.role).Scan(&v.patient, &v.encounter)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `INSERT INTO term (category, atc) VALUES ('drug', 'J01CA04') RETURNING id::text`).Scan(&v.term)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// in runs fn in a transaction of salus_app in the scope set.
func in(t *testing.T, app *pgxpool.Pool, set func(ctx context.Context, r medicationstatement.Repository) error,
	fn func(ctx context.Context, r medicationstatement.Repository) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		r := medicationstatement.NewPostgresRepository(app)
		if err := set(ctx, r); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

var at = time.Date(2026, 11, 2, 9, 10, 0, 0, time.UTC)

const (
	ownID    = "a1000000-0000-4000-8000-000000000001"
	clinicID = "a1000000-0000-4000-8000-000000000002"
)

// The patient writes their own entry and reads it back; the clinic with the care consent sees it in the profile, and
// records one in its visit, confirmed.
func TestRepository_WritesAndReadsInEachScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetClinic(ctx, v.clinic) }
	own := medicationstatement.MedicationStatement{ID: ownID, PatientID: v.patient, DateAsserted: at, InProfile: true,
		Code: medicationstatement.Code{TermID: v.term, Text: "Placeholder substance"}, Status: medicationstatement.StatusActive,
		Dosage: medicationstatement.Dosage{DoseValue: 2.5, DoseUnit: "mg", Frequency: 3, Period: 0.5, PeriodUnit: "d", AsNeeded: true}}
	if err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error { return r.Insert(ctx, own) }); err != nil {
		t.Fatalf("the patient inserts: %v", err)
	}
	recorded := medicationstatement.MedicationStatement{ID: clinicID, PatientID: v.patient, DateAsserted: at.Add(time.Minute),
		Source:       medicationstatement.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation: medicationstatement.Confirmation{At: at.Add(time.Minute), RoleID: v.role, OrganizationID: v.clinic},
		Code:         medicationstatement.Code{Text: "Placeholder food"},
		Status:       medicationstatement.StatusActive}
	if err := in(t, app, clinic, func(ctx context.Context, r medicationstatement.Repository) error { return r.Insert(ctx, recorded) }); err != nil {
		t.Fatalf("the clinic records: %v", err)
	}
	var got []medicationstatement.MedicationStatement
	err := in(t, app, clinic, func(ctx context.Context, r medicationstatement.Repository) (err error) {
		got, err = r.ByPatient(ctx, v.patient)
		return err
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("the clinic reads %d, %v; want both", len(got), err)
	}
	if g := got[0]; g.ID != clinicID || g.Source != recorded.Source || !g.Confirmation.At.Equal(recorded.Confirmation.At) ||
		g.Confirmation.RoleID != v.role || g.InProfile || g.VersionID != 1 || !g.LastUpdated.Equal(recorded.DateAsserted) {
		t.Errorf("the newest = %+v, want the clinic's entry as recorded", g)
	}
	if g := got[1]; g.Code != own.Code || g.Dosage != own.Dosage || g.Status != own.Status ||
		!g.Source.Patient() || !g.InProfile || !g.Confirmation.At.IsZero() {
		t.Errorf("the patient's = %+v, want it as written", g)
	}
}

// An update writes the next version over the one read; a stale version is a conflict; what the scope may not change
// is refused (the trigger); the patient deletes their own unconfirmed entry.
func TestRepository_UpdatesAtTheVersionAndDeletes(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetClinic(ctx, v.clinic) }
	own := medicationstatement.MedicationStatement{ID: ownID, PatientID: v.patient, DateAsserted: at, InProfile: true,
		Code: medicationstatement.Code{Text: "Placeholder substance"}, Status: medicationstatement.StatusActive}
	recorded := medicationstatement.MedicationStatement{ID: clinicID, PatientID: v.patient, DateAsserted: at,
		Source:       medicationstatement.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation: medicationstatement.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:         medicationstatement.Code{Text: "Placeholder food"},
		Status:       medicationstatement.StatusActive}
	if err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error { return r.Insert(ctx, own) }); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, clinic, func(ctx context.Context, r medicationstatement.Repository) error { return r.Insert(ctx, recorded) }); err != nil {
		t.Fatal(err)
	}
	edit := own
	edit.VersionID, edit.LastUpdated, edit.Status = 1, at.Add(time.Hour), medicationstatement.StatusStopped
	var version int32
	err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) (err error) {
		version, err = r.Update(ctx, edit)
		return err
	})
	if err != nil || version != 2 {
		t.Fatalf("the update: %d, %v; want version 2", version, err)
	}
	err = in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error {
		_, err := r.Update(ctx, edit)
		return err
	})
	if !errors.Is(err, medicationstatement.ErrVersionConflict) {
		t.Errorf("an update at the old version: %v, want ErrVersionConflict", err)
	}
	theirs := recorded
	theirs.VersionID, theirs.LastUpdated, theirs.Code.Text = 1, at.Add(time.Hour), "Placeholder change"
	err = in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error {
		_, err := r.Update(ctx, theirs)
		return err
	})
	if !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Errorf("the patient edits the clinic's entry: %v, want ErrForbidden", err)
	}
	if err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error {
		return r.Delete(ctx, ownID, 1)
	}); !errors.Is(err, medicationstatement.ErrVersionConflict) {
		t.Errorf("a delete at the old version: %v, want ErrVersionConflict", err)
	}
	if err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error {
		return r.Delete(ctx, ownID, 2)
	}); err != nil {
		t.Fatalf("the delete: %v", err)
	}
	err = in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) error {
		_, err := r.ByID(ctx, ownID)
		return err
	})
	if !errors.Is(err, medicationstatement.ErrNotFound) {
		t.Errorf("the deleted entry: %v, want ErrNotFound", err)
	}
}

// A clinic writes nothing into another clinic's visit; an input the table does not take is ErrInvalid.
func TestRepository_RefusesWhatTheScopeMayNotWrite(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	other := func(ctx context.Context, r medicationstatement.Repository) error {
		return r.SetClinic(ctx, "00000000-0000-4000-8000-0000000000ff")
	}
	recorded := medicationstatement.MedicationStatement{ID: clinicID, PatientID: v.patient, DateAsserted: at,
		Source:       medicationstatement.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation: medicationstatement.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:         medicationstatement.Code{Text: "Placeholder food"},
		Status:       medicationstatement.StatusActive}
	if err := in(t, app, other, func(ctx context.Context, r medicationstatement.Repository) error {
		return r.Insert(ctx, recorded)
	}); !errors.Is(err, medicationstatement.ErrForbidden) {
		t.Errorf("another clinic's scope: %v, want ErrForbidden", err)
	}
	bad := recorded
	bad.Dosage = medicationstatement.Dosage{DoseValue: 1} // without a unit
	if err := in(t, app, func(ctx context.Context, r medicationstatement.Repository) error { return r.SetClinic(ctx, v.clinic) },
		func(ctx context.Context, r medicationstatement.Repository) error { return r.Insert(ctx, bad) }); !errors.Is(err, medicationstatement.ErrInvalid) {
		t.Errorf("a dose without a unit: %v, want ErrInvalid", err)
	}
}
