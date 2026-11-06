package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// responseRows is the seeded questionnaire responses: per appointment of seedAppointments a completed one
// and one in progress, of two active versions of the appointment's clinic (versions), and a draft per clinic
// that no response references.
type responseRows struct {
	completed, open map[string]string // by appointment
	versions        map[string][]string
	draft           map[string]string
	roles           map[string]string // each clinic's practitioner role
}

// seedResponses creates, as the owner, the rows of seedAppointments, two active questionnaires and a draft per
// clinic, and for every appointment a completed and an in-progress response, each with a synthetic answer.
// due_at is the appointment's start, as the service copies it.
func seedResponses(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, appointmentRows, responseRows) {
	t.Helper()
	ctx := t.Context()
	p, dir, a := seedAppointments(t, owner)
	r := responseRows{completed: map[string]string{}, open: map[string]string{}, versions: map[string][]string{}, draft: map[string]string{},
		roles: map[string]string{clinicA: dir[clinicA].role, clinicB: dir[clinicB].role}}
	for _, org := range []string{clinicA, clinicB} {
		for _, status := range []string{"active", "active", "draft"} {
			var id string
			if err := owner.QueryRow(ctx, `INSERT INTO questionnaire (organization_id, series_id, version, status, title, published_at)
				VALUES ($1, gen_random_uuid(), 1, $2, 'Placeholder questionnaire', CASE WHEN $2 = 'active' THEN now() END)
				RETURNING id`, org, status).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if status == "draft" {
				r.draft[org] = id
			} else {
				r.versions[org] = append(r.versions[org], id)
			}
		}
	}
	for appt, org := range map[string]string{a.aPast: clinicA, a.aFuture: clinicA, a.bPast: clinicB, a.bFuture: clinicB, a.other: clinicA} {
		for i, status := range []string{"completed", "in-progress"} {
			var id string
			err := owner.QueryRow(ctx, `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id,
					practitioner_role_id, questionnaire_id, series_id, source, status, answers, due_at, authored_at)
				SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', $3,
					'[{"linkId": "q1", "values": [{"valueString": "Placeholder answer"}]}]', lower(a.period),
					CASE WHEN $3 = 'completed' THEN lower(a.period) - interval '1 hour' END
				FROM appointment a, questionnaire q
				WHERE a.id = $1 AND q.id = $2
				RETURNING id`, appt, r.versions[org][i], status).Scan(&id)
			if err != nil {
				t.Fatal(err)
			}
			if status == "completed" {
				r.completed[appt] = id
			} else {
				r.open[appt] = id
			}
		}
	}
	return p, a, r
}

// both is the completed and the open response of each appointment, sorted.
func both(r responseRows, appts ...string) []string {
	var out []string
	for _, a := range appts {
		out = append(out, r.completed[a], r.open[a])
	}
	return sorted(out...)
}

// exec runs stmt in the scope and returns the rows it affected.
func exec(t *testing.T, app *pgxpool.Pool, scope map[string]string, stmt string, args ...any) (int64, error) {
	t.Helper()
	var n int64
	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, stmt, args...)
		n = tag.RowsAffected()
		return err
	})
	return n, err
}

// Row-level security (SPEC.md §2 "Tables"): the patient their own; a clinic those of its patients with an
// active consent, and after a revocation the completed ones of appointments that started before it (D9); the
// notice scope the one appointment's of its clinic, whatever the consent; nothing else.
func TestQuestionnaireResponse_EachScopeSeesOnlyItsResponses(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: both(r, a.aPast, a.aFuture, a.bPast, a.bFuture)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: both(r, a.other)},
		{name: "patient B with clinic A set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicA}, want: both(r, a.other)},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: both(r, a.aPast, a.aFuture)},
		{name: "clinic B after the revocation: the past appointment's completed one only", scope: map[string]string{"app.organization_id": clinicB},
			want: sorted(r.completed[a.bPast])},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
		// The notice scope sets the clinic too, so clinic B's own view (bPast's completed one) shows beside it.
		{name: "notice: clinic B's cancelled appointment", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture},
			want: sorted(r.completed[a.bFuture], r.open[a.bFuture], r.completed[a.bPast])},
		{name: "notice: an appointment of another clinic", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.aFuture},
			want: sorted(r.completed[a.bPast])},
		{name: "an appointment alone", scope: map[string]string{"app.appointment_id": a.bFuture}},
		{name: "notice with an account set", scope: map[string]string{"app.account_id": p[1].account, "app.organization_id": clinicB, "app.appointment_id": a.bFuture}},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM questionnaire_response"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A revocation ends the clinic's view of the open responses when it commits, and of every response of a
// later appointment.
func TestQuestionnaireResponse_TheRevocationEndsTheClinicsViewWhenItCommits(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = '2026-11-01T08:00:00Z'
		WHERE patient_id = $1 AND organization_id = $2`, p[0].patient, clinicA); err != nil {
		t.Fatal(err)
	}
	// aPast started 2026-10-29, before the revocation; aFuture is after it.
	if got, want := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, "SELECT id::text FROM questionnaire_response"),
		sorted(r.completed[a.aPast]); !slices.Equal(got, want) {
		t.Errorf("clinic A after the revocation sees %v, want %v", got, want)
	}
}

// The patient scope sees the questionnaire versions its own responses reference, and no other; the clinic
// scope still sees its own library only.
func TestQuestionnaireResponse_APatientSeesTheVersionsTheirResponsesReference(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, r := seedResponses(t, owner)
	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(append(slices.Clone(r.versions[clinicA]), r.versions[clinicB]...)...)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(r.versions[clinicA]...)},
		{name: "account A", scope: map[string]string{"app.account_id": p[0].account}},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(append(slices.Clone(r.versions[clinicB]), r.draft[clinicB])...)},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM questionnaire"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees versions %v, want %v", tt.name, got, tt.want)
		}
	}
	for _, stmt := range []string{"UPDATE questionnaire SET title = 'x'", "DELETE FROM questionnaire"} {
		if n, err := exec(t, app, map[string]string{"app.patient_id": p[0].patient}, stmt); err != nil || n != 0 {
			t.Errorf("%s as patient A: %d rows, %v; want none", stmt, n, err)
		}
	}
}

// A clinic requests a questionnaire only for a patient with an active consent, in its own clinic; a patient
// never inserts one.
func TestQuestionnaireResponse_OnlyAClinicWithAnActiveConsentInserts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	if _, err := owner.Exec(t.Context(), "DELETE FROM questionnaire_response"); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
			series_id, source, status, due_at)
		VALUES ($1, $2, $3, $4, $5, gen_random_uuid(), 'added', 'in-progress', now())`
	tests := []struct {
		scope         map[string]string
		name          string
		org, appt     string
		patient, want string
	}{
		{name: "clinic A, active consent", scope: map[string]string{"app.organization_id": clinicA}, org: clinicA, appt: a.aFuture, patient: p[0].patient},
		{name: "clinic A, a patient without a consent", scope: map[string]string{"app.organization_id": clinicA}, org: clinicA, appt: a.other, patient: p[1].patient, want: "42501"},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB}, org: clinicB, appt: a.bPast, patient: p[0].patient, want: "42501"},
		{name: "clinic A, for clinic B", scope: map[string]string{"app.organization_id": clinicA}, org: clinicB, appt: a.bPast, patient: p[0].patient, want: "42501"},
		{name: "patient A, their own", scope: map[string]string{"app.patient_id": p[0].patient}, org: clinicA, appt: a.aPast, patient: p[0].patient, want: "42501"},
	}
	for _, tt := range tests {
		_, err := exec(t, app, tt.scope, insert, tt.org, tt.appt, tt.patient, r.roles[tt.org], r.versions[tt.org][0])
		if sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %q", tt.name, err, tt.want)
		}
	}
}

// Updates: the clinic its responses of patients with an active consent (a withdrawal), the patient their own
// (the answers); nobody else, and never a response of another scope.
func TestQuestionnaireResponse_EachScopeUpdatesOnlyWhatItMay(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	update := "UPDATE questionnaire_response SET updated_at = now(), version_id = version_id + 1 WHERE id = ANY($1::uuid[])"
	tests := []struct {
		scope map[string]string
		name  string
		ids   []string
		want  int64
	}{
		{name: "clinic A, its consenting patient's", scope: map[string]string{"app.organization_id": clinicA}, ids: both(r, a.aFuture), want: 2},
		{name: "clinic A, a patient without a consent", scope: map[string]string{"app.organization_id": clinicA}, ids: both(r, a.other)},
		{name: "clinic B, a past completed one after the revocation", scope: map[string]string{"app.organization_id": clinicB}, ids: both(r, a.bPast)},
		{name: "patient A, their own", scope: map[string]string{"app.patient_id": p[0].patient}, ids: both(r, a.aFuture, a.bFuture), want: 4},
		{name: "patient B, patient A's", scope: map[string]string{"app.patient_id": p[1].patient}, ids: both(r, a.aFuture)},
		{name: "notice scope", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture}, ids: both(r, a.bFuture)},
	}
	for _, tt := range tests {
		if n, err := exec(t, app, tt.scope, update, tt.ids); err != nil || n != tt.want {
			t.Errorf("%s: %d rows, %v; want %d", tt.name, n, err, tt.want)
		}
	}
}

// Deletes: only the discard scope (app.appointment_id with app.organization_id), and only the responses of
// that one appointment when it is cancelled. The clinic and the patient delete nothing.
func TestQuestionnaireResponse_OnlyTheDiscardScopeDeletesAndOnlyOfACancelledAppointment(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, _ := seedResponses(t, owner)
	tests := []struct {
		scope map[string]string
		name  string
		want  int64
	}{
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}},
		{name: "a booked appointment", scope: map[string]string{"app.organization_id": clinicA, "app.appointment_id": a.aFuture}},
		{name: "a cancelled appointment named with another clinic", scope: map[string]string{"app.organization_id": clinicA, "app.appointment_id": a.bFuture}},
		{name: "a cancelled appointment with a patient set", scope: map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicB, "app.appointment_id": a.bFuture}},
		{name: "the cancelled appointment", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture}, want: 2},
	}
	for _, tt := range tests {
		if n, err := exec(t, app, tt.scope, "DELETE FROM questionnaire_response"); err != nil || n != tt.want {
			t.Errorf("%s: deleted %d, %v; want %d", tt.name, n, err, tt.want)
		}
	}
	var left int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM questionnaire_response").Scan(&left); err != nil || left != 8 {
		t.Errorf("%d responses left, %v; want 8", left, err)
	}
}

func TestQuestionnaireResponse_TheAppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, r := seedResponses(t, owner)
	scope := map[string]string{"app.organization_id": clinicA}
	for _, stmt := range []string{
		"TRUNCATE questionnaire_response",
		"UPDATE questionnaire_response SET organization_id = organization_id", "UPDATE questionnaire_response SET appointment_id = appointment_id",
		"UPDATE questionnaire_response SET patient_id = patient_id", "UPDATE questionnaire_response SET practitioner_role_id = practitioner_role_id",
		"UPDATE questionnaire_response SET questionnaire_id = questionnaire_id", "UPDATE questionnaire_response SET series_id = series_id",
		"UPDATE questionnaire_response SET source = source", "UPDATE questionnaire_response SET due_at = due_at",
		"UPDATE questionnaire_response SET created_at = now()",
	} {
		if _, err := exec(t, app, scope, stmt); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	allowed := "UPDATE questionnaire_response SET status = 'stopped', answers = '[]', authored_at = NULL, updated_at = now(), version_id = version_id + 1 WHERE id = $1"
	for _, id := range r.open {
		if _, err := exec(t, app, scope, allowed, id); err != nil {
			t.Errorf("withdraw: %v", err)
		}
	}
}

// One live request per questionnaire series per appointment (a withdrawn one may be asked again); the
// statuses and sources; a submit time exactly when completed; no answers once withdrawn; the version of the
// response's clinic.
func TestQuestionnaireResponse_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, a, r := seedResponses(t, owner)
	ctx := t.Context()
	insert := func(appt, version, status, extra string) error {
		_, err := owner.Exec(ctx, `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id,
				questionnaire_id, series_id, source, status, due_at)
			SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', $3, lower(a.period)
			FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2`+extra, appt, version, status)
		return err
	}
	if err := insert(a.aFuture, r.versions[clinicA][0], "in-progress", ""); sqlState(err) != "23505" {
		t.Errorf("a second live request of a series: %v, want unique_violation", err)
	}
	if _, err := owner.Exec(ctx, "UPDATE questionnaire_response SET status = 'stopped', answers = '[]', authored_at = NULL WHERE id = $1", r.completed[a.aFuture]); err != nil {
		t.Fatal(err)
	}
	if err := insert(a.aFuture, r.versions[clinicA][0], "in-progress", ""); err != nil {
		t.Errorf("a request after a withdrawal: %v", err)
	}
	if err := insert(a.bPast, r.versions[clinicA][0], "in-progress", ""); sqlState(err) != "23503" {
		t.Errorf("a version of another clinic: %v, want foreign_key_violation", err)
	}
	for stmt, want := range map[string]string{
		"UPDATE questionnaire_response SET status = 'amended'":                               "23514",
		"UPDATE questionnaire_response SET source = 'patient'":                               "23514",
		"UPDATE questionnaire_response SET authored_at = NULL WHERE status = 'completed'":    "23514",
		"UPDATE questionnaire_response SET authored_at = now() WHERE status = 'in-progress'": "23514",
		"UPDATE questionnaire_response SET status = 'stopped' WHERE status = 'in-progress'":  "23514",
		"UPDATE questionnaire_response SET answers = '{}'":                                   "23514",
		"UPDATE questionnaire_response SET organization_id = '" + clinicB + "' WHERE TRUE":   "23503",
	} {
		if _, err := owner.Exec(ctx, stmt); sqlState(err) != want {
			t.Errorf("%s: %v, want SQLSTATE %s", stmt, err, want)
		}
	}
}

// The retention role deletes the unsubmitted responses past their due time in every clinic, as the purger
// asks, and cannot insert or update one.
func TestQuestionnaireResponse_TheRetentionRoleDeletesOnlyAndPastRowLevelSecurity(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	_, a, _ := seedResponses(t, owner)
	tag, err := ret.Exec(t.Context(), "DELETE FROM questionnaire_response WHERE status <> 'completed' AND due_at < $1", "2027-01-01T00:00:00Z")
	if err != nil || tag.RowsAffected() != 2 { // aPast's and bPast's open ones
		t.Errorf("purge: %d rows, %v; want 2", tag.RowsAffected(), err)
	}
	for _, stmt := range []string{
		"UPDATE questionnaire_response SET answers = '[]'",
		"INSERT INTO questionnaire_response SELECT * FROM questionnaire_response",
		"TRUNCATE questionnaire_response",
		"SELECT count(*) FROM questionnaire",
		"SELECT count(*) FROM appointment WHERE id = '" + a.aPast + "'",
	} {
		if _, err := ret.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.RetentionRole, err)
		}
	}
}
