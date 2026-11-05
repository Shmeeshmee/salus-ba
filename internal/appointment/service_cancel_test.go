package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

const (
	apptA  = "e1000000-0000-4000-8000-0000000000a1" // patient A with role A1, clinic A, at start
	apptB  = "e2000000-0000-4000-8000-0000000000b1" // patient B with role A1, clinic A, at start + 1 h
	apptA2 = "e3000000-0000-4000-8000-0000000000a2" // patient A with role A2, clinic A, at start + 2 h
	apptX  = "e4000000-0000-4000-8000-0000000000b2" // patient A with role B, clinic B, at start
)

func cancelFixture(at time.Time) *fixture {
	return newFixture(
		booked(apptA, patientA, roleA1, clinicA, at),
		booked(apptB, patientB, roleA1, clinicA, at.Add(time.Hour)),
		booked(apptA2, patientA, roleA2, clinicA, at.Add(2*time.Hour)),
		booked(apptX, patientA, roleB, clinicB, at),
	)
}

func TestCancel_ThePatientCancelsUpTo24HoursBeforeTheStart(t *testing.T) {
	f := cancelFixture(now.Add(24*time.Hour + time.Second))
	got, err := f.svc.Cancel(t.Context(), me, apptA)
	if err != nil || got.Status != appointment.StatusCancelled || got.Reason != appointment.ReasonPatient || got.CancelledBy != accountA ||
		!got.CancelledAt.Equal(now) || got.VersionID != 2 {
		t.Fatalf("Cancel() = %+v, %v; want cancelled by patient A (pat) now, at version 2", got, err)
	}
	if want := []string{"SetPatient", "ByID", "SetClinic", "Cancel"}; !slices.Equal(f.repo.calls, want) {
		t.Errorf("calls %v, want %v", f.repo.calls, want)
	}
	if want := []string{clinicA + "/" + "51" + apptA[2:]}; !slices.Equal(f.slots.released, want) {
		t.Errorf("released %v, want the appointment's slot %v", f.slots.released, want)
	}
	if want := []string{"cancelled " + apptA + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want %v", f.jobs.enqueued, want)
	}
	checkEvents(t, "Cancel", f.audit, event(me, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestCancel_StaffCancelAtAnyTime(t *testing.T) {
	for _, c := range []appointment.Caller{pract, admin} {
		f := cancelFixture(now.Add(time.Minute))
		got, err := f.svc.Cancel(t.Context(), c, apptA)
		if err != nil || got.Status != appointment.StatusCancelled || got.Reason != appointment.ReasonProvider || got.CancelledBy != c.AccountID {
			t.Errorf("Cancel() as %s = %+v, %v; want cancelled by them (prov)", c.AccountID, got, err)
			continue
		}
		if f.repo.patient != "" || f.repo.clinic != clinicA || len(f.slots.released) != 1 || len(f.jobs.enqueued) != 1 {
			t.Errorf("as %s: patient %q, clinic %q, released %v, enqueued %v; want clinic A's scope, the slot freed and the e-mail", c.AccountID,
				f.repo.patient, f.repo.clinic, f.slots.released, f.jobs.enqueued)
		}
		checkEvents(t, "Cancel as staff", f.audit, event(c, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
	}
}

func TestCancel_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		at      time.Time
		want    error
		consent *access.Consent // patient A's with clinic A, when changed
		name    string
		id      string
		caller  appointment.Caller
		closed  bool
	}{
		{name: "the patient 24 h - 1 s before", caller: me, id: apptA, at: now.Add(24*time.Hour - time.Second), want: appointment.ErrCancellationWindowClosed},
		{name: "another patient's", caller: other, id: apptA, at: start, want: appointment.ErrNotFound},
		{name: "a patient without a profile", caller: noProfile, id: apptA, at: start, want: appointment.ErrNotFound},
		{name: "another practitioner's", caller: pract2, id: apptA, at: start, want: appointment.ErrNotFound},
		{name: "another clinic's", caller: adminB, id: apptA, at: start, want: appointment.ErrNotFound},
		{name: "an unknown one", caller: admin, id: "e9000000-0000-4000-8000-0000000000e9", at: start, want: appointment.ErrNotFound},
		{name: "one cancelled already", caller: me, id: apptA, at: start, closed: true, want: appointment.ErrClosed},
		{name: "a past one after the revocation", caller: pract, id: apptA, at: now.Add(-time.Hour), want: appointment.ErrForbidden,
			consent: &access.Consent{RevokedAt: now.Add(-time.Minute)}},
		{name: "a future one after the revocation", caller: admin, id: apptA, at: start, want: appointment.ErrNotFound,
			consent: &access.Consent{RevokedAt: now.Add(-time.Minute)}},
	}
	for _, tt := range tests {
		f := cancelFixture(tt.at)
		if tt.consent != nil {
			f.consents.state[patientA+clinicA] = *tt.consent
		}
		if tt.closed {
			f.repo.appts[0].Status = appointment.StatusCancelled
		}
		_, err := f.svc.Cancel(t.Context(), tt.caller, tt.id)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Cancel() error = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.jobs.enqueued) != 0 || len(f.slots.released) != 0 || slices.Contains(f.repo.calls, "Cancel") {
			t.Errorf("%s: enqueued %v, released %v, calls %v; want nothing changed", tt.name, f.jobs.enqueued, f.slots.released, f.repo.calls)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestCancel_ASecondCancellationIsClosed(t *testing.T) {
	f := cancelFixture(start)
	if _, err := f.svc.Cancel(t.Context(), admin, apptA); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Cancel(t.Context(), admin, apptA); !errors.Is(err, appointment.ErrClosed) {
		t.Errorf("a second cancellation = %v, want ErrClosed", err)
	}
}

func TestSetStatus_ThePractitionerMarksTheirAppointmentOnceItHasStarted(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusFulfilled, appointment.StatusNoShow} {
		f := cancelFixture(now)
		got, err := f.svc.SetStatus(t.Context(), pract, apptA, status)
		if err != nil || got.Status != status || got.VersionID != 2 || !got.LastUpdated.Equal(now) {
			t.Errorf("SetStatus(%s) at the start = %+v, %v; want it at version 2", status, got, err)
		}
		if len(f.slots.released) != 0 || len(f.jobs.enqueued) != 0 {
			t.Errorf("SetStatus(%s): released %v, enqueued %v; want neither", status, f.slots.released, f.jobs.enqueued)
		}
		checkEvents(t, "SetStatus", f.audit, event(pract, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
	}
}

// After a revocation the practitioner still records how a past visit went: it is the care record
// (docs/booking/SPEC.md "Past and future after revocation").
func TestSetStatus_ThePractitionerMarksAPastAppointmentAfterARevocation(t *testing.T) {
	f := cancelFixture(now.Add(-time.Hour))
	f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Minute)}
	got, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusNoShow)
	if err != nil || got.Status != appointment.StatusNoShow || got.VersionID != 2 {
		t.Fatalf("SetStatus() after the revocation = %+v, %v; want a no-show at version 2", got, err)
	}
	checkEvents(t, "SetStatus", f.audit, event(pract, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestSetStatus_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		at      time.Time
		want    error
		consent *access.Consent // patient A's with clinic A, when changed
		name    string
		caller  appointment.Caller
		closed  bool
	}{
		{name: "a second before the start", caller: pract, at: now.Add(time.Second), want: appointment.ErrNotStarted},
		{name: "a second time", caller: pract, at: now.Add(-time.Hour), closed: true, want: appointment.ErrClosed},
		{name: "the admin", caller: admin, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
		{name: "the patient", caller: me, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
		{name: "another practitioner", caller: pract2, at: now.Add(-time.Hour), want: appointment.ErrNotFound},
		{name: "another clinic's admin", caller: adminB, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
		{name: "one that started after the revocation", caller: pract, at: now.Add(-time.Second), want: appointment.ErrNotFound,
			consent: &access.Consent{RevokedAt: now.Add(-time.Minute)}},
	}
	for _, tt := range tests {
		f := cancelFixture(tt.at)
		if tt.consent != nil {
			f.consents.state[patientA+clinicA] = *tt.consent
		}
		if tt.closed {
			f.repo.appts[0].Status = appointment.StatusFulfilled
		}
		_, err := f.svc.SetStatus(t.Context(), tt.caller, apptA, appointment.StatusFulfilled)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: SetStatus() error = %v, want %v", tt.name, err, tt.want)
		}
		if slices.ContainsFunc(f.repo.calls, func(c string) bool { return len(c) > 9 && c[:10] == "SetStatus:" }) {
			t.Errorf("%s: calls %v, want no change", tt.name, f.repo.calls)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestSetStatus_RefusesAnyOtherStatusBeforeAnyAccess(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusBooked, appointment.StatusCancelled, "arrived", ""} {
		f := cancelFixture(now.Add(-time.Hour))
		if _, err := f.svc.SetStatus(t.Context(), pract, apptA, status); !errors.Is(err, appointment.ErrInvalidStatus) {
			t.Errorf("SetStatus(%q) = %v, want ErrInvalidStatus", status, err)
		}
		if len(f.repo.calls) != 0 || len(f.audit.events) != 0 {
			t.Errorf("SetStatus(%q): calls %v, events %v; want none", status, f.repo.calls, f.audit.events)
		}
	}
}

func TestCancelForRevokedConsent_CancelsTheFutureAppointmentsBookedBeforeTheRevocationOnce(t *testing.T) {
	revoked := now.Add(-time.Minute)
	pastOne := booked("e5000000-0000-4000-8000-0000000000a5", patientA, roleA1, clinicA, now.Add(-2*time.Hour))
	startedAfter := booked("e6000000-0000-4000-8000-0000000000a6", patientA, roleA2, clinicA, revoked.Add(30*time.Second)) // started since
	rebooked := booked("e7000000-0000-4000-8000-0000000000a7", patientA, roleA1, clinicA, start.Add(5*time.Hour))
	rebooked.Created = revoked.Add(time.Second) // booked after granting again
	f := newFixture(
		booked(apptA, patientA, roleA1, clinicA, start),
		booked(apptA2, patientA, roleA2, clinicA, start.Add(2*time.Hour)),
		booked(apptB, patientB, roleA1, clinicA, start.Add(time.Hour)),
		booked(apptX, patientA, roleB, clinicB, start),
		pastOne, startedAfter, rebooked,
	)
	f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: revoked}

	n, err := f.svc.CancelForRevokedConsent(t.Context(), patientA, clinicA)
	if err != nil || n != 3 {
		t.Fatalf("CancelForRevokedConsent() = %d, %v; want 3", n, err)
	}
	var cancelled []string
	for _, a := range f.repo.appts {
		if a.Status == appointment.StatusCancelled {
			cancelled = append(cancelled, a.ID)
			if a.Reason != appointment.ReasonPatient || a.CancelledBy != "" || !a.CancelledAt.Equal(now) {
				t.Errorf("%s cancelled as %+v, want reason pat, by nobody, now", a.ID, a)
			}
		}
	}
	slices.Sort(cancelled)
	if want := []string{apptA, apptA2, startedAfter.ID}; !slices.Equal(cancelled, want) {
		t.Errorf("cancelled %v, want %v", cancelled, want)
	}
	if f.repo.patient != patientA || f.repo.clinic != clinicA || len(f.slots.released) != 3 {
		t.Errorf("in patient %q and clinic %q, released %v; want patient A's scope with clinic A and three slots", f.repo.patient, f.repo.clinic, f.slots.released)
	}
	if want := []string{"cancelled " + apptA + " " + clinicA, "cancelled " + apptA2 + " " + clinicA, "cancelled " + startedAfter.ID + " " + clinicA}; !slices.Equal(sortedCopy(f.jobs.enqueued), sortedCopy(want)) {
		t.Errorf("enqueued %v, want one cancellation e-mail each", f.jobs.enqueued)
	}
	want := []auditevent.AuditEvent{}
	for _, id := range []string{apptA, apptA2, startedAfter.ID} {
		want = append(want, jobEvent("consent_revoked", clinicA, auditevent.CodeUpdate, "Appointment", id))
	}
	if !slices.Equal(sortedEvents(f.audit.events), sortedEvents(want)) {
		t.Errorf("audit events %+v,\nwant %+v", f.audit.events, want)
	}

	// A retry finds nothing left to cancel.
	f.audit.events, f.jobs.enqueued, f.slots.released = nil, nil, nil
	if n, err := f.svc.CancelForRevokedConsent(t.Context(), patientA, clinicA); err != nil || n != 0 || len(f.jobs.enqueued) != 0 || len(f.audit.events) != 0 {
		t.Errorf("the retry = %d, %v, enqueued %v, events %v; want nothing", n, err, f.jobs.enqueued, f.audit.events)
	}
}

func TestCancelForRevokedConsent_DoesNothingWithoutARevocation(t *testing.T) {
	f := cancelFixture(start)
	if n, err := f.svc.CancelForRevokedConsent(t.Context(), patientA, clinicA); err != nil || n != 0 || len(f.jobs.enqueued) != 0 {
		t.Errorf("with an active consent never revoked = %d, %v, %v; want nothing", n, err, f.jobs.enqueued)
	}
}

func TestCancelForRevokedConsent_RollsEverythingBackOnAFailure(t *testing.T) {
	f := cancelFixture(start)
	f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Minute)}
	f.jobs.err = errBackend
	if _, err := f.svc.CancelForRevokedConsent(t.Context(), patientA, clinicA); !errors.Is(err, errBackend) {
		t.Fatalf("error = %v, want the jobs' error", err)
	}
	for _, a := range f.repo.appts {
		if a.Status != appointment.StatusBooked {
			t.Errorf("%s is %s after the rollback, want booked", a.ID, a.Status)
		}
	}
	if len(f.slots.released) != 0 || len(f.audit.events) != 0 {
		t.Errorf("released %v, events %v after the rollback; want none", f.slots.released, f.audit.events)
	}
}

func sortedCopy(s []string) []string {
	s = slices.Clone(s)
	slices.Sort(s)
	return s
}

func sortedEvents(e []auditevent.AuditEvent) []auditevent.AuditEvent {
	e = slices.Clone(e)
	slices.SortFunc(e, func(a, b auditevent.AuditEvent) int {
		if a.EntityReference < b.EntityReference {
			return -1
		}
		if a.EntityReference > b.EntityReference {
			return 1
		}
		return 0
	})
	return e
}
