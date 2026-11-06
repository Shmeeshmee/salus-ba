package questionnaireresponse_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seeded is a clinic with one practitioner role and two active questionnaires, and a patient with an active
// consent and two appointments, from at and an hour later. Synthetic.
type seeded struct {
	clinic, role, patient string
	appts                 [2]string
	versions, series      [2]string
}

func seedDB(t testing.TB, owner *pgxpool.Pool, at time.Time) seeded {
	t.Helper()
	ctx := t.Context()
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
		err = owner.QueryRow(ctx, `INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
			VALUES ($1, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', now()) RETURNING id::text, series_id::text`,
			s.clinic).Scan(&s.versions[i], &s.series[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// request is the response of appointment i to version v, as the service asks for it.
func (s seeded) request(i, v int) questionnaireresponse.Response {
	return questionnaireresponse.Response{
		OrganizationID: s.clinic, AppointmentID: s.appts[i], PatientID: s.patient, PractitionerRoleID: s.role,
		QuestionnaireID: s.versions[v], SeriesID: s.series[v], Source: questionnaireresponse.SourceAdded,
	}
}

// inScope runs fn on the repository in a transaction of its own, scoped to the clinic, or to the patient
// when clinic is empty.
func inScope(t *testing.T, app *pgxpool.Pool, clinic, patient string, fn func(ctx context.Context, r questionnaireresponse.Repository) error) error {
	t.Helper()
	r := questionnaireresponse.NewPostgresRepository(app)
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

var dbStart = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)

// The clinic requests; a second live request of a series is refused without an error (no row), and a
// withdrawn one may be requested again.
func TestPostgres_InsertStoresOneLiveRequestPerSeriesAndAppointment(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	at := dbStart.Add(-48 * time.Hour)
	in := s.request(0, 0)
	in.DueAt = dbStart

	var first questionnaireresponse.Response
	err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
		var created bool
		var err error
		if first, created, err = r.Insert(ctx, in, at); err != nil || !created {
			return errors.Join(err, errors.New("the first request was not created"))
		}
		if _, created, err := r.Insert(ctx, in, at); err != nil || created {
			return errors.Join(err, errors.New("a second live request was created"))
		}
		if _, err := r.Stop(ctx, first.ID, at); err != nil {
			return err
		}
		if _, created, err := r.Insert(ctx, in, at); err != nil || !created {
			return errors.Join(err, errors.New("the request after a withdrawal was not created"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := in
	want.ID, want.Status, want.VersionID, want.LastUpdated = first.ID, questionnaireresponse.StatusInProgress, 1, at
	first.DueAt, first.LastUpdated = first.DueAt.UTC(), first.LastUpdated.UTC() // the database's location
	if !reflect.DeepEqual(first, want) {
		t.Errorf("Insert() = %+v,\nwant %+v", first, want)
	}
}

// ByID and Lock in each scope; an id that is not a UUID or of another scope is ErrNotFound.
func TestPostgres_ByIDAndLockShowTheResponseInItsScopes(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	other := seedDB(t, owner, dbStart.Add(24*time.Hour))
	id := insert(t, app, s, 0, 0)

	for name, scope := range map[string][2]string{"clinic": {s.clinic, ""}, "patient": {"", s.patient}} {
		err := inScope(t, app, scope[0], scope[1], func(ctx context.Context, r questionnaireresponse.Repository) error {
			for _, read := range []func(context.Context, string) (questionnaireresponse.Response, error){r.ByID, r.Lock} {
				if got, err := read(ctx, id); err != nil || got.ID != id || got.AppointmentID != s.appts[0] {
					return errors.Join(err, errors.New("not the response"))
				}
			}
			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, scope := range map[string][2]string{"another clinic": {other.clinic, ""}, "another patient": {"", other.patient}} {
		err := inScope(t, app, scope[0], scope[1], func(ctx context.Context, r questionnaireresponse.Repository) error {
			if _, err := r.ByID(ctx, id); !errors.Is(err, questionnaireresponse.ErrNotFound) {
				return errors.Join(err, errors.New("ByID found it"))
			}
			if _, err := r.Lock(ctx, id); !errors.Is(err, questionnaireresponse.ErrNotFound) {
				return errors.Join(err, errors.New("Lock found it"))
			}
			if _, err := r.ByID(ctx, "not-a-uuid"); !errors.Is(err, questionnaireresponse.ErrNotFound) {
				return errors.Join(err, errors.New("a malformed id"))
			}
			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func insert(t *testing.T, app *pgxpool.Pool, s seeded, appt, version int) string {
	t.Helper()
	var id string
	in := s.request(appt, version)
	in.DueAt = dbStart.Add(time.Duration(appt) * time.Hour)
	if err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
		got, _, err := r.Insert(ctx, in, dbStart.Add(-48*time.Hour))
		id = got.ID
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

// The answers are read back typed, every value kind; a withdrawal clears them and counts a version.
func TestPostgres_ReadsTheAnswersAndStopClearsThem(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	id := insert(t, app, s, 0, 0)
	if _, err := owner.Exec(t.Context(), `UPDATE questionnaire_response SET answers = '[
		{"linkId": "q1", "values": [{"valueBoolean": true}]},
		{"linkId": "q2", "values": [{"valueString": "Placeholder"}, {"valueString": "Another placeholder"}]},
		{"linkId": "q3", "values": [{"valueInteger": -2147483648}]},
		{"linkId": "q4", "values": [{"valueDecimal": 12345678901234.5}]},
		{"linkId": "q5", "values": [{"valueDate": "2030-01-02"}]}]' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	yes, minInt := true, int32(-2147483648)
	want := []questionnaireresponse.Answer{
		{LinkID: "q1", Values: []questionnaireresponse.Value{{Boolean: &yes}}},
		{LinkID: "q2", Values: []questionnaireresponse.Value{{String: "Placeholder"}, {String: "Another placeholder"}}},
		{LinkID: "q3", Values: []questionnaireresponse.Value{{Integer: &minInt}}},
		{LinkID: "q4", Values: []questionnaireresponse.Value{{Decimal: json.Number("12345678901234.5")}}},
		{LinkID: "q5", Values: []questionnaireresponse.Value{{Date: "2030-01-02"}}},
	}
	at := dbStart.Add(-time.Hour)
	err := inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		got, err := r.ByID(ctx, id)
		if err != nil || !reflect.DeepEqual(got.Answers, want) {
			return errors.Join(err, errors.New("the answers differ"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var stopped questionnaireresponse.Response
	if err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
		stopped, err = r.Stop(ctx, id, at)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if stopped.Status != questionnaireresponse.StatusStopped || len(stopped.Answers) != 0 || stopped.VersionID != 2 || !stopped.LastUpdated.Equal(at) {
		t.Errorf("Stop() = %+v, want stopped at %v, no answers, version 2", stopped, at)
	}
}

// Search filters by appointment, patient and status, by due time; a malformed id matches nothing.
func TestPostgres_SearchFiltersAndOrdersByDueTime(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	later := insert(t, app, s, 1, 0)
	earlier := insert(t, app, s, 0, 1)
	if _, err := owner.Exec(t.Context(), "UPDATE questionnaire_response SET status = 'completed', authored_at = now() WHERE id = $1", later); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		f    questionnaireresponse.Filter
		want []string
	}{
		{want: []string{earlier, later}},
		{f: questionnaireresponse.Filter{AppointmentID: s.appts[1]}, want: []string{later}},
		{f: questionnaireresponse.Filter{PatientID: s.patient, Status: questionnaireresponse.StatusInProgress}, want: []string{earlier}},
		{f: questionnaireresponse.Filter{Status: questionnaireresponse.StatusStopped}},
		{f: questionnaireresponse.Filter{AppointmentID: "not-a-uuid"}},
		{f: questionnaireresponse.Filter{PatientID: "not-a-uuid"}},
	}
	for _, tt := range tests {
		var got []string
		err := inScope(t, app, s.clinic, "", func(ctx context.Context, r questionnaireresponse.Repository) error {
			all, err := r.Search(ctx, tt.f)
			for _, x := range all {
				got = append(got, x.ID)
			}
			return err
		})
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("Search(%+v) = %v, %v; want %v", tt.f, got, err, tt.want)
		}
	}
}

// The purger, on salus_retention's pool, deletes in every clinic the unsubmitted responses (in progress or
// withdrawn) whose due time is before the cut-off, to the second, and keeps completed ones and those not yet
// due.
func TestPurger_DeletesUnsubmittedResponsesOfEveryClinicOnceDue(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	cutOff := dbStart
	for _, s := range []seeded{seedDB(t, owner, dbStart.Add(-2*time.Second)), seedDB(t, owner, dbStart.Add(-time.Second))} {
		for i, status := range []string{"in-progress", "stopped", "completed"} {
			_, err := owner.Exec(t.Context(), `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id,
					practitioner_role_id, questionnaire_id, series_id, source, status, due_at, authored_at)
				SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', $3, lower(a.period),
					CASE WHEN $3 = 'completed' THEN now() END
				FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2`,
				s.appts[0], s.versions[i/2], status) // the withdrawn one is no live request of its series
			if err != nil {
				t.Fatal(err)
			}
		}
		// Not yet due: the second appointment, an hour later, in progress.
		if err := owner.QueryRow(t.Context(), `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id,
				practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
			SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', 'in-progress', $3
			FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2 RETURNING id::text`,
			s.appts[1], s.versions[0], cutOff.Add(time.Second)).Scan(new(string)); err != nil {
			t.Fatal(err)
		}
	}
	// The first clinic's responses were due 2 s before the cut-off, the second's 1 s before.
	n, err := questionnaireresponse.NewPurger(questionnaireresponse.NewPostgresRepository(ret)).Purge(t.Context(), cutOff)
	if err != nil || n != 4 {
		t.Fatalf("Purge() = %d, %v; want the 4 unsubmitted ones past due", n, err)
	}
	var left []string
	rows, err := owner.Query(t.Context(), "SELECT status FROM questionnaire_response ORDER BY status")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			t.Fatal(err)
		}
		left = append(left, status)
	}
	if want := []string{"completed", "completed", "in-progress", "in-progress"}; !slices.Equal(left, want) {
		t.Errorf("left %v, want the completed ones and the two not yet due", left)
	}
	if n, err := questionnaireresponse.NewPurger(questionnaireresponse.NewPostgresRepository(ret)).Purge(t.Context(), cutOff); err != nil || n != 0 {
		t.Errorf("Purge() again = %d, %v; want nothing", n, err)
	}
}

// In the patient's scope the answers are saved and the response completed, each a new version; neither
// touches a response that is no longer in progress, or another patient's.
func TestPostgres_SaveAnswersAndCompleteInThePatientsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	other := seedDB(t, owner, dbStart)
	id := insert(t, app, s, 0, 0)
	no, yes := false, true
	saved := []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{Boolean: &no}}}}
	kept := []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{Boolean: &yes}}}}
	at := dbStart.Add(-time.Hour)

	var got, done questionnaireresponse.Response
	err := inScope(t, app, "", other.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		if _, err := r.SaveAnswers(ctx, id, saved, at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("another patient saved"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		var err error
		if got, err = r.SaveAnswers(ctx, id, saved, at); err != nil {
			return err
		}
		if done, err = r.Complete(ctx, id, kept, at.Add(time.Minute)); err != nil {
			return err
		}
		if _, err := r.SaveAnswers(ctx, id, saved, at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("saved a completed response"))
		}
		if _, err := r.Complete(ctx, id, nil, at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("completed it twice"))
		}
		if _, err := r.SaveAnswers(ctx, "not-a-uuid", saved, at); !errors.Is(err, questionnaireresponse.ErrNotFound) {
			return errors.Join(err, errors.New("a malformed id"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Answers, saved) || got.VersionID != 2 || !got.LastUpdated.Equal(at) || got.Status != questionnaireresponse.StatusInProgress {
		t.Errorf("SaveAnswers() = %+v, want the answers at version 2, in progress", got)
	}
	if !reflect.DeepEqual(done.Answers, kept) || done.VersionID != 3 || done.Status != questionnaireresponse.StatusCompleted ||
		!done.AuthoredAt.Equal(at.Add(time.Minute)) || !done.LastUpdated.Equal(at.Add(time.Minute)) {
		t.Errorf("Complete() = %+v, want it completed with the kept answers at version 3", done)
	}
}

// A submit with no answers kept stores an empty list.
func TestPostgres_CompleteWithoutAnswers(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	id := insert(t, app, s, 0, 0)
	if err := inScope(t, app, "", s.patient, func(ctx context.Context, r questionnaireresponse.Repository) error {
		_, err := r.Complete(ctx, id, nil, dbStart.Add(-time.Hour))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := owner.QueryRow(t.Context(), "SELECT answers::text FROM questionnaire_response WHERE id = $1", id).Scan(&stored); err != nil || stored != "[]" {
		t.Errorf("stored answers %q, %v; want []", stored, err)
	}
}

// benchService is the Service on the real database as salus_app, with a response of the seeded patient in
// progress, and the patient's caller.
func benchService(b *testing.B) (*questionnaireresponse.Service, *pgxpool.Pool, string, appointment.Caller) {
	b.Helper()
	owner, app := pgtest.NewAppDatabase(b)
	s := seedDB(b, owner, dbStart)
	items, _ := large()
	id := insertB(b, app, s)
	svc := questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(app), Audit: &benchAudit{}, Tx: postgres.NewTransactor(app),
		Clock:        clock.Fixed{T: dbStart.Add(-48 * time.Hour)},
		Appointments: stubAttendance{a: appointment.Appointment{ID: s.appts[0], PatientID: s.patient, PractitionerRoleID: s.role, OrganizationID: s.clinic, Status: appointment.StatusBooked, Start: dbStart}},
		Questionnaires: stubVersion{q: questionnaire.Questionnaire{ID: s.versions[0], SeriesID: s.series[0], OrganizationID: s.clinic,
			Status: questionnaire.StatusActive, Items: items}},
	})
	return svc, owner, id, appointment.Caller{AccountID: accountA, PatientID: s.patient}
}

func insertB(b *testing.B, app *pgxpool.Pool, s seeded) string {
	b.Helper()
	in := s.request(0, 0)
	in.DueAt = dbStart
	var id string
	r := questionnaireresponse.NewPostgresRepository(app)
	if err := postgres.NewTransactor(app).WithTx(context.Background(), func(ctx context.Context) error {
		if err := r.SetClinic(ctx, s.clinic); err != nil {
			return err
		}
		got, _, err := r.Insert(ctx, in, dbStart.Add(-72*time.Hour))
		id = got.ID
		return err
	}); err != nil {
		b.Fatal(err)
	}
	return id
}

// The discard job's delete (SPEC.md §2 "Cancellation and revocation"): in the clinic's scope with the notice
// of the appointment, every response of that appointment once it is cancelled, submitted or not, also after a
// revocation; nothing of an appointment still booked, nothing without the notice, nothing of another one.
func TestPostgres_DeleteOfAppointmentDeletesEveryResponseOfTheCancelledAppointmentInTheNoticeScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner, dbStart)
	first, second, kept := insert(t, app, s, 0, 0), insert(t, app, s, 0, 1), insert(t, app, s, 1, 0)
	if _, err := owner.Exec(t.Context(), `UPDATE questionnaire_response SET status = 'completed', authored_at = now() WHERE id = $1`, second); err != nil {
		t.Fatal(err)
	}
	deleteIn := func(clinic, notice, appt string) []string {
		t.Helper()
		var ids []string
		r := questionnaireresponse.NewPostgresRepository(app)
		if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := r.SetClinic(ctx, clinic); err != nil {
				return err
			}
			if notice != "" {
				if err := r.SetNotice(ctx, notice); err != nil {
					return err
				}
			}
			var err error
			ids, err = r.DeleteOfAppointment(ctx, appt)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return ids
	}

	if ids := deleteIn(s.clinic, s.appts[0], s.appts[0]); len(ids) != 0 {
		t.Errorf("deleted %v of a booked appointment, want none", ids)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'pat' WHERE id = $1`, s.appts[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now()`); err != nil {
		t.Fatal(err)
	}
	if ids := deleteIn(s.clinic, "", s.appts[0]); len(ids) != 0 {
		t.Errorf("deleted %v without the notice, want none", ids)
	}
	if ids := deleteIn(s.clinic, s.appts[1], s.appts[0]); len(ids) != 0 {
		t.Errorf("deleted %v with the notice of another appointment, want none", ids)
	}
	if ids := deleteIn(s.clinic, s.appts[0], "not-a-uuid"); len(ids) != 0 {
		t.Errorf("deleted %v for an id that is not a UUID, want none", ids)
	}
	ids := deleteIn(s.clinic, s.appts[0], s.appts[0])
	slices.Sort(ids)
	want := []string{first, second}
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("deleted %v, want both responses of the cancelled appointment %v", ids, want)
	}
	var left []string
	rows, err := owner.Query(t.Context(), "SELECT id::text FROM questionnaire_response ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		left = append(left, id)
	}
	if !slices.Equal(left, []string{kept}) {
		t.Errorf("left %v, want only the other appointment's %s", left, kept)
	}
}
