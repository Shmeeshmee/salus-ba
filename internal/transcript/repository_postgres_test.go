package transcript_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedEncounter creates, as the owner, a clinic with a practitioner, a patient with an active consent, an
// appointment at at and its encounter in progress, and returns the clinic and the encounter. Synthetic.
func seedEncounter(t testing.TB, owner *pgxpool.Pool, at time.Time) (clinic, encounterID string) {
	t.Helper()
	err := owner.QueryRow(context.Background(), `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
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

// inClinic runs fn on the repository in a transaction of its own, scoped to the clinic.
func inClinic(t testing.TB, app *pgxpool.Pool, clinic string, fn func(ctx context.Context, r transcript.Repository) error) error {
	t.Helper()
	r := transcript.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(context.Background(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinic); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

func segmentOf(encounterID, id string, at time.Time) transcript.Event {
	return transcript.Event{
		EncounterID: encounterID, SegmentID: id, Speaker: transcript.SpeakerPatient, Lang: "bs",
		Original: "placeholder original", Translation: "[de] placeholder original", IsFinal: true,
		TsStart: at, TsEnd: at.Add(1500 * time.Millisecond),
	}
}

// A segment is stored once under its id, and read back in the order it was said, as a final event, with its
// translation or the flag that its translation failed.
func TestInsert_StoresASegmentOnceAndByEncounterReadsThemInOrder(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	clinic, enc := seedEncounter(t, owner, at)
	second, first := segmentOf(enc, segmentID2, at.Add(time.Minute)), segmentOf(enc, segmentID, at)
	first.Translation, first.TranslationFailed = "", true

	var created []bool
	err := inClinic(t, app, clinic, func(ctx context.Context, r transcript.Repository) error {
		for _, e := range []transcript.Event{second, first, second} {
			ok, err := r.Insert(ctx, e)
			if err != nil {
				return err
			}
			created = append(created, ok)
		}
		return nil
	})
	if err != nil || !slices.Equal(created, []bool{true, true, false}) {
		t.Fatalf("Insert() created %v, %v; want true, true, false", created, err)
	}
	var got []transcript.Event
	err = inClinic(t, app, clinic, func(ctx context.Context, r transcript.Repository) error {
		var err error
		got, err = r.ByEncounter(ctx, enc)
		return err
	})
	if err != nil || len(got) != 2 || got[0] != first || got[1] != second {
		t.Errorf("ByEncounter() = %+v, %v;\nwant %+v and %+v", got, err, first, second)
	}
}

// Another clinic's scope neither reads nor writes the encounter's transcript (row-level security), and an
// encounter id that is no UUID reads nothing.
func TestInsert_AnotherClinicNeitherReadsNorWritesTheTranscript(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	clinic, enc := seedEncounter(t, owner, at)
	other, _ := seedEncounter(t, owner, at)
	if err := inClinic(t, app, clinic, func(ctx context.Context, r transcript.Repository) error {
		_, err := r.Insert(ctx, segmentOf(enc, segmentID, at))
		return err
	}); err != nil {
		t.Fatal(err)
	}

	err := inClinic(t, app, other, func(ctx context.Context, r transcript.Repository) error {
		_, err := r.Insert(ctx, segmentOf(enc, segmentID2, at))
		return err
	})
	if err == nil {
		t.Error("Insert() in another clinic's scope succeeded, want the row-level security refusal")
	}
	for name, id := range map[string]string{"another clinic": enc, "no UUID": "Encounter/1"} {
		var got []transcript.Event
		err := inClinic(t, app, other, func(ctx context.Context, r transcript.Repository) error {
			var err error
			got, err = r.ByEncounter(ctx, id)
			return err
		})
		if err != nil || len(got) != 0 {
			t.Errorf("%s: ByEncounter() = %+v, %v; want none", name, got, err)
		}
	}
}

// asPatient runs fn on the repository in a transaction of its own, scoped to the patient.
func asPatient(t testing.TB, app *pgxpool.Pool, patient string, fn func(ctx context.Context, r transcript.Repository) error) error {
	t.Helper()
	r := transcript.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(context.Background(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.patient_id", patient); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

// SAL-17: the patient's own scope reads the stored transcript of their own encounter (the service decides whether
// the clinic shares it) and writes none; another patient's scope reads nothing.
func TestByEncounter_ThePatientReadsOnlyTheirOwnTranscript(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	clinic, enc := seedEncounter(t, owner, at)
	_, otherEnc := seedEncounter(t, owner, at)
	if err := inClinic(t, app, clinic, func(ctx context.Context, r transcript.Repository) error {
		_, err := r.Insert(ctx, segmentOf(enc, segmentID, at))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var patient, other string
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT patient_id::text FROM encounter WHERE id = $1), (SELECT patient_id::text FROM encounter WHERE id = $2)`,
		enc, otherEnc).Scan(&patient, &other); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		patient string
		want    int
	}{"the patient": {patient, 1}, "another patient": {other, 0}} {
		var got []transcript.Event
		err := asPatient(t, app, tc.patient, func(ctx context.Context, r transcript.Repository) error {
			var err error
			got, err = r.ByEncounter(ctx, enc)
			return err
		})
		if err != nil || len(got) != tc.want {
			t.Errorf("%s: ByEncounter() = %d segments, %v; want %d", name, len(got), err, tc.want)
		}
	}
	err := asPatient(t, app, patient, func(ctx context.Context, r transcript.Repository) error {
		_, err := r.Insert(ctx, segmentOf(enc, segmentID2, at))
		return err
	})
	if err == nil {
		t.Error("Insert() in the patient's scope succeeded, want the refusal")
	}
}
