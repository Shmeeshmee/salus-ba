package suggestion_test

import (
	"context"
	"errors"
	"reflect"
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

// inScope runs fn in a transaction scoped to the clinic, as the encounter's service sets it.
func inScope(t *testing.T, app *pgxpool.Pool, clinic string, fn func(ctx context.Context, r suggestion.Repository) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinic); err != nil {
			return err
		}
		return fn(ctx, suggestion.NewPostgresRepository(app))
	})
}

func insert(t *testing.T, app *pgxpool.Pool, clinic string, e suggestion.Event) (bool, error) {
	t.Helper()
	var created bool
	err := inScope(t, app, clinic, func(ctx context.Context, r suggestion.Repository) error {
		var err error
		created, err = r.Insert(ctx, e)
		return err
	})
	return created, err
}

// A suggestion is stored once; a later insert of it changes nothing.
func TestInsert_StoresOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	clinic, enc := seedEncounter(t, owner)
	e := valid()
	e.EncounterID = enc
	pinned := e
	pinned.Status, pinned.Text, pinned.Confidence = suggestion.StatusPinned, "Placeholder text changed", 0.1

	var got []bool
	for _, ev := range []suggestion.Event{e, e, pinned} {
		created, err := insert(t, app, clinic, ev)
		if err != nil {
			t.Fatalf("Insert() error = %v", err)
		}
		got = append(got, created)
	}
	if want := []bool{true, false, false}; !slices.Equal(got, want) {
		t.Errorf("Insert() = %v, want %v", got, want)
	}
	var text, status string
	var version int
	var confidence float32
	var sources []string
	if err := owner.QueryRow(t.Context(), `SELECT text, status, version_id, confidence, source_segment_ids::text[] FROM suggestion WHERE id = $1`, e.ID).
		Scan(&text, &status, &version, &confidence, &sources); err != nil {
		t.Fatal(err)
	}
	if text != e.Text || status != "shown" || version != 1 || confidence != e.Confidence || !slices.Equal(sources, e.SourceSegmentIDs) {
		t.Errorf("stored %q %s v%d %v %v, want the first event at version 1", text, status, version, confidence, sources)
	}
}

// Another clinic's scope cannot write a suggestion of the encounter (row-level security).
func TestInsert_AnotherClinicCannotWrite(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, enc := seedEncounter(t, owner)
	other, _ := seedEncounter(t, owner)
	e := valid()
	e.EncounterID = enc
	if _, err := insert(t, app, other, e); err == nil {
		t.Error("Insert() in another clinic's scope succeeded, want the row-level security refusal")
	}
}

// SetStatus changes the status of a suggestion of the encounter and gives it back as an event; the same status
// again changes nothing. A suggestion of another encounter, or one another clinic's scope does not show, is
// ErrNotFound.
func TestSetStatus_ChangesTheStatusOfASuggestionOfTheEncounter(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	clinic, enc := seedEncounter(t, owner)
	other, otherEnc := seedEncounter(t, owner)
	e := valid()
	e.EncounterID = enc
	if _, err := insert(t, app, clinic, e); err != nil {
		t.Fatal(err)
	}
	setStatus := func(scope, encounterID string, status suggestion.Status) (suggestion.Event, bool, error) {
		var got suggestion.Event
		var changed bool
		err := inScope(t, app, scope, func(ctx context.Context, r suggestion.Repository) error {
			var err error
			got, changed, err = r.SetStatus(ctx, encounterID, e.ID, status)
			return err
		})
		return got, changed, err
	}
	want := e
	want.Status = suggestion.StatusDismissed
	for i, wantChanged := range []bool{true, false} {
		got, changed, err := setStatus(clinic, enc, suggestion.StatusDismissed)
		if err != nil || changed != wantChanged || !reflect.DeepEqual(got, want) {
			t.Errorf("SetStatus() #%d = %+v, %v, %v; want %+v, %v", i, got, changed, err, want, wantChanged)
		}
	}
	var version int
	if err := owner.QueryRow(t.Context(), `SELECT version_id FROM suggestion WHERE id = $1`, e.ID).Scan(&version); err != nil || version != 2 {
		t.Errorf("version %d, %v; want 2", version, err)
	}
	for name, tt := range map[string][2]string{"another encounter": {clinic, otherEnc}, "another clinic": {other, enc}} {
		if _, _, err := setStatus(tt[0], tt[1], suggestion.StatusPinned); !errors.Is(err, suggestion.ErrNotFound) {
			t.Errorf("%s: SetStatus() error = %v, want ErrNotFound", name, err)
		}
	}
}

// ByEncounter gives the encounter's suggestions, oldest first, with their statuses; another clinic's scope sees
// none.
func TestByEncounter_GivesTheEncountersSuggestions(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	clinic, enc := seedEncounter(t, owner)
	other, otherEnc := seedEncounter(t, owner)
	first, second, elsewhere := valid(), valid(), valid()
	first.EncounterID, second.EncounterID, elsewhere.EncounterID = enc, enc, otherEnc
	second.ID, elsewhere.ID = segmentID, unknownID
	second.Type, second.Text, second.Status = suggestion.TypeRedFlag, "Placeholder red flag", suggestion.StatusPinned
	second.SourceSegmentIDs = []string{suggestionID, segmentID}
	for _, ev := range []struct {
		clinic string
		e      suggestion.Event
	}{{clinic, first}, {clinic, second}, {other, elsewhere}} {
		if _, err := insert(t, app, ev.clinic, ev.e); err != nil {
			t.Fatal(err)
		}
	}
	byEncounter := func(scope string) ([]suggestion.Event, error) {
		var got []suggestion.Event
		err := inScope(t, app, scope, func(ctx context.Context, r suggestion.Repository) error {
			var err error
			got, err = r.ByEncounter(ctx, enc)
			return err
		})
		return got, err
	}
	if got, err := byEncounter(clinic); err != nil || !reflect.DeepEqual(got, []suggestion.Event{first, second}) {
		t.Errorf("ByEncounter() = %+v, %v; want %+v", got, err, []suggestion.Event{first, second})
	}
	if got, err := byEncounter(other); err != nil || len(got) != 0 {
		t.Errorf("ByEncounter() in another clinic's scope = %+v, %v; want none", got, err)
	}
}
