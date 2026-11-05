package worker_test

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"

	"github.com/riverqueue/river"

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
