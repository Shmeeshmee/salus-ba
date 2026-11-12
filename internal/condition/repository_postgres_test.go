package condition_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// signed is a clinic with a practitioner role, a patient, an encounter and its signed document. Synthetic.
type signed struct {
	clinic, role, patient, encounter, document string
}

func seedSigned(t *testing.T, owner *pgxpool.Pool) signed {
	t.Helper()
	ctx := t.Context()
	var s signed
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		SELECT organization_id::text, id::text FROM r`).Scan(&s.clinic, &s.role)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
		ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT id, $2, $1, gen_random_uuid(), tstzrange('2026-11-02T09:00:00Z', '2026-11-02T09:30:00Z') FROM p RETURNING *),
		e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM ap RETURNING id, patient_id),
		tm AS (INSERT INTO document_template (organization_id, title, fields)
			VALUES ($1, 'Placeholder template', '[{"key":"diagnosis","type":"coded","label":"Placeholder"}]') RETURNING id),
		d AS (INSERT INTO document_draft (encounter_id, template_id, status, fields)
			SELECT e.id, tm.id, 'entered-in-error', '[{"key":"diagnosis","missing":true}]' FROM e, tm RETURNING id, encounter_id)
		SELECT e.patient_id::text, e.id::text, d.id::text FROM e, d`, s.clinic, s.role).Scan(&s.patient, &s.encounter, &s.document)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (s signed) condition(key, code, status string, at time.Time) condition.Condition {
	return condition.Condition{
		ID: "00000000-0000-4000-8000-0000000001" + fmt.Sprintf("%02x", key[0]), RecordedAt: at, OrganizationID: s.clinic, PatientID: s.patient, EncounterID: s.encounter, DocumentID: s.document,
		PractitionerRoleID: s.role, FieldKey: key, Code: code, VerificationStatus: status,
	}
}

// inClinic runs fn in a transaction of salus_app scoped to the clinic, as composition's signing does.
func inClinic(t *testing.T, app *pgxpool.Pool, clinic string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinic); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// The clinic writes its Conditions in its scope; salus_stats reads every clinic's confirmed ones recorded in the
// range, oldest first, and the owner lists the unconfirmed ones.
func TestRepository_InsertsAndReadsTheDiagnoses(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedSigned(t, owner)
	at := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	err := inClinic(t, app, s.clinic, func(ctx context.Context) error {
		return condition.NewPostgresRepository(app).Insert(ctx, []condition.Condition{
			s.condition("b", "I10", condition.VerificationConfirmed, at.Add(time.Hour)),
			s.condition("a", "J06.9", condition.VerificationConfirmed, at),
			s.condition("c", "J00", condition.VerificationUnconfirmed, at),
			s.condition("d", "E11.9", condition.VerificationConfirmed, at.Add(48*time.Hour)), // outside the range
		})
	})
	if err != nil {
		t.Fatalf("the clinic inserts: %v", err)
	}
	again := s.condition("a", "J00", condition.VerificationConfirmed, at)
	again.ID = "00000000-0000-4000-8000-0000000002aa" // a new Condition, of a field that has one
	err = inClinic(t, app, s.clinic, func(ctx context.Context) error {
		return condition.NewPostgresRepository(app).Insert(ctx, []condition.Condition{again})
	})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "23505" || pgErr.ConstraintName != "condition_document_field_key" {
		t.Errorf("a second Condition of the same document field: %v, want the unique violation of condition_document_field_key", err)
	}
	stats := statsPool(t, owner)
	got, err := condition.NewPostgresRepository(stats).Diagnoses(t.Context(), s.clinic, at, at.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Code != "J06.9" || got[1].Code != "I10" || got[0].PatientID != s.patient ||
		got[0].PractitionerRoleID != s.role || !got[0].RecordedAt.Equal(at) {
		t.Errorf("Diagnoses = %+v, want J06.9 then I10, confirmed and in the range", got)
	}
	ids, err := condition.NewPostgresRepository(owner).Unconfirmed(t.Context())
	if err != nil || len(ids) != 1 {
		t.Errorf("Unconfirmed = %v, %v; want the one", ids, err)
	}
}

// statsPool logs in as salus_stats on owner's database.
func statsPool(t *testing.T, owner *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(t.Context(), pgtest.StatsURL(t, owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// A clinic cannot write a Condition into another clinic's scope.
func TestRepository_TheClinicWritesOnlyItsOwn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedSigned(t, owner)
	err := inClinic(t, app, "00000000-0000-4000-8000-0000000000ff", func(ctx context.Context) error {
		return condition.NewPostgresRepository(app).Insert(ctx, []condition.Condition{s.condition("a", "J00", condition.VerificationConfirmed, time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC))})
	})
	if err == nil {
		t.Error("another clinic's scope wrote the Condition")
	}
}

// ByID reads a Condition in the clinic's scope and the patient's, with its meta, and not in another clinic's.
func TestReadRepository_ReadsInTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedSigned(t, owner)
	at := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	c := s.condition("a", "J06.9", condition.VerificationConfirmed, at)
	if err := inClinic(t, app, s.clinic, func(ctx context.Context) error {
		return condition.NewPostgresRepository(app).Insert(ctx, []condition.Condition{c})
	}); err != nil {
		t.Fatal(err)
	}
	read := func(set func(ctx context.Context, r condition.ReadRepository) error) (condition.Condition, error) {
		var got condition.Condition
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			r := condition.NewPostgresReadRepository(app)
			if err := set(ctx, r); err != nil {
				return err
			}
			var err error
			got, err = r.ByID(ctx, c.ID)
			return err
		})
		return got, err
	}
	got, err := read(func(ctx context.Context, r condition.ReadRepository) error { return r.SetClinic(ctx, s.clinic) })
	if err != nil || got.Code != "J06.9" || got.VersionID != 1 || got.LastUpdated.IsZero() || got.DocumentID != s.document || !got.RecordedAt.Equal(at) {
		t.Errorf("the clinic: %+v, %v", got, err)
	}
	if _, err := read(func(ctx context.Context, r condition.ReadRepository) error { return r.SetPatient(ctx, s.patient) }); err != nil {
		t.Errorf("the patient: %v", err)
	}
	if _, err := read(func(ctx context.Context, r condition.ReadRepository) error {
		return r.SetClinic(ctx, "00000000-0000-4000-8000-0000000000ff")
	}); !errors.Is(err, condition.ErrNotFound) {
		t.Errorf("another clinic: %v, want ErrNotFound", err)
	}
}

// Confirmed reads, in the clinic's scope, the confirmed coded fields of the documents: never an unconfirmed one, nor
// another clinic's (migrations/history_questionnaire_test.go covers a shared visit's lookup).
func TestReadRepository_ReadsTheConfirmedFields(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedSigned(t, owner)
	at := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	if err := inClinic(t, app, s.clinic, func(ctx context.Context) error {
		return condition.NewPostgresRepository(app).Insert(ctx, []condition.Condition{
			s.condition("a", "J06.9", condition.VerificationConfirmed, at), s.condition("b", "K21.0", condition.VerificationUnconfirmed, at),
		})
	}); err != nil {
		t.Fatal(err)
	}
	read := func(clinic string) map[string][]string {
		t.Helper()
		var got map[string][]string
		if err := inClinic(t, app, clinic, func(ctx context.Context) error {
			var err error
			got, err = condition.NewPostgresReadRepository(app).Confirmed(ctx, clinic, "00000000-0000-4000-8000-0000000000f1", []string{s.document, "nope"})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := read(s.clinic); len(got) != 1 || !slices.Equal(got[s.document], []string{"a"}) {
		t.Errorf("the clinic: %v, want field a of the document", got)
	}
	if got := read("00000000-0000-4000-8000-0000000000ff"); len(got) != 0 {
		t.Errorf("another clinic: %v, want none", got)
	}
}
