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

// correctionWindow is how long after the start a no-show or an attendance is corrected (Q6).
const correctionWindow = 30 * 24 * time.Hour

// Q6: the appointment's practitioner and the clinic's admin correct a no-show into an attendance and back, within 30
// days of the start: the appointment at its next version, a status_corrected row by staff in its history, the clinic's
// day refreshed, audited as an update.
func TestSetStatus_ThePractitionerAndTheAdminCorrectTheOutcomeWithin30Days(t *testing.T) {
	for _, tt := range []struct {
		from, to appointment.Status
		account  string
		caller   appointment.Caller
	}{
		{appointment.StatusNoShow, appointment.StatusFulfilled, practAcc, pract},
		{appointment.StatusFulfilled, appointment.StatusNoShow, practAcc, pract},
		{appointment.StatusNoShow, appointment.StatusFulfilled, adminAcc, admin},
		{appointment.StatusFulfilled, appointment.StatusNoShow, adminAcc, admin},
	} {
		f, _ := noShowFixture(now.Add(-correctionWindow), 0, 180)
		f.repo.appts[0].Status = tt.from
		got, err := f.svc.SetStatus(t.Context(), tt.caller, apptA, tt.to)
		if err != nil || got.Status != tt.to || got.VersionID != 2 || !got.LastUpdated.Equal(now) {
			t.Fatalf("%s to %s by %s = %+v, %v; want it at version 2", tt.from, tt.to, tt.account, got, err)
		}
		want := []appointment.Change{{ID: "c0000001-0000-4000-8000-000000000000", At: now, AppointmentID: apptA, Kind: appointment.ChangeStatusCorrected,
			Via: appointment.ViaStaff, ByAccountID: tt.account, FromStatus: tt.from, ToStatus: tt.to}}
		if !slices.Equal(f.repo.changes, want) {
			t.Errorf("history %+v,\nwant %+v", f.repo.changes, want)
		}
		if len(f.stats.refreshes) != 1 || len(f.jobs.enqueued) != 0 || len(f.slots.released) != 0 {
			t.Errorf("refreshes %v, jobs %v, released %v; want the clinic's day refreshed only", f.stats.refreshes, f.jobs.enqueued, f.slots.released)
		}
		checkEvents(t, "SetStatus", f.audit, event(tt.caller, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
	}
}

// After a revocation the correction of a past visit stays possible: it is the care record.
func TestSetStatus_ACorrectionOfAPastVisitAfterARevocation(t *testing.T) {
	for _, c := range []appointment.Caller{pract, admin} {
		f, _ := noShowFixture(now.Add(-48*time.Hour), 0, 180)
		f.repo.appts[0].Status = appointment.StatusNoShow
		f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Hour)}
		if got, err := f.svc.SetStatus(t.Context(), c, apptA, appointment.StatusFulfilled); err != nil || got.Status != appointment.StatusFulfilled {
			t.Errorf("%s: SetStatus() after the revocation = %+v, %v", c.AccountID, got, err)
		}
	}
}

// A correction is refused, audited and leaves no history: after 30 days, to anyone but the practitioner and the
// admin, of a cancelled appointment, of another clinic's.
func TestSetStatus_RefusesAndAuditsEveryCorrectionItMayNotMake(t *testing.T) {
	for _, tt := range []struct {
		want   error
		name   string
		from   appointment.Status
		at     time.Time
		caller appointment.Caller
	}{
		{name: "a second after 30 days", caller: pract, from: appointment.StatusNoShow, at: now.Add(-correctionWindow - time.Second), want: appointment.ErrCorrectionWindowClosed},
		{name: "the admin after 30 days", caller: admin, from: appointment.StatusNoShow, at: now.Add(-correctionWindow - time.Second), want: appointment.ErrCorrectionWindowClosed},
		{name: "another practitioner", caller: pract2, from: appointment.StatusNoShow, at: now.Add(-time.Hour), want: appointment.ErrNotFound},
		{name: "the patient", caller: me, from: appointment.StatusNoShow, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
		{name: "another clinic's admin", caller: adminB, from: appointment.StatusNoShow, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
		{name: "a cancelled one", caller: pract, from: appointment.StatusCancelled, at: now.Add(-time.Hour), want: appointment.ErrClosed},
		{name: "the admin on a cancelled one", caller: admin, from: appointment.StatusCancelled, at: now.Add(-time.Hour), want: appointment.ErrForbidden},
	} {
		f, _ := noShowFixture(tt.at, 0, 180)
		f.repo.appts[0].Status = tt.from
		_, err := f.svc.SetStatus(t.Context(), tt.caller, apptA, appointment.StatusFulfilled)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: SetStatus() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.repo.changes) != 0 || slices.ContainsFunc(f.repo.calls, func(c string) bool { return len(c) > 14 && c[:14] == "CorrectStatus:" }) {
			t.Errorf("%s: history %v, calls %v; want no change", tt.name, f.repo.changes, f.repo.calls)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
	}
}

// Corrected meanwhile by someone else: the correction finds it in another status and is closed.
func TestSetStatus_ACorrectionRacedByAnotherIsClosed(t *testing.T) {
	f, _ := noShowFixture(now.Add(-time.Hour), 0, 180)
	f.repo.appts[0].Status = appointment.StatusNoShow
	p := f.svc.Ports
	p.NoShows = racedNoShows{fakeNoShows{r: f.repo}}
	f.svc = appointment.NewService(p)
	if _, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusFulfilled); !errors.Is(err, appointment.ErrClosed) {
		t.Errorf("SetStatus() = %v, want ErrClosed", err)
	}
	if len(f.repo.changes) != 0 {
		t.Errorf("history %v, want none", f.repo.changes)
	}
}
