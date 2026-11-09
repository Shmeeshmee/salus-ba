package suggestion_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedEncounter creates, as the owner, a clinic with a practitioner, a patient with an active consent, an
// appointment and its encounter in progress, and returns the clinic and the encounter. Synthetic.
func seedEncounter(t *testing.T, owner *pgxpool.Pool) (clinic, encounterID string) {
	t.Helper()
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id),
		a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		pa AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT pa.id, r.organization_id, 'active', $1::timestamptz - interval '30 days' FROM pa, r RETURNING patient_id),
		ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT c.patient_id, r.id, r.organization_id, gen_random_uuid(), tstzrange($1::timestamptz, $1::timestamptz + interval '30 minutes') FROM c, r
			RETURNING id, patient_id, practitioner_role_id, organization_id)
		INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', $1 FROM ap
		RETURNING organization_id::text, id::text`, at).Scan(&clinic, &encounterID)
	if err != nil {
		t.Fatal(err)
	}
	return clinic, encounterID
}

func save(t *testing.T, app *pgxpool.Pool, clinic string, e suggestion.Event) (suggestion.Saved, error) {
	t.Helper()
	var saved suggestion.Saved
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinic); err != nil {
			return err
		}
		var err error
		saved, err = suggestion.NewPostgresRepository(app).Save(ctx, e)
		return err
	})
	return saved, err
}

// A suggestion is stored once; a later event changes its status only, once.
func TestSave_StoresOnceAndThenChangesTheStatusOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	clinic, enc := seedEncounter(t, owner)
	e := valid()
	e.EncounterID = enc
	pinned := e
	pinned.Status, pinned.Text, pinned.Confidence = suggestion.StatusPinned, "Placeholder text changed", 0.1

	var got []suggestion.Saved
	for _, ev := range []suggestion.Event{e, e, pinned, pinned} {
		saved, err := save(t, app, clinic, ev)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		got = append(got, saved)
	}
	want := []suggestion.Saved{suggestion.SavedCreated, suggestion.SavedUnchanged, suggestion.SavedUpdated, suggestion.SavedUnchanged}
	if !slices.Equal(got, want) {
		t.Errorf("Save() = %v, want %v", got, want)
	}
	var text, status string
	var version int
	var confidence float32
	var sources []string
	if err := owner.QueryRow(t.Context(), `SELECT text, status, version_id, confidence, source_segment_ids::text[] FROM suggestion WHERE id = $1`, e.ID).
		Scan(&text, &status, &version, &confidence, &sources); err != nil {
		t.Fatal(err)
	}
	if text != e.Text || status != "pinned" || version != 2 || confidence != e.Confidence || !slices.Equal(sources, e.SourceSegmentIDs) {
		t.Errorf("stored %q %s v%d %v %v, want the first text and confidence, pinned at version 2", text, status, version, confidence, sources)
	}
}

// Another clinic's scope cannot write a suggestion of the encounter (row-level security).
func TestSave_AnotherClinicCannotWrite(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, enc := seedEncounter(t, owner)
	other, _ := seedEncounter(t, owner)
	e := valid()
	e.EncounterID = enc
	if _, err := save(t, app, other, e); err == nil {
		t.Error("Save() in another clinic's scope succeeded, want the row-level security refusal")
	}
}
