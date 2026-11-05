package videosession_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

// The status says why the consultation ended or its patient was removed (SPEC.md §2 "Consultation status",
// changed after package 6), for every path that ends or removes: the practitioner's end, the window's close
// with the room empty, without the practitioner, at the hard cap, a cancellation, a revocation after and
// before the start, the webhook's re-check; and a patient who left on their own.
func TestStatus_SaysWhyTheConsultationEndedOrThePatientWasRemoved(t *testing.T) {
	for _, tt := range reasonPaths() {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.fixture()
			if err := tt.run(t, f); err != nil {
				t.Fatal(err)
			}
			got, err := f.svc.Status(t.Context(), me, apptA)
			if err != nil || got.State != tt.state || got.Reason != tt.want {
				t.Errorf("Status() = %+v, %v; want %s with the reason %q", got, err, tt.state, tt.want)
			}
		})
	}
}

// reasonPath is a path that ends the consultation or removes its patient, and the state and reason it leaves.
type reasonPath struct {
	fixture func() *fixture
	run     func(t *testing.T, f *fixture) error
	name    string
	want    videosession.Reason
	state   videosession.State
}

func reasonPaths() []reasonPath {
	inRoom := func() []string { return []string{"p_practitioner", "p_patient"} } // a fresh slice: a removal changes it
	return []reasonPath{
		{name: "the practitioner ends it", state: videosession.StateEnded, want: videosession.ReasonEnded,
			fixture: func() *fixture { return opened(during, start, start) },
			run: func(t *testing.T, f *fixture) error {
				_, err := f.svc.End(t.Context(), pract, apptA)
				return err
			}},
		{name: "the window closes with the room empty", state: videosession.StateEnded, want: videosession.ReasonEnded,
			fixture: func() *fixture { return opened(closes, start, start) },
			run:     func(t *testing.T, f *fixture) error { _, err := closeWindow(t, f); return err }},
		{name: "the window closes without the practitioner", state: videosession.StateEnded, want: videosession.ReasonPractitionerDidNotJoin,
			fixture: func() *fixture { return opened(closes, time.Time{}, start) },
			run:     func(t *testing.T, f *fixture) error { _, err := closeWindow(t, f); return err }},
		{name: "the hard cap", state: videosession.StateEnded, want: videosession.ReasonTimeLimit,
			fixture: func() *fixture {
				f := opened(hardCap, start, start)
				f.provider.in = inRoom()
				return f
			},
			run: func(t *testing.T, f *fixture) error { _, err := closeWindow(t, f); return err }},
		{name: "a cancellation during the call", state: videosession.StateEnded, want: videosession.ReasonCancelled,
			fixture: func() *fixture {
				f := opened(during, start, start)
				f.provider.in = inRoom()
				f.cancel()
				return f
			},
			run: func(t *testing.T, f *fixture) error { return f.svc.EndAccess(t.Context(), apptA, clinicA) }},
		{name: "a revocation after the start", state: videosession.StateOpen, want: videosession.ReasonConsentWithdrawn,
			fixture: func() *fixture {
				f := revoked()
				f.provider.in = inRoom()
				return f
			},
			run: func(t *testing.T, f *fixture) error { return f.svc.EndPatientAccess(t.Context(), patientA, clinicA) }},
		{name: "a revocation before the start, which cancels", state: videosession.StateEnded, want: videosession.ReasonConsentWithdrawn,
			fixture: revokedBeforeTheStart,
			run:     func(t *testing.T, f *fixture) error { return f.svc.EndPatientAccess(t.Context(), patientA, clinicA) }},
		{name: "the webhook's re-check, the consent revoked", state: videosession.StateOpen, want: videosession.ReasonConsentWithdrawn,
			fixture: func() *fixture {
				f := opened(now, start, time.Time{})
				f.provider.in = inRoom()
				f.consents.revoked = true
				return f
			},
			run: func(t *testing.T, f *fixture) error { _, err := deliver(t, f, joined("p_patient")); return err }},
		{name: "the webhook's re-check, the appointment out of the clinic's scope", state: videosession.StateOpen, want: videosession.ReasonConsentWithdrawn,
			fixture: func() *fixture {
				f := opened(now, start, time.Time{})
				f.provider.in = inRoom()
				f.appts.hidden = true
				return f
			},
			run: func(t *testing.T, f *fixture) error { _, err := deliver(t, f, joined("p_patient")); return err }},
		{name: "the webhook's re-check, the appointment cancelled", state: videosession.StateOpen, want: videosession.ReasonCancelled,
			fixture: func() *fixture {
				f := opened(now, start, time.Time{})
				f.provider.in = inRoom()
				f.cancel()
				return f
			},
			run: func(t *testing.T, f *fixture) error { _, err := deliver(t, f, joined("p_patient")); return err }},
		{name: "the patient left on their own", state: videosession.StateOpen, want: videosession.ReasonLeft,
			fixture: func() *fixture { return opened(during, start, start) },
			run:     func(*testing.T, *fixture) error { return nil }},
		{name: "nobody joined yet", state: videosession.StateOpen,
			fixture: func() *fixture { return opened(now, time.Time{}, time.Time{}) },
			run:     func(*testing.T, *fixture) error { return nil }},
	}
}

// A removal keeps its first reason: the consent withdrawn stays the reason once the practitioner ends the
// consultation, and a later cancellation does not replace it.
func TestStatus_KeepsTheFirstRemovalsReason(t *testing.T) {
	f := revoked()
	f.provider.in = []string{"p_practitioner", "p_patient"}
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	f.provider.in = []string{"p_practitioner", "p_patient"} // the patient again, on a token issued before
	f.cancel()
	if err := f.svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Status(t.Context(), me, apptA)
	if err != nil || got.State != videosession.StateEnded || got.EndReason != videosession.EndCancelled || got.Reason != videosession.ReasonConsentWithdrawn {
		t.Errorf("Status() = %+v, %v; want ended cancelled, with the reason consent_withdrawn", got, err)
	}
}

// revokedBeforeTheStart is appointment A, both in the room before its start, after the patient revoked their
// consent: the revocation cancelled it, and the clinic's scope shows neither it nor its session, so its
// practitioner's and admin's access rule finds nothing.
func revokedBeforeTheStart() *fixture {
	f := opened(now, now.Add(-time.Minute), now.Add(-time.Minute))
	f.provider.in = []string{"p_practitioner", "p_patient"}
	f.consents.revoked = true
	f.cancel()
	f.repo.hidden, f.appts.hidden = true, true
	return f
}

// The practitioner who was in the room reads the status after a revocation before the start cancelled the
// appointment and hid it from the clinic: the consultation ended, and why. Audited as their read of the
// appointment.
func TestStatus_ThePractitionerWhoWasInTheRoomReadsItAfterTheClinicLostSightOfIt(t *testing.T) {
	f := revokedBeforeTheStart()
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	got, err := f.svc.Status(t.Context(), pract, apptA)
	want := videosession.Status{
		State: videosession.StateEnded, PractitionerJoinedAt: now.Add(-time.Minute), PatientJoinedAt: now.Add(-time.Minute),
		EndedAt: now, EndReason: videosession.EndCancelled, Reason: videosession.ReasonConsentWithdrawn,
	}
	if err != nil || got != want {
		t.Errorf("Status() by the practitioner = %+v, %v; want %+v", got, err, want)
	}
	checkEvents(t, "the practitioner's read", f.audit, statusRead(pract, "Appointment/"+apptA, auditevent.OutcomeSuccess))
}

// The room rule is the practitioner's alone: another practitioner of the clinic, its admin, another clinic's
// staff and the practitioner before they joined get ErrNotFound, audited without the entity.
func TestStatus_TheRoomRuleShowsNobodyElse(t *testing.T) {
	otherClinic := appointment.Caller{AccountID: "ac300000-0000-4000-8000-0000000000b3", OrganizationID: clinicB, PractitionerID: practA1}
	for _, tt := range []struct {
		fixture func() *fixture
		name    string
		caller  appointment.Caller
	}{
		{name: "another practitioner", caller: pract2, fixture: revokedBeforeTheStart},
		{name: "the clinic's admin", caller: admin, fixture: revokedBeforeTheStart},
		{name: "another clinic", caller: otherClinic, fixture: revokedBeforeTheStart},
		{name: "the practitioner before they joined", caller: pract, fixture: func() *fixture {
			f := revokedBeforeTheStart()
			s := f.session()
			s.PractitionerJoinedAt = time.Time{}
			f.repo.sessions[apptA] = s
			return f
		}},
	} {
		f := tt.fixture()
		got, err := f.svc.Status(t.Context(), tt.caller, apptA)
		if !errors.Is(err, videosession.ErrNotFound) || got != (videosession.Status{}) {
			t.Errorf("%s: Status() = %+v, %v; want ErrNotFound", tt.name, got, err)
		}
		checkEvents(t, tt.name, f.audit, statusRead(tt.caller, "", auditevent.OutcomeMinorFailure))
	}
}

// A failure while reading by the room rule fails the read, audited nothing.
func TestStatus_TheRoomRuleFailsWithTheBackend(t *testing.T) {
	f := revokedBeforeTheStart()
	f.appts.practitionerErr = errBackend
	if _, err := f.svc.Status(t.Context(), pract, apptA); !errors.Is(err, errBackend) {
		t.Errorf("Status() = %v, want the backend's error", err)
	}
	checkEvents(t, "a failed read", f.audit)
}
