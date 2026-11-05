package videosession_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

const endAgent = "consultation_end"

// statusRead is the audit event of c's read of the consultation of appointment A (entity "" when the caller
// may not see it) with outcome.
func statusRead(c appointment.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := joinEvent(c, entity, outcome)
	e.Subtype.Code, e.Action = auditevent.CodeRead, auditevent.ActionRead
	return e
}

// The appointment's patient, its practitioner and the clinic's admin read where the consultation stands, each
// in their own scope, audited as a read of the appointment: not started before the first token request, open
// while the session runs, ended with the reason once it has.
func TestStatus_GivesTheParticipantsAndTheAdminTheConsultationsState(t *testing.T) {
	practitionerJoined, patientJoined := start.Add(time.Minute), start.Add(2*time.Minute)
	for _, tt := range []struct {
		fixture func() *fixture
		want    videosession.Status
		name    string
	}{
		{name: "no session yet", fixture: func() *fixture { return newFixture(now) }, want: videosession.Status{State: videosession.StateNotStarted}},
		{name: "open, nobody joined", fixture: func() *fixture { return opened(now, time.Time{}, time.Time{}) }, want: videosession.Status{State: videosession.StateOpen}},
		{
			name: "open, both joined", fixture: func() *fixture { return opened(now, practitionerJoined, patientJoined) },
			want: videosession.Status{State: videosession.StateOpen, PractitionerJoinedAt: practitionerJoined, PatientJoinedAt: patientJoined},
		},
		{
			name: "ended, the practitioner absent",
			fixture: func() *fixture {
				f := opened(closes, time.Time{}, patientJoined)
				s := f.session()
				s.EndedAt, s.EndReason = closes, videosession.EndPractitionerAbsent
				f.repo.sessions[apptA] = s
				return f
			},
			want: videosession.Status{State: videosession.StateEnded, PatientJoinedAt: patientJoined, EndedAt: closes, EndReason: videosession.EndPractitionerAbsent},
		},
	} {
		for _, c := range []appointment.Caller{me, pract, admin} {
			f := tt.fixture()
			got, err := f.svc.Status(t.Context(), c, apptA)
			if err != nil || got != tt.want {
				t.Errorf("%s: Status() by %s = %+v, %v; want %+v", tt.name, c.AccountID, got, err, tt.want)
			}
			if (c == me) != (f.repo.patient == patientA) || (c != me) != (f.repo.clinic == clinicA) {
				t.Errorf("%s: read by %s in patient %q, clinic %q; want the caller's own scope", tt.name, c.AccountID, f.repo.patient, f.repo.clinic)
			}
			checkEvents(t, tt.name, f.audit, statusRead(c, "Appointment/"+apptA, auditevent.OutcomeSuccess))
			if len(f.provider.listed)+len(f.provider.rooms)+len(f.repo.opened) != 0 {
				t.Errorf("%s: asked LiveKit %v %v, opened %v; want a read of our own record only", tt.name, f.provider.listed, f.provider.rooms, f.repo.opened)
			}
		}
	}
}

// After a revocation the clinic still sees the consultation of a past appointment (the name level).
func TestStatus_GivesTheNameLevelTheState(t *testing.T) {
	f := opened(end, start, start)
	f.change(func(a *appointment.Attendance) { a.Level, a.ConsentActive = access.Name, false })
	if got, err := f.svc.Status(t.Context(), admin, apptA); err != nil || got.State != videosession.StateOpen {
		t.Errorf("Status() at the name level = %+v, %v; want it open", got, err)
	}
}

// Anyone the appointment's access rule does not show it to gets ErrNotFound, audited without the entity.
func TestStatus_IsNotFoundForAnyoneElseAudited(t *testing.T) {
	for _, c := range []appointment.Caller{other, pract2, adminB, anonymous} {
		f := opened(now, start, start)
		got, err := f.svc.Status(t.Context(), c, apptA)
		if !errors.Is(err, videosession.ErrNotFound) || got != (videosession.Status{}) {
			t.Errorf("Status() by %q = %+v, %v; want ErrNotFound", c.AccountID, got, err)
		}
		checkEvents(t, "a refused status", f.audit, statusRead(c, "", auditevent.OutcomeMinorFailure))
	}
}

func TestStatus_FailsWithTheBackendAndAuditsNothing(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the appointment": func(f *fixture) { f.appts.err = errBackend },
		"the session":     func(f *fixture) { f.repo.err = errBackend },
		"the audit":       func(f *fixture) { f.audit.err = errBackend },
	} {
		f := opened(now, start, start)
		fail(f)
		if _, err := f.svc.Status(t.Context(), me, apptA); !errors.Is(err, errBackend) || strings.Contains(err.Error(), apptA) {
			t.Errorf("%s failing: Status() = %v, want the backend's error without the id", name, err)
		}
		checkEvents(t, name+" failing", f.audit)
	}
}

// endRequest is the audit event of c's request to end the consultation of appointment A (entity "" when the
// caller may not see it) with outcome.
func endRequest(c appointment.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return joinEvent(c, entity, outcome)
}

// endedBy is the audit event of the session's end on the practitioner's request.
func endedBy() auditevent.AuditEvent {
	e := endEvent()
	e.AgentName = endAgent
	return e
}

// The practitioner ends the consultation after joining: in one transaction the session ends (practitioner),
// the encounter finishes with whether the patient joined, and both are audited; the room is deleted after
// the commit. The answer is the consultation's state.
func TestEnd_ThePractitionerEndsTheConsultation(t *testing.T) {
	for _, patientJoined := range []time.Time{start.Add(time.Minute), {}} {
		f := opened(start.Add(20*time.Minute), start, patientJoined)
		got, err := f.svc.End(t.Context(), pract, apptA)
		at := start.Add(20 * time.Minute)
		want := videosession.Status{State: videosession.StateEnded, PractitionerJoinedAt: start, PatientJoinedAt: patientJoined, EndedAt: at, EndReason: videosession.EndPractitioner}
		if err != nil || got != want {
			t.Fatalf("End() = %+v, %v; want %+v", got, err, want)
		}
		if s := f.session(); !s.EndedAt.Equal(at) || s.EndReason != videosession.EndPractitioner {
			t.Errorf("session = %+v, want ended at %v by the practitioner", s, at)
		}
		finished := endAgent + " " + apptA + " " + at.Format(time.RFC3339) + " in " + clinicA + ", patient joined " + map[bool]string{true: "true", false: "false"}[!patientJoined.IsZero()]
		if !slices.Equal(f.encounters.finished, []string{finished}) {
			t.Errorf("finished %v, want %q", f.encounters.finished, finished)
		}
		if !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
			t.Errorf("deleted %v (in a transaction: %t), want the room after the commit", f.provider.deleted, f.provider.inTx)
		}
		checkEvents(t, "End", f.audit, endRequest(pract, "Appointment/"+apptA, auditevent.OutcomeSuccess), endedBy())
	}
}

// Ending again answers the same, finishes nothing more and deletes the room again (a retry heals a failed
// delete); a session the window or a concurrent request ended first is answered as it ended.
func TestEnd_IsIdempotent(t *testing.T) {
	f := opened(start.Add(20*time.Minute), start, start)
	first, err := f.svc.End(t.Context(), pract, apptA)
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	again, err := f.svc.End(t.Context(), pract, apptA)
	if err != nil || again != first || len(f.encounters.finished) != 1 || len(f.provider.deleted) != 2 {
		t.Errorf("End() again = %+v, %v after finishing %v and deleting %v; want %+v, one finish, the room deleted twice",
			again, err, f.encounters.finished, f.provider.deleted, first)
	}
	checkEvents(t, "End again", f.audit, endRequest(pract, "Appointment/"+apptA, auditevent.OutcomeSuccess))

	f = opened(start.Add(20*time.Minute), start, start)
	f.repo.endedElsewhere = true
	got, err := f.svc.End(t.Context(), pract, apptA)
	if err != nil || got.State != videosession.StateEnded || got.EndReason != videosession.EndWindowClosed || len(f.encounters.finished) != 0 {
		t.Errorf("End() after a concurrent end = %+v, %v, finished %v; want the end as it happened, nothing finished", got, err, f.encounters.finished)
	}
	checkEvents(t, "End after a concurrent end", f.audit, endRequest(pract, "Appointment/"+apptA, auditevent.OutcomeSuccess))
}

// After a revocation the practitioner still ends the consultation of the appointment that started before it.
func TestEnd_ThePractitionerEndsItAfterARevocation(t *testing.T) {
	f := opened(start.Add(20*time.Minute), start, start)
	f.change(func(a *appointment.Attendance) { a.Level, a.Profile, a.ConsentActive = access.Name, access.Name, false })
	if got, err := f.svc.End(t.Context(), pract, apptA); err != nil || got.State != videosession.StateEnded {
		t.Errorf("End() after a revocation = %+v, %v; want it ended", got, err)
	}
}

// Only the appointment's practitioner may end it: the patient, the admin, another practitioner, another
// clinic and anyone else get ErrNotFound, audited (with the entity for those who may see the appointment),
// and nothing ends.
func TestEnd_IsNotFoundForAnyoneButThePractitionerAudited(t *testing.T) {
	for _, tt := range []struct {
		entity string
		caller appointment.Caller
	}{
		{caller: me, entity: "Appointment/" + apptA}, {caller: admin, entity: "Appointment/" + apptA},
		{caller: pract2}, {caller: adminB}, {caller: other}, {caller: anonymous},
	} {
		f := opened(start.Add(20*time.Minute), start, start)
		got, err := f.svc.End(t.Context(), tt.caller, apptA)
		if !errors.Is(err, videosession.ErrNotFound) || got != (videosession.Status{}) {
			t.Errorf("End() by %q = %+v, %v; want ErrNotFound", tt.caller.AccountID, got, err)
		}
		if !f.session().EndedAt.IsZero() || len(f.provider.deleted)+len(f.encounters.finished) != 0 {
			t.Errorf("End() by %q ended %+v, deleted %v; want nothing", tt.caller.AccountID, f.session(), f.provider.deleted)
		}
		checkEvents(t, "a refused end", f.audit, endRequest(tt.caller, tt.entity, auditevent.OutcomeMinorFailure))
	}
}

// Before the practitioner joined there is nothing to end: ErrNotStarted, audited.
func TestEnd_IsRefusedBeforeThePractitionerJoined(t *testing.T) {
	for name, f := range map[string]*fixture{
		"no session":       newFixture(now),
		"never joined yet": opened(now, time.Time{}, start),
	} {
		got, err := f.svc.End(t.Context(), pract, apptA)
		if !errors.Is(err, videosession.ErrNotStarted) || got != (videosession.Status{}) {
			t.Errorf("%s: End() = %+v, %v; want ErrNotStarted", name, got, err)
		}
		if len(f.provider.deleted)+len(f.encounters.finished) != 0 {
			t.Errorf("%s: deleted %v, finished %v; want nothing", name, f.provider.deleted, f.encounters.finished)
		}
		checkEvents(t, name, f.audit, endRequest(pract, "Appointment/"+apptA, auditevent.OutcomeMinorFailure))
	}
}

// A failure before the commit changes nothing; a failed delete of the room leaves the end committed and fails
// the request, which the practitioner repeats.
func TestEnd_Fails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the appointment": func(f *fixture) { f.appts.err = errBackend },
		"the session":     func(f *fixture) { f.repo.err = errBackend },
		"the end":         func(f *fixture) { f.repo.endErr = errBackend },
		"the encounter":   func(f *fixture) { f.encounters.err = errBackend },
		"the audit":       func(f *fixture) { f.audit.err = errBackend },
	} {
		f := opened(start.Add(20*time.Minute), start, start)
		fail(f)
		if _, err := f.svc.End(t.Context(), pract, apptA); !errors.Is(err, errBackend) || strings.Contains(err.Error(), "r_room") {
			t.Errorf("%s failing: End() = %v, want the backend's error without the room", name, err)
		}
		if !f.session().EndedAt.IsZero() || len(f.provider.deleted) != 0 {
			t.Errorf("%s failing: session %+v, deleted %v; want nothing changed", name, f.session(), f.provider.deleted)
		}
		checkEvents(t, name+" failing", f.audit)
	}

	f := opened(start.Add(20*time.Minute), start, start)
	f.provider.deleteErr = errBackend
	if _, err := f.svc.End(t.Context(), pract, apptA); !errors.Is(err, errBackend) {
		t.Errorf("a failed delete: End() = %v, want the provider's error", err)
	}
	if f.session().EndReason != videosession.EndPractitioner || len(f.encounters.finished) != 1 {
		t.Errorf("a failed delete: session %+v, finished %v; want the end committed", f.session(), f.encounters.finished)
	}
}
