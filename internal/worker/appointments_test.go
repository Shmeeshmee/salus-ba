package worker_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// booking is the rows seedBooking created.
type booking struct {
	email, patient, future, futureSlot, past, pastSlot string
}

// seedBooking creates, as the owner, the listed clinic with a practitioner, a schedule and two busy slots, a
// patient (Ana Example, a fresh synthetic address) with an active consent and an active transcription consent
// (SAL-12), and their two appointments: one a week ahead and one that started an hour ago.
func seedBooking(t *testing.T, owner *pgxpool.Pool) booking {
	t.Helper()
	ctx := t.Context()
	b := booking{email: "patient-" + strings.ToLower(rand.Text()) + "@example.test"}
	var role, sched string
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id),
		s AS (INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) SELECT organization_id, id, 30 FROM r RETURNING id, practitioner_role_id)
		SELECT practitioner_role_id::text, id::text FROM s`, clinicID).Scan(&role, &sched)
	if err != nil {
		t.Fatal(err)
	}
	// The patient chose English, so the e-mails these tests read are the English ones (docs/i18n SPEC §3).
	if err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role, language) VALUES ($1, 'h', 'patient', 'en') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ana', 'Example', '1990-01-01' FROM a RETURNING id)
		INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $2, 'active', now() - interval '1 day' FROM p
		RETURNING patient_id::text`, b.email, clinicID).Scan(&b.patient); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO transcription_consent (patient_id, organization_id, status, date_time)
		VALUES ($1, $2, 'active', now() - interval '1 day')`, b.patient, clinicID); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		id, slot *string
		at       string
	}{{&b.future, &b.futureSlot, "7 days"}, {&b.past, &b.pastSlot, "-1 hour"}} {
		if err := owner.QueryRow(ctx, `WITH s AS (INSERT INTO slot (schedule_id, organization_id, period, status)
				VALUES ($1, $2, tstzrange(date_trunc('minute', now()) + $4::interval, date_trunc('minute', now()) + $4::interval + interval '30 minutes'), 'busy')
				RETURNING id, period)
			INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
			SELECT $3, $5, $2, id, period, now() - interval '1 day' FROM s RETURNING id::text, slot_id::text`,
			sched, clinicID, b.patient, x.at, role).Scan(x.id, x.slot); err != nil {
			t.Fatal(err)
		}
	}
	return b
}

func TestConsentRevoked_CancelsTheFutureAppointmentsFreesTheirSlotsAndEnqueuesTheirEmailsOnce(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	if _, err := db.owner.Exec(t.Context(), "UPDATE consent SET status = 'inactive', revoked_at = now() - interval '1 minute'"); err != nil {
		t.Fatal(err)
	}
	args := consent.RevokedArgs{PatientID: b.patient, OrganizationID: clinicID}

	for run := range 2 { // the second run is River's retry: it finds nothing left
		ended, logs := runWorkerJob(t, db, args)
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
		if !strings.Contains(logs, `"cancelled":`+map[int]string{0: "1", 1: "0"}[run]) {
			t.Errorf("run %d: the log lacks the count:\n%s", run, logs)
		}
		for _, v := range []string{b.email, "Ana", "Example"} {
			if strings.Contains(logs, v) {
				t.Errorf("run %d: the log holds %q:\n%s", run, v, logs)
			}
		}
	}
	var future, past, futureSlot, pastSlot string
	if err := db.owner.QueryRow(t.Context(), `SELECT
		(SELECT status || '/' || cancellation_reason FROM appointment WHERE id = $1),
		(SELECT status FROM appointment WHERE id = $2),
		(SELECT status FROM slot WHERE id = $3), (SELECT status FROM slot WHERE id = $4)`, b.future, b.past, b.futureSlot, b.pastSlot).
		Scan(&future, &past, &futureSlot, &pastSlot); err != nil {
		t.Fatal(err)
	}
	if future != "cancelled/pat" || futureSlot != "free" || past != "booked" || pastSlot != "busy" {
		t.Errorf("the future appointment %s with its slot %s, the past one %s with %s; want the future one cancelled (pat) and freed, the past one untouched",
			future, futureSlot, past, pastSlot)
	}
	var jobs int
	var jobArgs string
	if err := db.owner.QueryRow(t.Context(), `SELECT count(*), min(args::text) FROM river_job WHERE kind = 'appointment_cancelled_email' AND queue = 'email'`).
		Scan(&jobs, &jobArgs); err != nil {
		t.Fatal(err)
	}
	if want := `{"appointment_id": "` + b.future + `", "organization_id": "` + clinicID + `"}`; jobs != 1 || jobArgs != want {
		t.Errorf("%d cancellation e-mails with %s, want one with %s", jobs, jobArgs, want)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'consent_revoked' AND purpose_code = 'HOPERAT'
		AND subtype_code = 'update' AND entity_reference = $1`, "Appointment/"+b.future); n != 1 {
		t.Errorf("%d audit events of the cancellation by the job, want 1", n)
	}
}

// runEmailJob starts the notifier, enqueues args on the email queue, and returns the job's final state.
func runEmailJob(t *testing.T, app *pgxpool.Pool, args river.JobArgs) rivertype.JobState {
	t.Helper()
	client, err := worker.NewNotifier(app, slog.New(slog.DiscardHandler), smtpAddr(t), "https://app.example.test", "https://patient.example.test")
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}
	events, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobCancelled, river.EventKindJobFailed)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(context.WithoutCancel(t.Context())) }()
	if _, err := client.Insert(t.Context(), args, &river.InsertOpts{Queue: "email"}); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		return ev.Job.State
	case <-time.After(15 * time.Second):
		t.Fatalf("the %s job did not finish within 15s", args.Kind())
	}
	return ""
}

func TestNotifier_SendsTheAppointmentEmailsToThePatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	b := seedBooking(t, owner)

	for _, args := range []river.JobArgs{
		appointment.BookedEmailArgs{AppointmentID: b.future, OrganizationID: clinicID},
		appointment.ReminderEmailArgs{AppointmentID: b.future, OrganizationID: clinicID},
	} {
		if state := runEmailJob(t, app, args); state != rivertype.JobStateCompleted {
			t.Fatalf("%s: job state %s, want completed", args.Kind(), state)
		}
		text := mailpitText(t, b.email)
		if !strings.Contains(text, "Synthetic clinic") || !strings.Contains(text, "(Europe/Sarajevo)") || !strings.Contains(text, "https://patient.example.test/appointments") {
			t.Errorf("%s: the e-mail lacks the clinic, the zone or the link:\n%s", args.Kind(), text)
		}
		for _, never := range []string{"Ana", "Cy", b.future} {
			if strings.Contains(text, never) {
				t.Errorf("%s: the e-mail holds %q:\n%s", args.Kind(), never, text)
			}
		}
	}
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name IN ('appointment_booked_email', 'appointment_reminder_email')
		AND purpose_code = 'HOPERAT' AND subtype_code = 'read'`).Scan(&n); err != nil || n != 8 {
		t.Errorf("%d audit events of the jobs' reads (%v), want 8: the appointment, the patient, the account and the "+
			"role's default (the questionnaire check, docs/questionnaire package 7), twice", n, err)
	}
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name IN ('appointment_booked_email', 'appointment_reminder_email')
		AND subtype_code = 'read' AND entity_reference LIKE 'PractitionerRole/%'`).Scan(&n); err != nil || n != 2 {
		t.Errorf("%d audit events of the role's default read by the jobs (%v), want 2", n, err)
	}
}

// After a revocation the clinic's scope no longer shows the cancelled appointment; the job's does, and the
// patient gets the cancellation. A reminder of it then sends nothing, and an unknown appointment cancels the
// job.
func TestNotifier_MailsACancellationAfterARevocationAndNothingThatNoLongerApplies(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	b := seedBooking(t, owner)
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() - interval '1 minute';
		UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'pat' WHERE id = '`+b.future+`'`); err != nil {
		t.Fatal(err)
	}
	if state := runEmailJob(t, app, appointment.CancelledEmailArgs{AppointmentID: b.future, OrganizationID: clinicID}); state != rivertype.JobStateCompleted {
		t.Fatalf("cancellation: job state %s, want completed", state)
	}
	if text := mailpitText(t, b.email); !strings.Contains(text, "is cancelled") {
		t.Errorf("the cancellation e-mail:\n%s", text)
	}
	if state := runEmailJob(t, app, appointment.ReminderEmailArgs{AppointmentID: b.future, OrganizationID: clinicID}); state != rivertype.JobStateCompleted {
		t.Errorf("a reminder of a cancelled appointment: job state %s, want completed", state)
	}
	if n := mailpitCount(t, b.email); n != 0 {
		t.Errorf("%d e-mails after the reminder of a cancelled appointment, want none", n)
	}
	unknown := appointment.BookedEmailArgs{AppointmentID: "e9000000-0000-4000-8000-0000000000e9", OrganizationID: clinicID}
	if state := runEmailJob(t, app, unknown); state != rivertype.JobStateCancelled {
		t.Errorf("an unknown appointment: job state %s, want cancelled", state)
	}
}

// mailpitCount returns how many messages to address the compose Mailpit holds.
func mailpitCount(t *testing.T, address string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		os.Getenv("TEST_MAILPIT_URL")+"/api/v1/search?query="+url.QueryEscape(`to:"`+address+`"`), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var found struct{ Messages []struct{ ID string } }
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		t.Fatal(err)
	}
	return len(found.Messages)
}
