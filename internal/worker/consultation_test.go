package worker_test

import (
	"context"
	"crypto/rand"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

// liveKit is the provider on the compose LiveKit (TEST_LIVEKIT_*), which the worker is built with; without it
// the tests fail.
func liveKit(t *testing.T) videosession.Provider {
	t.Helper()
	url, key, secret := os.Getenv("TEST_LIVEKIT_URL"), os.Getenv("TEST_LIVEKIT_API_KEY"), os.Getenv("TEST_LIVEKIT_API_SECRET")
	if url == "" || key == "" || secret == "" {
		t.Fatal("TEST_LIVEKIT_URL, TEST_LIVEKIT_API_KEY and TEST_LIVEKIT_API_SECRET are not set; run the tests with `make test`")
	}
	return videosession.NewLiveKitProvider(url, key, secret)
}

// consultationRows is a video session of the past appointment of seedBooking whose window closed a minute
// ago, with random names.
type consultationRows struct {
	session, room, patientIdentity, practitionerIdentity string
}

// seedConsultation opens, as the owner, the session of the past appointment; the practitioner joined (and the
// encounter is in progress) when practitionerJoined, the patient when patientJoined.
func seedConsultation(t *testing.T, db logins, b booking, practitionerJoined, patientJoined bool) consultationRows {
	t.Helper()
	random := func(prefix string) string { return prefix + strings.ToLower(rand.Text()) }
	c := consultationRows{room: random("r_"), patientIdentity: random("p_"), practitionerIdentity: random("p_")}
	err := db.owner.QueryRow(t.Context(), `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity,
			practitioner_identity, closes_at, practitioner_joined_at, patient_joined_at)
		SELECT id, organization_id, $2, $3, $4, now() - interval '1 minute',
			CASE WHEN $5 THEN lower(period) END, CASE WHEN $6 THEN lower(period) END
		FROM appointment WHERE id = $1 RETURNING id::text`,
		b.past, c.room, c.patientIdentity, c.practitionerIdentity, practitionerJoined, patientJoined).Scan(&c.session)
	if err != nil {
		t.Fatal(err)
	}
	if practitionerJoined {
		if _, err := db.owner.Exec(t.Context(), `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM appointment WHERE id = $1`, b.past); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// checkConsultationLog fails when the job's log holds a name, an e-mail, the room or an identity.
func checkConsultationLog(t *testing.T, logs string, b booking, c consultationRows) {
	t.Helper()
	for _, v := range []string{b.email, "Ana", "Cy", c.room, c.patientIdentity, c.practitionerIdentity} {
		if strings.Contains(logs, v) {
			t.Errorf("the log holds %q:\n%s", v, logs)
		}
	}
	if !strings.Contains(logs, `"session_id":"`+c.session+`"`) {
		t.Errorf("the log lacks the session id:\n%s", logs)
	}
}

// The window closed with the room empty: the job ends the session (window_closed), finishes the encounter
// and marks the appointment fulfilled (the patient joined), audited by the job; a retry changes nothing.
func TestConsultationWindowClose_EndsTheSessionWithTheRoomEmpty(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, true, true)
	args := videosession.WindowCloseArgs{SessionID: c.session, OrganizationID: clinicID}

	for run := range 2 {
		ended, logs := runWorkerJob(t, db, args)
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
		checkConsultationLog(t, logs, b, c)
	}
	var reason, encounter, appt string
	if err := db.owner.QueryRow(t.Context(), `SELECT
		(SELECT end_reason FROM video_session WHERE id = $1 AND ended_at IS NOT NULL),
		(SELECT status FROM encounter WHERE appointment_id = $2 AND period_end IS NOT NULL),
		(SELECT status FROM appointment WHERE id = $2)`, c.session, b.past).Scan(&reason, &encounter, &appt); err != nil {
		t.Fatal(err)
	}
	if reason != "window_closed" || encounter != "finished" || appt != "fulfilled" {
		t.Errorf("session %s, encounter %s, appointment %s; want window_closed, finished, fulfilled", reason, encounter, appt)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'consultation_window_close' AND purpose_code = 'HOPERAT'
		AND organization_id = $1`, clinicID); n != 3 {
		t.Errorf("%d audit events by the job, want 3: the end, the encounter's update and the appointment's", n)
	}
}

// The practitioner never joined: the session ends practitioner_absent with the patient's join kept, there is
// no encounter, and the appointment stays booked for the clinic to mark.
func TestConsultationWindowClose_EndsPractitionerAbsentAndLeavesTheAppointmentBooked(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, false, true)

	ended, logs := runWorkerJob(t, db, videosession.WindowCloseArgs{SessionID: c.session, OrganizationID: clinicID})
	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	checkConsultationLog(t, logs, b, c)
	var reason, appt string
	var patientJoined bool
	var encounters int
	if err := db.owner.QueryRow(t.Context(), `SELECT
		(SELECT end_reason FROM video_session WHERE id = $1), (SELECT patient_joined_at IS NOT NULL FROM video_session WHERE id = $1),
		(SELECT status FROM appointment WHERE id = $2), (SELECT count(*) FROM encounter)`, c.session, b.past).
		Scan(&reason, &patientJoined, &appt, &encounters); err != nil {
		t.Fatal(err)
	}
	if reason != "practitioner_absent" || !patientJoined || appt != "booked" || encounters != 0 {
		t.Errorf("session %s (patient joined %t), appointment %s, %d encounters; want practitioner_absent, the join kept, booked, none",
			reason, patientJoined, appt, encounters)
	}
}

// occupied is a video provider whose every room holds one participant; it changes nothing.
type occupied struct{ videosession.Provider }

func (occupied) Participants(context.Context, string) ([]string, error) {
	return []string{"p_someone"}, nil
}

// With someone in the room before the hard cap the job snoozes, logging the session and the snooze only, and
// changes nothing.
func TestConsultationWindowClose_SnoozesWhileTheRoomIsOccupied(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, true, true)

	ended, logs := runWorkerJobWith(t, db, occupied{liveKit(t)}, videosession.WindowCloseArgs{SessionID: c.session, OrganizationID: clinicID})
	if ended != river.EventKindJobSnoozed || !strings.Contains(logs, `"snooze_seconds":60`) {
		t.Fatalf("the job ended %s, want snoozed a minute:\n%s", ended, logs)
	}
	checkConsultationLog(t, logs, b, c)
	if n := countWhere(t, db, "SELECT count(*) FROM video_session WHERE ended_at IS NULL AND id = $1", c.session); n != 1 {
		t.Errorf("the session ended, want it open")
	}
}

// recording is the video provider recording the removals and deletes the jobs ask for, then passing them on.
type recording struct {
	videosession.Provider
	removed []string
	deleted []string
	mu      sync.Mutex
}

func (r *recording) RemoveParticipant(ctx context.Context, room, identity string) error {
	r.mu.Lock()
	r.removed = append(r.removed, room+" "+identity)
	r.mu.Unlock()
	return r.Provider.RemoveParticipant(ctx, room, identity)
}

func (r *recording) DeleteRoom(ctx context.Context, room string) error {
	r.mu.Lock()
	r.deleted = append(r.deleted, room)
	r.mu.Unlock()
	return r.Provider.DeleteRoom(ctx, room)
}

func (r *recording) calls() (removed, deleted []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.removed), slices.Clone(r.deleted)
}

// checkNoNames fails when the job's log holds a name, an e-mail, the room or an identity.
func checkNoNames(t *testing.T, logs string, b booking, c consultationRows) {
	t.Helper()
	for _, v := range []string{b.email, "Ana", "Cy", c.room, c.patientIdentity, c.practitionerIdentity} {
		if strings.Contains(logs, v) {
			t.Errorf("the log holds %q:\n%s", v, logs)
		}
	}
}

// A cancellation during the call (consultation_access_end): the job removes the patient, ends the session
// cancelled, finishes the encounter and deletes the room; the appointment stays cancelled. It logs the
// appointment's id only, and a repeat changes nothing.
func TestConsultationAccessEnd_EndsTheCancelledConsultation(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, true, true)
	if _, err := db.owner.Exec(t.Context(), `UPDATE appointment SET status = 'cancelled', cancellation_reason = 'prov', cancelled_at = now()
		WHERE id = $1`, b.past); err != nil {
		t.Fatal(err)
	}
	p := &recording{Provider: liveKit(t)}
	for run := range 2 {
		ended, logs := runWorkerJobWith(t, db, p, appointment.AccessEndArgs{AppointmentID: b.past, OrganizationID: clinicID})
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
		checkNoNames(t, logs, b, c)
		if !strings.Contains(logs, `"appointment_id":"`+b.past+`"`) {
			t.Errorf("run %d: the log lacks the appointment's id:\n%s", run, logs)
		}
	}
	// Each run asks LiveKit the same (what is gone stays gone); the database changes once.
	removed, deleted := p.calls()
	if want := c.room + " " + c.patientIdentity; !slices.Equal(removed, []string{want, want}) || !slices.Equal(deleted, []string{c.room, c.room}) {
		t.Errorf("removed %v, deleted %v; want the patient removed and the room deleted on each run", removed, deleted)
	}
	var reason, encounter, appt string
	if err := db.owner.QueryRow(t.Context(), `SELECT
		(SELECT end_reason FROM video_session WHERE id = $1),
		(SELECT status FROM encounter WHERE appointment_id = $2 AND period_end IS NOT NULL),
		(SELECT status FROM appointment WHERE id = $2)`, c.session, b.past).Scan(&reason, &encounter, &appt); err != nil {
		t.Fatal(err)
	}
	if reason != "cancelled" || encounter != "finished" || appt != "cancelled" {
		t.Errorf("session %s, encounter %s, appointment %s; want cancelled, finished, cancelled", reason, encounter, appt)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'consultation_access_end' AND purpose_code = 'HOPERAT'
		AND organization_id = $1`, clinicID); n != 2 {
		t.Errorf("%d audit events by the job, want 2: the end and the encounter's update", n)
	}
}

// The revocation (consent_revoked) during a consultation that started before it: the job removes the patient
// from its room and leaves the practitioner and the session. The future appointment it cancels had a session
// too, which the clinic's scope no longer shows: the job ends it cancelled and deletes its room. A retry
// changes nothing.
func TestConsentRevoked_RemovesThePatientFromTheConsultations(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := seedConsultation(t, db, b, true, true)
	future := consultationRows{room: "r_" + strings.ToLower(rand.Text()), patientIdentity: "p_" + strings.ToLower(rand.Text()),
		practitionerIdentity: "p_" + strings.ToLower(rand.Text())}
	if err := db.owner.QueryRow(t.Context(), `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity,
			practitioner_identity, closes_at)
		SELECT id, organization_id, $2, $3, $4, upper(period) + interval '30 minutes' FROM appointment WHERE id = $1 RETURNING id::text`,
		b.future, future.room, future.patientIdentity, future.practitionerIdentity).Scan(&future.session); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(t.Context(), "UPDATE consent SET status = 'inactive', revoked_at = now() - interval '1 minute'"); err != nil {
		t.Fatal(err)
	}
	p := &recording{Provider: liveKit(t)}
	for run := range 2 {
		ended, logs := runWorkerJobWith(t, db, p, consent.RevokedArgs{PatientID: b.patient, OrganizationID: clinicID})
		if ended != river.EventKindJobCompleted {
			t.Fatalf("run %d: the job ended %s, want completed:\n%s", run, ended, logs)
		}
		checkNoNames(t, logs, b, c)
		checkNoNames(t, logs, b, future)
	}
	// Each run asks LiveKit the same (what is gone stays gone); the database changes once.
	removed, deleted := p.calls()
	slices.Sort(removed)
	want := []string{c.room + " " + c.patientIdentity, c.room + " " + c.patientIdentity, future.room + " " + future.patientIdentity,
		future.room + " " + future.patientIdentity}
	slices.Sort(want)
	if !slices.Equal(removed, want) || !slices.Equal(deleted, []string{future.room, future.room}) {
		t.Errorf("removed %v, deleted %v; want the patient removed from both rooms and only the cancelled appointment's room %s deleted, on each run",
			removed, deleted, future.room)
	}
	var started, cancelled string
	if err := db.owner.QueryRow(t.Context(), `SELECT
		(SELECT coalesce(end_reason, 'open') FROM video_session WHERE id = $1),
		(SELECT coalesce(end_reason, 'open') FROM video_session WHERE id = $2)`, c.session, future.session).Scan(&started, &cancelled); err != nil {
		t.Fatal(err)
	}
	if started != "open" || cancelled != "cancelled" {
		t.Errorf("the started consultation %s, the cancelled one %s; want open and cancelled", started, cancelled)
	}
	if n := countWhere(t, db, `SELECT count(*) FROM audit_event WHERE agent_name = 'consent_revoked' AND subtype_code = 'operation'
		AND entity_reference = $1`, "Appointment/"+b.future); n != 1 {
		t.Errorf("%d audit events of the cancelled consultation's end by the job, want 1", n)
	}
}
