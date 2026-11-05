package videosession_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	accessEndAgent  = "consultation_access_end"
	revocationAgent = "consent_revoked"
)

// during is a time in the consultation: both joined at the start, five minutes in.
var during = start.Add(5 * time.Minute)

// agentEnd is the audit event of the agent's end of appointment A's session.
func agentEnd(agent string) auditevent.AuditEvent {
	e := endEvent()
	e.AgentName = agent
	return e
}

func (f *fixture) cancel() {
	f.change(func(att *appointment.Attendance) { att.Appointment.Status = appointment.StatusCancelled })
}

// A cancellation during the call (consultation_access_end): the patient is removed, then the session ends
// cancelled in the clinic's scope, the Encounter finishes (its outcome stays the cancellation), the end is
// audited by the job, and the room is deleted after the commit, which disconnects the practitioner too.
func TestEndAccess_OnACancelledAppointmentRemovesThePatientEndsTheSessionAndDeletesTheRoom(t *testing.T) {
	f := opened(during, start, start)
	f.cancel()
	if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatalf("EndAccess() = %v", err)
	}
	if want := []string{"r_room p_patient"}; !slices.Equal(f.provider.removed, want) {
		t.Errorf("removed %v, want %v", f.provider.removed, want)
	}
	if s := f.session(); !s.EndedAt.Equal(during) || s.EndReason != videosession.EndCancelled {
		t.Errorf("session %+v, want ended cancelled now", s)
	}
	want := accessEndAgent + " " + apptA + " " + during.Format(time.RFC3339) + " in " + clinicA + ", patient joined true"
	if !slices.Equal(f.encounters.finished, []string{want}) {
		t.Errorf("finished %v, want %v", f.encounters.finished, want)
	}
	if !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
		t.Errorf("deleted %v (in a transaction: %t), want the room deleted after the commit", f.provider.deleted, f.provider.inTx)
	}
	checkEvents(t, "EndAccess", f.audit, agentEnd(accessEndAgent))
}

// Repeated, the job changes nothing: the session keeps its end, nothing is finished or audited again; it
// only asks LiveKit again to remove and delete what is gone (a retry after a failed delete heals).
func TestEndAccess_RepeatedChangesNothing(t *testing.T) {
	f := opened(during, start, start)
	f.cancel()
	if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatal(err)
	}
	ended := f.session()
	f.svc.Clock = clock.Fixed{T: during.Add(time.Minute)}
	f.audit.events, f.encounters.finished, f.provider.deleted = nil, nil, nil
	if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatalf("EndAccess() again = %v", err)
	}
	if s := f.session(); s != ended || len(f.encounters.finished) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, finished %v, events %v; want it as it ended and nothing new", s, f.encounters.finished, f.audit.events)
	}
	if !slices.Equal(f.provider.deleted, []string{"r_room"}) {
		t.Errorf("deleted %v, want the room deleted again", f.provider.deleted)
	}
}

// With no session (nobody asked for a token), or one the clinic's scope does not show, it does nothing.
func TestEndAccess_WithNoSessionDoesNothing(t *testing.T) {
	for name, f := range map[string]*fixture{"no session": newFixture(during), "out of scope": opened(during, start, start)} {
		f.cancel()
		f.repo.hidden = true
		if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
			t.Errorf("%s: EndAccess() = %v, want nil", name, err)
		}
		if len(f.provider.removed) != 0 || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 || len(f.encounters.finished) != 0 {
			t.Errorf("%s: removed %v, deleted %v, events %v, finished %v; want nothing", name, f.provider.removed, f.provider.deleted,
				f.audit.events, f.encounters.finished)
		}
	}
}

// For an appointment still booked it only removes the patient: the consultation goes on.
func TestEndAccess_OnABookedAppointmentOnlyRemovesThePatient(t *testing.T) {
	f := opened(during, start, start)
	if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.provider.removed, []string{"r_room p_patient"}) || len(f.provider.deleted) != 0 || !f.session().EndedAt.IsZero() ||
		len(f.audit.events) != 0 {
		t.Errorf("removed %v, deleted %v, session %+v, events %v; want only the patient removed", f.provider.removed, f.provider.deleted,
			f.session(), f.audit.events)
	}
}

// A failure fails the job for River to retry: a failed read or removal ends nothing; a failed delete leaves
// the end committed.
func TestEndAccess_FailsForARetry(t *testing.T) {
	for name, setup := range map[string]func(*fixture){
		"the session":       func(f *fixture) { f.repo.err = errBackend },
		"the appointment":   func(f *fixture) { f.appts.err = errBackend },
		"the removal":       func(f *fixture) { f.provider.removeErr = errBackend },
		"the end":           func(f *fixture) { f.repo.endErr = errBackend },
		"the room's delete": func(f *fixture) { f.provider.deleteErr = errBackend },
	} {
		f := opened(during, start, start)
		f.cancel()
		setup(f)
		if err := f.svc.EndAccess(t.Context(), apptA, clinicA); !errors.Is(err, errBackend) {
			t.Errorf("with %s failing: EndAccess() = %v, want the error", name, err)
		}
		if ended := !f.session().EndedAt.IsZero(); ended != (name == "the room's delete") {
			t.Errorf("with %s failing: session %+v, ended %t", name, f.session(), ended)
		}
	}
}

// revoked is the fixture after patient A revoked their consent with clinic A during the consultation of
// appointment A, which had started: the clinic still sees it (the care record) and it stays booked.
func revoked() *fixture {
	f := opened(during, start, start)
	f.consents.revoked = true
	return f
}

// The revocation job removes the patient from the open room of an appointment that started before the
// revocation, read in the patient's scope; the practitioner stays and the session goes on, for them to end.
func TestEndPatientAccess_RemovesThePatientFromTheRoomOfAStartedAppointmentAndLeavesThePractitioner(t *testing.T) {
	f := revoked()
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("EndPatientAccess() = %v", err)
	}
	if !slices.Equal(f.provider.removed, []string{"r_room p_patient"}) || f.provider.inTx {
		t.Errorf("removed %v (in a transaction: %t), want the patient removed after the commit", f.provider.removed, f.provider.inTx)
	}
	if !f.session().EndedAt.IsZero() || len(f.provider.deleted) != 0 || len(f.encounters.finished) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, deleted %v, finished %v, events %v; want the consultation to go on", f.session(), f.provider.deleted,
			f.encounters.finished, f.audit.events)
	}
}

// An appointment that had not started is cancelled by the revocation, and the clinic's scope no longer shows
// its session: the job removes the patient and ends the session cancelled in the room's scope, audited, and
// deletes the room.
func TestEndPatientAccess_EndsTheSessionOfAnAppointmentTheRevocationCancelled(t *testing.T) {
	f := revoked()
	f.cancel()
	f.repo.hidden, f.appts.hidden = true, true
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("EndPatientAccess() = %v", err)
	}
	if s := f.session(); !s.EndedAt.Equal(during) || s.EndReason != videosession.EndCancelled {
		t.Errorf("session %+v, want ended cancelled now", s)
	}
	if !slices.Equal(f.provider.removed, []string{"r_room p_patient"}) || !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
		t.Errorf("removed %v, deleted %v (in a transaction: %t); want the patient removed and the room deleted after the commit",
			f.provider.removed, f.provider.deleted, f.provider.inTx)
	}
	checkEvents(t, "EndPatientAccess", f.audit, agentEnd(revocationAgent))

	f.audit.events, f.provider.removed, f.provider.deleted = nil, nil, nil
	ended := f.session()
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("EndPatientAccess() again = %v", err)
	}
	if f.session() != ended || len(f.audit.events) != 0 {
		t.Errorf("again: session %+v, events %v; want nothing changed", f.session(), f.audit.events)
	}
}

// Only the clinic's sessions whose room may still be open count: another clinic's, and one past its hard cap
// (its room is gone), are left alone. Repeated, the job changes nothing.
func TestEndPatientAccess_LeavesOtherClinicsAndPastSessionsAndChangesNothingWhenRepeated(t *testing.T) {
	f := revoked()
	f.repo.sessions[apptA2] = videosession.Session{
		ID: "5e000000-0000-4000-8000-000000000002", AppointmentID: apptA2, OrganizationID: clinicB, ClosesAt: closes,
		Room: "r_other", PatientIdentity: "p_other", PractitionerIdentity: "p_other_practitioner",
	}
	f.repo.sessions["e5000000-0000-4000-8000-0000000000a5"] = videosession.Session{
		ID: "5e000000-0000-4000-8000-000000000003", AppointmentID: "e5000000-0000-4000-8000-0000000000a5", OrganizationID: clinicA,
		ClosesAt: during.Add(-(hardCap.Sub(closes))), Room: "r_past", PatientIdentity: "p_past", PractitionerIdentity: "p_past_practitioner",
	}
	for range 2 {
		f.provider.removed = nil
		if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(f.provider.removed, []string{"r_room p_patient"}) || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
			t.Errorf("removed %v, deleted %v, events %v; want only the open room of clinic A", f.provider.removed, f.provider.deleted, f.audit.events)
		}
	}
}

// With no session it does nothing.
func TestEndPatientAccess_WithNoSessionDoesNothing(t *testing.T) {
	f := newFixture(during)
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil || len(f.provider.removed) != 0 {
		t.Errorf("EndPatientAccess() = %v, removed %v; want nothing", err, f.provider.removed)
	}
}

// A failure fails the job for River to retry; a failed removal from one room does not keep the patient in
// another.
func TestEndPatientAccess_FailsForARetry(t *testing.T) {
	for name, setup := range map[string]func(*fixture){
		"the sessions":    func(f *fixture) { f.repo.err = errBackend },
		"the appointment": func(f *fixture) { f.appts.err = errBackend },
		"the removal":     func(f *fixture) { f.provider.removeErr = errBackend },
	} {
		f := revoked()
		setup(f)
		if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); !errors.Is(err, errBackend) {
			t.Errorf("with %s failing: EndPatientAccess() = %v, want the error", name, err)
		}
	}
	f := revoked()
	f.repo.sessions[apptA2] = videosession.Session{
		ID: "5e000000-0000-4000-8000-000000000002", AppointmentID: apptA2, OrganizationID: clinicA, ClosesAt: closes,
		Room: "r_second", PatientIdentity: "p_second", PractitionerIdentity: "p_second_practitioner",
	}
	f.appts.attendance[practAcc+apptA2] = f.appts.attendance[practAcc+apptA]
	f.provider.removeErr = errBackend
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); !errors.Is(err, errBackend) || len(f.provider.removed) != 2 {
		t.Errorf("EndPatientAccess() = %v, removed %v; want the error after trying both rooms", err, f.provider.removed)
	}
}
