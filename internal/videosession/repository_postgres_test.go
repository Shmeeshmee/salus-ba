package videosession_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seeded is a listed clinic with one practitioner role and its practitioner's account, and a patient with an
// active consent and an appointment booked from at for half an hour. Synthetic.
type seeded struct {
	clinic, role, practitioner, practitionerAccount, patient, patientAccount, appointment string
}

func seedDB(t testing.TB, owner *pgxpool.Pool, at time.Time) seeded {
	t.Helper()
	ctx := context.Background()
	var s seeded
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id, practitioner_id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, practitioner_id)
			SELECT gen_random_uuid()::text || '@example.test', 'h', 'practitioner', organization_id, practitioner_id FROM r RETURNING id)
		SELECT r.organization_id::text, r.id::text, r.practitioner_id::text, a.id::text FROM r, a`).
		Scan(&s.clinic, &s.role, &s.practitioner, &s.practitionerAccount)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id, account_id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', $2::timestamptz - interval '30 days' FROM p),
		b AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT id, $3, $1, gen_random_uuid(), tstzrange($2::timestamptz, $2::timestamptz + interval '30 minutes') FROM p RETURNING id)
		SELECT p.id::text, p.account_id::text, b.id::text FROM p, b`, s.clinic, at, s.role).
		Scan(&s.patient, &s.patientAccount, &s.appointment)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// session is a session to open for the appointment of s, with the names given.
func session(s seeded, room string, at time.Time) videosession.Session {
	return videosession.Session{
		AppointmentID: s.appointment, OrganizationID: s.clinic, Room: room, PatientIdentity: "p_pat_" + room, PractitionerIdentity: "p_pra_" + room,
		ClosesAt: at.Add(time.Hour), CreatedAt: at,
	}
}

// open opens sess in the patient's scope in a transaction of its own.
func open(t *testing.T, app *pgxpool.Pool, s seeded, sess videosession.Session) (videosession.Session, error) {
	t.Helper()
	repo := videosession.NewPostgresRepository(app)
	var out videosession.Session
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetPatient(ctx, s.patient); err != nil {
			return err
		}
		var err error
		out, _, err = repo.Open(ctx, sess)
		return err
	})
	return out, err
}

// inClinic runs fn on the repository in a transaction of its own in the clinic's scope.
func inClinic(t *testing.T, app *pgxpool.Pool, clinic string, fn func(ctx context.Context, r videosession.Repository) error) error {
	t.Helper()
	repo := videosession.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, clinic); err != nil {
			return err
		}
		return fn(ctx, repo)
	})
}

// Open says whether it created the session: the first request enqueues the window's close, a later one does
// not.
func TestOpen_SaysWhetherItCreatedTheSession(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	var created []bool
	for _, room := range []string{"r_first", "r_second"} {
		err := inClinic(t, app, s.clinic, func(ctx context.Context, r videosession.Repository) error {
			_, c, err := r.Open(ctx, session(s, room, at))
			created = append(created, c)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(created, []bool{true, false}) {
		t.Errorf("created %v, want the first only", created)
	}
}

// End ends a session once with its reason, in the clinic's scope; a second End changes nothing. ByID reads it
// in the scope.
func TestEnd_EndsASessionOnceAndByIDReadsIt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	opened, err := open(t, app, s, session(s, "r_end", at))
	if err != nil {
		t.Fatal(err)
	}
	ended := at.Add(40 * time.Minute)
	end := func(reason videosession.EndReason, when time.Time) (videosession.Session, bool, error) {
		var out videosession.Session
		var changed bool
		err := inClinic(t, app, s.clinic, func(ctx context.Context, r videosession.Repository) error {
			var err error
			out, changed, err = r.End(ctx, opened.ID, reason, when)
			return err
		})
		return out, changed, err
	}
	got, changed, err := end(videosession.EndWindowClosed, ended)
	if err != nil || !changed || got.ID != opened.ID || !got.EndedAt.Equal(ended) || got.EndReason != videosession.EndWindowClosed || got.Room != "r_end" {
		t.Fatalf("End() = %+v, %t, %v; want the session ended, window_closed", got, changed, err)
	}
	if again, changed, err := end(videosession.EndTimeLimit, ended.Add(time.Hour)); err != nil || changed || again != (videosession.Session{}) {
		t.Errorf("End() again = %+v, %t, %v; want no change", again, changed, err)
	}
	for _, tt := range []struct {
		want             error
		name, clinic, id string
	}{
		{name: "the clinic", clinic: s.clinic, id: opened.ID},
		{name: "another clinic", clinic: "b0000000-0000-4000-8000-00000000000b", id: opened.ID, want: videosession.ErrNotFound},
		{name: "not a UUID", clinic: s.clinic, id: "x", want: videosession.ErrNotFound},
	} {
		var read videosession.Session
		err := inClinic(t, app, tt.clinic, func(ctx context.Context, r videosession.Repository) error {
			var err error
			read, err = r.ByID(ctx, tt.id)
			return err
		})
		if !errors.Is(err, tt.want) || (tt.want == nil && read != got) {
			t.Errorf("%s: ByID() = %+v, %v; want %v", tt.name, read, err, tt.want)
		}
	}
	err = inClinic(t, app, s.clinic, func(ctx context.Context, r videosession.Repository) error {
		_, changed, err := r.End(ctx, "x", videosession.EndTimeLimit, ended)
		if changed {
			t.Error("End() with an id that is not a UUID changed a session")
		}
		return err
	})
	if err != nil {
		t.Errorf("End() with an id that is not a UUID = %v, want no change", err)
	}
}

func TestOpen_StoresTheFirstSessionAndReturnsItAfterwards(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)

	first, err := open(t, app, s, session(s, "r_first", at))
	want := session(s, "r_first", at)
	want.ID = first.ID
	if err != nil || first.ID == "" || first.CreatedAt.Unix() != at.Unix() || first.ClosesAt.Unix() != want.ClosesAt.Unix() {
		t.Fatalf("Open() = %+v, %v; want the session stored", first, err)
	}
	want.CreatedAt, want.ClosesAt = first.CreatedAt, first.ClosesAt
	if first != want {
		t.Errorf("Open() = %+v,\nwant %+v", first, want)
	}
	again, err := open(t, app, s, session(s, "r_second", at.Add(time.Minute)))
	if err != nil || again != first {
		t.Errorf("second Open() = %+v, %v; want the first session %+v", again, err, first)
	}
}

// Two first requests at once: one inserts, the other waits for its commit, does nothing and reads it.
func TestOpen_TwoFirstRequestsAtOnceShareOneSession(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)

	var wg sync.WaitGroup
	got := make([]videosession.Session, 2)
	errs := make([]error, 2)
	for i, room := range []string{"r_one", "r_two"} {
		wg.Go(func() { got[i], errs[i] = open(t, app, s, session(s, room, at)) })
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || got[0] != got[1] {
		t.Errorf("Open() at once = %+v, %v and %+v, %v; want one session for both", got[0], errs[0], got[1], errs[1])
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM video_session").Scan(&n); err != nil || n != 1 {
		t.Errorf("%d sessions, %v; want 1", n, err)
	}
}

// A clinic's scope opens the session of its appointment too, and reads the end of an ended one.
func TestOpen_InTheClinicsScopeGivesTheEndedSession(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	if _, err := open(t, app, s, session(s, "r_ended", at)); err != nil {
		t.Fatal(err)
	}
	ended := at.Add(20 * time.Minute)
	if _, err := owner.Exec(t.Context(), `UPDATE video_session SET patient_joined_at = $1, practitioner_joined_at = $1, ended_at = $2,
		end_reason = 'practitioner'`, at, ended); err != nil {
		t.Fatal(err)
	}
	repo := videosession.NewPostgresRepository(app)
	var got videosession.Session
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetClinic(ctx, s.clinic); err != nil {
			return err
		}
		var err error
		got, _, err = repo.Open(ctx, session(s, "r_other", at))
		return err
	})
	if err != nil || got.Room != "r_ended" || !got.EndedAt.Equal(ended) || got.EndReason != videosession.EndPractitioner ||
		!got.PatientJoinedAt.Equal(at) || !got.PractitionerJoinedAt.Equal(at) {
		t.Errorf("Open() in the clinic's scope = %+v, %v; want the ended session with both joins", got, err)
	}
}

// Outside the scope the insert is refused by row-level security and nothing is stored.
func TestOpen_FailsOutsideTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedDB(t, owner, at)
	s.patient = "aa000000-0000-4000-8000-0000000000aa" // no such patient
	if _, err := open(t, app, s, session(s, "r_x", at)); err == nil {
		t.Error("Open() in another patient's scope = nil, want an error")
	}
	bad := session(s, "r_y", at)
	bad.AppointmentID = "not-a-uuid"
	if _, err := open(t, app, s, bad); err == nil {
		t.Error("Open() with an appointment id that is not a UUID = nil, want an error")
	}
}

// The purger deletes, in every clinic, the sessions that ended before the cut-off and those never ended whose
// window closed before it; the rest stay.
func TestPurger_DeletesSessionsOfEveryClinicAfterTheirEndOrWindow(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	cutOff := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	gone, kept := map[string]bool{}, map[string]bool{}
	for _, tt := range []struct {
		ended, closes time.Time
		purged        bool
	}{
		{ended: cutOff.Add(-time.Second), closes: cutOff.Add(time.Hour), purged: true},
		{ended: cutOff.Add(time.Second), closes: cutOff.Add(-time.Hour)},
		{closes: cutOff.Add(-time.Second), purged: true},
		{closes: cutOff.Add(time.Second)},
	} {
		s := seedDB(t, owner, cutOff.Add(-48*time.Hour)) // a clinic of its own each
		var id string
		err := owner.QueryRow(t.Context(), `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity, practitioner_identity,
				closes_at, ended_at, end_reason)
			VALUES ($1::uuid, $2, 'r_' || $1::text, 'p_a' || $1::text, 'p_b' || $1::text, $3, $4, CASE WHEN $4::timestamptz IS NOT NULL THEN 'window_closed' END)
			RETURNING id::text`, s.appointment, s.clinic, tt.closes, pgtypeTime(tt.ended)).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		if tt.purged {
			gone[id] = true
		} else {
			kept[id] = true
		}
	}

	n, err := videosession.NewPurger(videosession.NewPostgresRepository(ret)).Purge(t.Context(), cutOff)

	if err != nil || n != 2 {
		t.Errorf("Purge() = %d, %v; want the two sessions past their period", n, err)
	}
	rows, err := owner.Query(t.Context(), "SELECT id::text FROM video_session")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		if gone[id] {
			t.Errorf("session %s past its period is still there", id)
		}
		delete(kept, id)
	}
	if len(kept) != 0 {
		t.Errorf("sessions %v inside their period were deleted", kept)
	}
}

// pgtypeTime is t for a query argument, NULL when zero.
func pgtypeTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
