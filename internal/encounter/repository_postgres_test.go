package encounter_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seeded is a clinic with one practitioner role, and a patient with an active consent and two appointments
// from at and an hour later. Synthetic.
type seeded struct {
	clinic, role, patient string
	appts                 [2]string
}

func seedDB(t testing.TB, owner *pgxpool.Pool, at time.Time) seeded {
	t.Helper()
	ctx := context.Background()
	var s seeded
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		SELECT organization_id::text, id::text FROM r`).Scan(&s.clinic, &s.role)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id)
		INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', $2::timestamptz - interval '30 days' FROM p
		RETURNING patient_id::text`, s.clinic, at).Scan(&s.patient)
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.appts {
		err = owner.QueryRow(ctx, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes')) RETURNING id::text`,
			s.patient, s.role, s.clinic, at.Add(time.Duration(i)*time.Hour)).Scan(&s.appts[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// inScope runs fn on the repository in a transaction of its own, scoped to the clinic (or the patient).
func inScope(t *testing.T, app *pgxpool.Pool, clinic, patient string, fn func(ctx context.Context, r encounter.Repository) error) error {
	t.Helper()
	r := encounter.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		var err error
		if clinic != "" {
			err = r.SetClinic(ctx, clinic)
		} else {
			err = r.SetPatient(ctx, patient)
		}
		if err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

func (s seeded) encounter(i int, at time.Time) encounter.Encounter {
	return encounter.Encounter{
		AppointmentID: s.appts[i], OrganizationID: s.clinic, PatientID: s.patient, PractitionerRoleID: s.role,
		Status: encounter.StatusInProgress, PeriodStart: at,
	}
}

func startIn(t *testing.T, app *pgxpool.Pool, s seeded, e encounter.Encounter) (encounter.Encounter, bool, error) {
	t.Helper()
	var out encounter.Encounter
	var created bool
	err := inScope(t, app, s.clinic, "", func(ctx context.Context, r encounter.Repository) error {
		var err error
		out, created, err = r.Start(ctx, e)
		return err
	})
	return out, created, err
}

// Start stores the first encounter of an appointment and gives it back afterwards, with the first time.
func TestStart_StoresTheFirstEncounterAndReturnsItAfterwards(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)

	first, created, err := startIn(t, app, s, s.encounter(0, at))
	want := s.encounter(0, at)
	want.ID, want.VersionID, want.PeriodStart, want.LastUpdated = first.ID, 1, first.PeriodStart, first.LastUpdated
	if err != nil || !created || first != want || !first.PeriodStart.Equal(at) || first.LastUpdated.IsZero() {
		t.Fatalf("Start() = %+v, %t, %v;\nwant %+v created", first, created, err, want)
	}
	again, created, err := startIn(t, app, s, s.encounter(0, at.Add(time.Minute)))
	if err != nil || created || again != first {
		t.Errorf("second Start() = %+v, %t, %v; want the first %+v", again, created, err, first)
	}
}

// Two starts at once (the webhook delivered twice): one row, both given the same encounter.
func TestStart_TwoAtOnceCreateOneEncounter(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)

	var wg sync.WaitGroup
	got := make([]encounter.Encounter, 2)
	created := make([]bool, 2)
	errs := make([]error, 2)
	for i := range got {
		wg.Go(func() {
			got[i], created[i], errs[i] = startIn(t, app, s, s.encounter(0, at.Add(time.Duration(i)*time.Second)))
		})
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || got[0] != got[1] || created[0] == created[1] {
		t.Errorf("Start() at once = %+v %t %v and %+v %t %v; want one encounter, created once", got[0], created[0], errs[0], got[1], created[1], errs[1])
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM encounter").Scan(&n); err != nil || n != 1 {
		t.Errorf("%d encounters, %v; want 1", n, err)
	}
}

// Finish ends an encounter in progress once; a second Finish, or one for an appointment without an
// encounter, changes nothing.
func TestFinish_FinishesAnEncounterInProgressOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	started, _, err := startIn(t, app, s, s.encounter(0, at))
	if err != nil {
		t.Fatal(err)
	}
	end := at.Add(25 * time.Minute)
	finish := func(appt string, when time.Time) (encounter.Encounter, bool, error) {
		var out encounter.Encounter
		var changed bool
		err := inScope(t, app, s.clinic, "", func(ctx context.Context, r encounter.Repository) error {
			var err error
			out, changed, err = r.Finish(ctx, appt, when)
			return err
		})
		return out, changed, err
	}
	got, changed, err := finish(s.appts[0], end)
	if err != nil || !changed || got.ID != started.ID || got.Status != encounter.StatusFinished || !got.PeriodEnd.Equal(end) ||
		!got.LastUpdated.Equal(end) || got.VersionID != 2 || !got.PeriodStart.Equal(at) {
		t.Fatalf("Finish() = %+v, %t, %v; want finished at the end, at version 2", got, changed, err)
	}
	for name, appt := range map[string]string{"again": s.appts[0], "never started": s.appts[1], "not a UUID": "x"} {
		if _, changed, err := finish(appt, end.Add(time.Minute)); err != nil || changed {
			t.Errorf("Finish() %s = %t, %v; want no change", name, changed, err)
		}
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM encounter").Scan(&n); err != nil || n != 1 {
		t.Errorf("%d encounters, %v; want 1", n, err)
	}
}

// ByID and Search read in the scope: the clinic's and the patient's own; nothing for another clinic.
func TestByIDAndSearch_ReadInTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	first, _, err := startIn(t, app, s, s.encounter(0, at))
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := startIn(t, app, s, s.encounter(1, at.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	const otherClinic = "b0000000-0000-4000-8000-00000000000b"
	for _, tt := range []struct {
		name, clinic, patient string
		filter                encounter.Filter
		want                  []string
	}{
		{name: "the clinic", clinic: s.clinic, want: []string{first.ID, second.ID}},
		{name: "the clinic by appointment", clinic: s.clinic, filter: encounter.Filter{AppointmentID: s.appts[1]}, want: []string{second.ID}},
		{name: "the clinic by patient", clinic: s.clinic, filter: encounter.Filter{PatientID: s.patient}, want: []string{first.ID, second.ID}},
		{name: "the patient", patient: s.patient, want: []string{first.ID, second.ID}},
		{name: "another clinic", clinic: otherClinic},
		{name: "another patient", clinic: s.clinic, filter: encounter.Filter{PatientID: otherClinic}},
		{name: "ids that are not UUIDs", clinic: s.clinic, filter: encounter.Filter{AppointmentID: "x", PatientID: "y"}},
	} {
		var got []encounter.Encounter
		err := inScope(t, app, tt.clinic, tt.patient, func(ctx context.Context, r encounter.Repository) error {
			var err error
			got, err = r.Search(ctx, tt.filter)
			return err
		})
		if err != nil || !slices.Equal(ids(got), tt.want) {
			t.Errorf("%s: Search() = %v, %v; want %v", tt.name, ids(got), err, tt.want)
		}
	}
	for _, tt := range []struct {
		want             error
		name, clinic, id string
	}{
		{name: "the clinic", clinic: s.clinic, id: first.ID},
		{name: "another clinic", clinic: otherClinic, id: first.ID, want: encounter.ErrNotFound},
		{name: "not a UUID", clinic: s.clinic, id: "x", want: encounter.ErrNotFound},
	} {
		var got encounter.Encounter
		err := inScope(t, app, tt.clinic, "", func(ctx context.Context, r encounter.Repository) error {
			var err error
			got, err = r.ByID(ctx, tt.id)
			return err
		})
		if !errors.Is(err, tt.want) || (tt.want == nil && got != first) {
			t.Errorf("%s: ByID() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// Organization gives the clinic of an encounter outside any scope, and ErrNotFound for an unknown id or one
// that is not a UUID.
func TestOrganization_GivesTheClinicOfAnEncounterOutsideAnyScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 2, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	e, _, err := startIn(t, app, s, s.encounter(0, at))
	if err != nil {
		t.Fatal(err)
	}
	r := encounter.NewPostgresRepository(app)
	if got, err := r.Organization(t.Context(), e.ID); err != nil || got != s.clinic {
		t.Errorf("Organization() = %q, %v; want %s", got, err, s.clinic)
	}
	for _, id := range []string{"00000000-0000-4000-8000-000000000000", "Encounter/1"} {
		if _, err := r.Organization(t.Context(), id); !errors.Is(err, encounter.ErrNotFound) {
			t.Errorf("Organization(%q) error = %v, want ErrNotFound", id, err)
		}
	}
}
