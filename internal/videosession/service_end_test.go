package videosession_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

// The window closes 30 minutes after the scheduled end (10:00); the hard cap is 2 hours after it (11:30).
var (
	closes  = end.Add(30 * time.Minute)
	hardCap = end.Add(2 * time.Hour)
)

const windowAgent = "consultation_window_close"

// The first token request enqueues the window's close in the session's insert transaction, with the session
// and clinic ids, at the window's close; later requests enqueue nothing.
func TestJoin_TheFirstRequestEnqueuesTheWindowClose(t *testing.T) {
	f := newFixture(now)
	for _, c := range []appointment.Caller{me, pract, me} {
		if _, err := f.svc.Join(t.Context(), c, apptA); err != nil {
			t.Fatal(err)
		}
	}
	s := f.repo.sessions[apptA]
	if want := []string{s.ID + " " + clinicA + " " + closes.Format(time.RFC3339)}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want %v", f.jobs.enqueued, want)
	}
}

// A failed enqueue fails the join and rolls the session back with it.
func TestJoin_FailsWhenTheWindowCloseCannotBeEnqueued(t *testing.T) {
	f := newFixture(now)
	f.jobs.err = errBackend
	if _, err := f.svc.Join(t.Context(), me, apptA); !errors.Is(err, errBackend) {
		t.Fatalf("Join() = %v, want the queue's error", err)
	}
	if len(f.repo.sessions) != 0 || len(f.provider.grants) != 0 {
		t.Errorf("sessions %v, grants %v; want none", f.repo.sessions, f.provider.grants)
	}
}

// opened is the fixture at the time at with appointment A's session open; the practitioner and the patient
// joined when they did (zero for never).
func opened(at, practitionerJoined, patientJoined time.Time) *fixture {
	f := newFixture(at)
	f.repo.sessions[apptA] = videosession.Session{
		ID: "5e000000-0000-4000-8000-000000000001", AppointmentID: apptA, OrganizationID: clinicA, ClosesAt: closes, CreatedAt: now,
		Room: "r_room", PatientIdentity: "p_patient", PractitionerIdentity: "p_practitioner",
		PractitionerJoinedAt: practitionerJoined, PatientJoinedAt: patientJoined,
	}
	return f
}

func (f *fixture) session() videosession.Session { return f.repo.sessions[apptA] }

// endEvent is the audit event of the window-close job's end of appointment A's session.
func endEvent() auditevent.AuditEvent {
	e := joinEvent(anonymous, "Appointment/"+apptA, auditevent.OutcomeSuccess)
	e.Purpose.Code, e.AgentName, e.OrganizationID = auditevent.CodeHOPERAT, windowAgent, clinicA
	return e
}

func closeWindow(t *testing.T, f *fixture) (time.Duration, error) {
	t.Helper()
	return f.svc.CloseWindow(t.Context(), f.session().ID, clinicA)
}

// The window closes with the room empty: the session ends (window_closed) and the encounter finishes, with
// whether the patient joined, in one transaction in the clinic's scope, audited; the room is deleted after
// the commit.
func TestCloseWindow_WithTheRoomEmptyEndsTheSessionAndDeletesTheRoom(t *testing.T) {
	for _, patientJoined := range []bool{true, false} {
		joinedAt := time.Time{}
		if patientJoined {
			joinedAt = start
		}
		f := opened(closes, start, joinedAt)
		snooze, err := closeWindow(t, f)
		if err != nil || snooze != 0 {
			t.Fatalf("CloseWindow() = %v, %v; want done", snooze, err)
		}
		if s := f.session(); !s.EndedAt.Equal(closes) || s.EndReason != videosession.EndWindowClosed {
			t.Errorf("session %+v, want ended window_closed at the close", s)
		}
		want := windowAgent + " " + apptA + " " + closes.Format(time.RFC3339) + " in " + clinicA + ", patient joined " + map[bool]string{true: "true", false: "false"}[patientJoined]
		if !slices.Equal(f.encounters.finished, []string{want}) {
			t.Errorf("finished %v, want %v", f.encounters.finished, want)
		}
		if !slices.Equal(f.provider.listed, []string{"r_room"}) || !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
			t.Errorf("listed %v, deleted %v (in a transaction: %t); want the room asked and deleted outside it", f.provider.listed, f.provider.deleted, f.provider.inTx)
		}
		checkEvents(t, "CloseWindow", f.audit, endEvent())
	}
}

// With someone in the room the consultation runs over: the job snoozes a minute and changes nothing.
func TestCloseWindow_WithAParticipantSnoozesAMinute(t *testing.T) {
	f := opened(closes, start, start)
	f.provider.in = []string{"p_practitioner"}
	snooze, err := closeWindow(t, f)
	if err != nil || snooze != time.Minute {
		t.Fatalf("CloseWindow() = %v, %v; want a minute's snooze", snooze, err)
	}
	if s := f.session(); !s.EndedAt.IsZero() || len(f.encounters.finished) != 0 || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, finished %v, deleted %v, events %v; want nothing", s, f.encounters.finished, f.provider.deleted, f.audit.events)
	}
}

// When the practitioner never joined the session ends practitioner_absent and the room is deleted, whoever
// is in it; there is no encounter to finish, so the appointment stays booked, and the patient's join is kept.
func TestCloseWindow_WhenThePractitionerNeverJoinedEndsPractitionerAbsent(t *testing.T) {
	f := opened(closes, time.Time{}, start)
	f.provider.in = []string{"p_patient"}
	snooze, err := closeWindow(t, f)
	if err != nil || snooze != 0 {
		t.Fatalf("CloseWindow() = %v, %v; want done", snooze, err)
	}
	if s := f.session(); s.EndReason != videosession.EndPractitionerAbsent || !s.EndedAt.Equal(closes) || !s.PatientJoinedAt.Equal(start) {
		t.Errorf("session %+v, want ended practitioner_absent with the patient's join kept", s)
	}
	if len(f.encounters.finished) != 0 || len(f.provider.listed) != 0 || !slices.Equal(f.provider.deleted, []string{"r_room"}) {
		t.Errorf("finished %v, listed %v, deleted %v; want no encounter, the room deleted", f.encounters.finished, f.provider.listed, f.provider.deleted)
	}
	checkEvents(t, "CloseWindow", f.audit, endEvent())
}

// The hard cap, 2 hours after the scheduled end (± 1 s): before it an occupied room snoozes, never past the
// cap; at and after it the job ends the session (time_limit), finishes the encounter and deletes the room
// whoever is in it.
func TestCloseWindow_AtTheHardCapEndsAnOccupiedRoom(t *testing.T) {
	for _, tt := range []struct {
		at     time.Time
		snooze time.Duration
		ended  bool
	}{
		{at: hardCap.Add(-time.Minute - time.Second), snooze: time.Minute},
		{at: hardCap.Add(-20 * time.Second), snooze: 20 * time.Second},
		{at: hardCap.Add(-time.Second), snooze: time.Second},
		{at: hardCap, ended: true},
		{at: hardCap.Add(time.Second), ended: true},
	} {
		f := opened(tt.at, start, start)
		f.provider.in = []string{"p_practitioner", "p_patient"}
		snooze, err := closeWindow(t, f)
		if err != nil || snooze != tt.snooze {
			t.Errorf("at %v: CloseWindow() = %v, %v; want a snooze of %v", tt.at, snooze, err, tt.snooze)
		}
		s := f.session()
		if tt.ended != (s.EndReason == videosession.EndTimeLimit) || tt.ended != slices.Equal(f.provider.deleted, []string{"r_room"}) ||
			tt.ended != (len(f.encounters.finished) == 1) || (tt.ended && !s.EndedAt.Equal(tt.at)) {
			t.Errorf("at %v: session %+v, deleted %v, finished %v; want ended by the time limit: %t", tt.at, s, f.provider.deleted, f.encounters.finished, tt.ended)
		}
	}
}

// A session ended already (by the practitioner, or by a run whose room delete failed) changes nothing; the
// room is deleted again, which is idempotent, so a retry heals.
func TestCloseWindow_OnAnEndedSessionDeletesTheRoomOnly(t *testing.T) {
	f := opened(closes, start, start)
	s := f.session()
	s.EndedAt, s.EndReason = end, videosession.EndPractitioner
	f.repo.sessions[apptA] = s
	if snooze, err := closeWindow(t, f); err != nil || snooze != 0 {
		t.Fatalf("CloseWindow() = %v, %v; want done", snooze, err)
	}
	if f.session() != s || len(f.encounters.finished) != 0 || len(f.audit.events) != 0 || !slices.Equal(f.provider.deleted, []string{"r_room"}) {
		t.Errorf("session %+v, finished %v, events %v, deleted %v; want only the room deleted", f.session(), f.encounters.finished, f.audit.events, f.provider.deleted)
	}
}

// A session the clinic's scope does not show (gone, or after a revocation) is left alone.
func TestCloseWindow_DoesNothingForASessionOutOfScope(t *testing.T) {
	f := opened(closes, start, start)
	if snooze, err := f.svc.CloseWindow(t.Context(), f.session().ID, clinicB); err != nil || snooze != 0 {
		t.Fatalf("CloseWindow() = %v, %v; want done", snooze, err)
	}
	if !f.session().EndedAt.IsZero() || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, deleted %v, events %v; want nothing", f.session(), f.provider.deleted, f.audit.events)
	}
}

// The room is deleted after the commit: when the delete fails the end stays committed and the job fails, to
// be retried; the retry deletes the room.
func TestCloseWindow_ADeleteFailureLeavesTheEndCommittedForARetry(t *testing.T) {
	f := opened(closes, start, start)
	f.provider.deleteErr = errBackend
	if _, err := closeWindow(t, f); !errors.Is(err, errBackend) {
		t.Fatalf("CloseWindow() = %v, want the provider's error (River retries)", err)
	}
	if s := f.session(); s.EndReason != videosession.EndWindowClosed || len(f.encounters.finished) != 1 || len(f.audit.events) != 1 {
		t.Errorf("session %+v, finished %v, events %v; want the end committed", s, f.encounters.finished, f.audit.events)
	}
	f.provider.deleteErr = nil
	if snooze, err := closeWindow(t, f); err != nil || snooze != 0 || len(f.provider.deleted) != 2 || len(f.encounters.finished) != 1 {
		t.Errorf("retry: %v, %v, deleted %v, finished %v; want the room deleted again and nothing else", snooze, err, f.provider.deleted, f.encounters.finished)
	}
}

// A failure before or inside the end's transaction changes nothing and deletes no room.
func TestCloseWindow_FailsWithTheBackendAndChangesNothing(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"read":         func(f *fixture) { f.repo.err = errBackend },
		"participants": func(f *fixture) { f.provider.listErr = errBackend },
		"end":          func(f *fixture) { f.repo.endErr = errBackend },
		"encounter":    func(f *fixture) { f.encounters.err = errBackend },
		"audit":        func(f *fixture) { f.audit.err = errBackend },
	} {
		f := opened(closes, start, start)
		fail(f)
		_, err := closeWindow(t, f)
		if !errors.Is(err, errBackend) {
			t.Errorf("%s: CloseWindow() = %v, want its error", name, err)
		}
		if !f.session().EndedAt.IsZero() || len(f.provider.deleted) != 0 || len(f.encounters.finished) != 0 {
			t.Errorf("%s: session %+v, deleted %v, finished %v; want nothing", name, f.session(), f.provider.deleted, f.encounters.finished)
		}
		if strings.Contains(err.Error(), "r_room") || strings.Contains(err.Error(), "p_patient") {
			t.Errorf("%s: error %q holds the room or an identity", name, err)
		}
	}
}
