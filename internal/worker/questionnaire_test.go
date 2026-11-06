package worker_test

import (
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// placeholderTitle is the synthetic questionnaire's title, which no log line may hold.
const placeholderTitle = "Placeholder questionnaire about a condition"

// seedQuestionnaire creates, as the owner, an active questionnaire of the clinic of seedBooking with one
// question, and makes it the default of the booking's role; it returns the version's id.
func seedQuestionnaire(t *testing.T, db logins, b booking) string {
	t.Helper()
	var id string
	err := db.owner.QueryRow(t.Context(), `WITH q AS (INSERT INTO questionnaire (organization_id, series_id, version, status, title, items, published_at)
			VALUES ($1, gen_random_uuid(), 1, 'active', $3, '[{"linkId": "q1", "text": "Placeholder question", "type": "boolean"}]', now())
			RETURNING id, series_id, organization_id),
		d AS (INSERT INTO questionnaire_default (practitioner_role_id, organization_id, series_id)
			SELECT a.practitioner_role_id, q.organization_id, q.series_id FROM q, appointment a WHERE a.id = $2)
		SELECT id::text FROM q`, clinicID, b.future, placeholderTitle).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// checkQuestionnaireLog fails when the job's log holds a name, the e-mail or the title, or lacks the
// appointment's id.
func checkQuestionnaireLog(t *testing.T, logs string, b booking, appointmentID string) {
	t.Helper()
	for _, v := range []string{b.email, "Ana", "Cy", placeholderTitle, "Placeholder question"} {
		if strings.Contains(logs, v) {
			t.Errorf("the log holds %q:\n%s", v, logs)
		}
	}
	if !strings.Contains(logs, `"appointment_id":"`+appointmentID+`"`) {
		t.Errorf("the log lacks the appointment's id:\n%s", logs)
	}
}

// The booking's job requests the role's default once, in progress, due at the start, audited by the job.
func TestQuestionnaireDefaultRequest_RequestsTheRolesDefaultOnce(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	version := seedQuestionnaire(t, db, b)
	args := appointment.QuestionnaireRequestArgs{AppointmentID: b.future, OrganizationID: clinicID}

	for run := range 2 { // the second run is River's retry: one row
		ended, logs := runWorkerJob(t, db, args)
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
		checkQuestionnaireLog(t, logs, b, b.future)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM questionnaire_response r JOIN appointment a ON a.id = r.appointment_id
		WHERE r.appointment_id = $1 AND r.questionnaire_id = $2 AND r.source = 'default' AND r.status = 'in-progress'
		AND r.due_at = lower(a.period) AND r.patient_id = a.patient_id AND r.answers = '[]'`, b.future, version); n != 1 {
		t.Errorf("%d default requests of the appointment, want 1", n)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM questionnaire_response`); n != 1 {
		t.Errorf("%d responses in all, want 1", n)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'questionnaire_default_request' AND purpose_code = 'HOPERAT'
		AND subtype_code = 'create' AND entity_reference LIKE 'QuestionnaireResponse/%'`); n != 1 {
		t.Errorf("%d audit events of the request by the job, want 1", n)
	}
	// The past appointment has started: nothing is requested for it.
	ended, logs := runWorkerJob(t, db, appointment.QuestionnaireRequestArgs{AppointmentID: b.past, OrganizationID: clinicID})
	if n := countWhere(t, db, `SELECT count(*) FROM questionnaire_response WHERE appointment_id = $1`, b.past); ended != river.EventKindJobCompleted || n != 0 {
		t.Errorf("the past appointment: the job ended %s with %d requests, want completed with none:\n%s", ended, n, logs)
	}
}

// The revocation cancels the future appointment and enqueues its discard; the discard deletes the future
// appointment's responses, submitted or not, and keeps the past one's completed response (D9). A discard of the
// past appointment, which is not cancelled, deletes nothing; repeated, nothing changes.
func TestQuestionnaireResponsesDiscard_AfterARevocationDeletesTheFutureAppointmentsResponsesOnly(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	version := seedQuestionnaire(t, db, b)
	if _, err := db.owner.Exec(t.Context(), `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id,
			questionnaire_id, series_id, source, status, due_at, authored_at, answers)
		SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', s.status, lower(a.period),
			CASE WHEN s.status = 'completed' THEN lower(a.period) - interval '1 hour' END, '[{"linkId": "q1", "values": [{"valueBoolean": true}]}]'
		FROM appointment a, questionnaire q, (VALUES ($1::uuid, 'in-progress'), ($2::uuid, 'completed')) AS s (appointment_id, status)
		WHERE a.id = s.appointment_id AND q.id = $3`, b.future, b.past, version); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(t.Context(), "UPDATE consent SET status = 'inactive', revoked_at = now() - interval '1 minute'"); err != nil {
		t.Fatal(err)
	}
	if ended, logs := runWorkerJob(t, db, consent.RevokedArgs{PatientID: b.patient, OrganizationID: clinicID}); ended != river.EventKindJobCompleted {
		t.Fatalf("the revocation ended %s, want completed:\n%s", ended, logs)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM river_job WHERE kind = 'questionnaire_responses_discard' AND queue = 'default'
		AND args::text = $1`, `{"appointment_id": "`+b.future+`", "organization_id": "`+clinicID+`"}`); n != 1 {
		t.Errorf("%d discard jobs of the cancelled appointment enqueued by the revocation, want 1", n)
	}

	for _, appt := range []string{b.future, b.future, b.past} {
		ended, logs := runWorkerJob(t, db, appointment.QuestionnaireDiscardArgs{AppointmentID: appt, OrganizationID: clinicID})
		if ended != river.EventKindJobCompleted {
			t.Fatalf("the discard of %s ended %s, want completed:\n%s", appt, ended, logs)
		}
		checkQuestionnaireLog(t, logs, b, appt)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM questionnaire_response WHERE appointment_id = $1`, b.future); n != 0 {
		t.Errorf("%d responses of the cancelled appointment left, want none", n)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM questionnaire_response WHERE appointment_id = $1 AND status = 'completed'`, b.past); n != 1 {
		t.Errorf("%d completed responses of the past appointment, want it kept", n)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'questionnaire_responses_discard' AND purpose_code = 'HOPERAT'
		AND subtype_code = 'delete' AND entity_reference LIKE 'QuestionnaireResponse/%'`); n != 1 {
		t.Errorf("%d audit events of the discard by the job, want 1", n)
	}
}

// e1 is COPY.md E1.
const e1 = "There is a questionnaire to fill in before your appointment. You can find it in the Salus app."

// The confirmation and the reminder carry E1 while a questionnaire is outstanding, read in the notifier's
// notice scope as salus_app: the role's default before its request, a response in progress; not once it is
// submitted, nor without a default. Never the title or a question.
func TestNotifier_MentionsAnOutstandingQuestionnaireWithoutItsTitle(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	version := seedQuestionnaire(t, db, b)
	setResponse := func(status string) {
		t.Helper()
		if _, err := db.owner.Exec(t.Context(), `DELETE FROM questionnaire_response;
			INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id,
				source, status, due_at, authored_at)
			SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'default', '`+status+`', lower(a.period),
				CASE WHEN '`+status+`' = 'completed' THEN now() END
			FROM appointment a, questionnaire q WHERE a.id = '`+b.future+`' AND q.id = '`+version+`'`); err != nil {
			t.Fatal(err)
		}
	}
	booked := appointment.BookedEmailArgs{AppointmentID: b.future, OrganizationID: clinicID}
	reminder := appointment.ReminderEmailArgs{AppointmentID: b.future, OrganizationID: clinicID}
	for _, tt := range []struct {
		setup func()
		args  river.JobArgs
		name  string
		want  bool
	}{
		{name: "the confirmation before the default's request", args: booked, want: true, setup: func() {}},
		{name: "the reminder with the default in progress", args: reminder, want: true, setup: func() { setResponse("in-progress") }},
		{name: "the reminder with the default submitted", args: reminder, want: false, setup: func() { setResponse("completed") }},
		{name: "the confirmation without a default", args: booked, want: false, setup: func() {
			if _, err := db.owner.Exec(t.Context(), "DELETE FROM questionnaire_response; DELETE FROM questionnaire_default"); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		tt.setup()
		if state := runEmailJob(t, db.app, tt.args); state != rivertype.JobStateCompleted {
			t.Fatalf("%s: job state %s, want completed", tt.name, state)
		}
		text := mailpitText(t, b.email)
		if strings.Contains(text, e1) != tt.want {
			t.Errorf("%s: E1 in the e-mail is %v, want %v:\n%s", tt.name, !tt.want, tt.want, text)
		}
		for _, never := range []string{placeholderTitle, "Placeholder question", version} {
			if strings.Contains(text, never) {
				t.Errorf("%s: the e-mail holds %q", tt.name, never)
			}
		}
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name IN ('appointment_booked_email', 'appointment_reminder_email')
		AND purpose_code = 'HOPERAT' AND subtype_code = 'search-type'`); n != 4 {
		t.Errorf("%d audit events of the jobs' checks, want 4: one per e-mail", n)
	}
}
