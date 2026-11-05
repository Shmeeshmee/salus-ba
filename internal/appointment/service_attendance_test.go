package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Attendance gives the consultation (videosession) the appointment with the caller's access facts, in the
// caller's scope, and audits nothing: the consultation audits its own decision.
func TestAttendance_GivesTheAppointmentWithTheCallersAccessFacts(t *testing.T) {
	tests := []struct {
		scope  [2]string // the patient and the clinic the fake repository was scoped to
		name   string
		caller appointment.Caller
		want   appointment.Attendance
	}{
		{name: "its patient", caller: me, scope: [2]string{patientA, ""},
			want: appointment.Attendance{PractitionerID: practA1, Level: access.Full, Profile: access.Full, ConsentActive: true, Patient: true}},
		{name: "the practitioner of its role", caller: pract, scope: [2]string{"", clinicA},
			want: appointment.Attendance{PractitionerID: practA1, Level: access.Full, Profile: access.Full, ConsentActive: true, Practitioner: true}},
		{name: "the clinic's admin", caller: admin, scope: [2]string{"", clinicA},
			want: appointment.Attendance{PractitionerID: practA1, Level: access.Full, Profile: access.Name, ConsentActive: true}},
	}
	for _, tt := range tests {
		f := cancelFixture(start)
		got, err := f.svc.Attendance(t.Context(), tt.caller, apptA)
		tt.want.Appointment = f.repo.appts[0]
		if err != nil || got != tt.want {
			t.Errorf("%s: Attendance() = %+v, %v;\nwant %+v", tt.name, got, err, tt.want)
		}
		if [2]string{f.repo.patient, f.repo.clinic} != tt.scope {
			t.Errorf("%s: scoped to patient %q, clinic %q; want %q", tt.name, f.repo.patient, f.repo.clinic, tt.scope)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("%s: audited %v, want nothing", tt.name, f.audit.events)
		}
	}
}

// After a revocation the patient keeps their appointment and its practitioner a past one at the name level,
// both with the consent inactive.
func TestAttendance_GivesTheInactiveConsentAfterARevocation(t *testing.T) {
	for _, c := range []appointment.Caller{me, pract} {
		f := cancelFixture(revokedAt.Add(-time.Hour))
		f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
		got, err := f.svc.Attendance(t.Context(), c, apptA)
		if err != nil || got.ConsentActive || got.Level == access.None || got.Patient != (c == me) || got.Practitioner != (c == pract) {
			t.Errorf("as %s: Attendance() = %+v, %v; want the appointment with the consent inactive", c.AccountID, got, err)
		}
	}
}

// Anyone the appointment is hidden from gets ErrNotFound, unaudited.
func TestAttendance_IsNotFoundForAnyoneElse(t *testing.T) {
	for _, c := range []appointment.Caller{other, noProfile, pract2, adminB} {
		f := cancelFixture(start)
		if _, err := f.svc.Attendance(t.Context(), c, apptA); !errors.Is(err, appointment.ErrNotFound) {
			t.Errorf("as %s: Attendance() = %v, want ErrNotFound", c.AccountID, err)
		}
		if len(f.audit.events) != 0 {
			t.Errorf("as %s: audited %v, want nothing", c.AccountID, f.audit.events)
		}
	}
}

func TestAttendance_FailsWithTheBackend(t *testing.T) {
	f := cancelFixture(start)
	f.consents.err = errBackend
	if _, err := f.svc.Attendance(t.Context(), me, apptA); !errors.Is(err, errBackend) {
		t.Errorf("Attendance() = %v, want the consent's error", err)
	}
}

// inClinic runs fn in a transaction scoped to the clinic, as the consultation's paths are (the webhook, the
// consultation's jobs).
func inClinic(t *testing.T, f *fixture, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return fakeTransactor{f: f}.WithTx(t.Context(), func(ctx context.Context) error {
		if err := f.repo.SetClinic(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// Lookup gives the consultation's lifecycle the appointment in the caller's transaction and scope,
// unaudited: the encounter audits its own write.
func TestLookup_GivesTheAppointmentInTheCallersScopeUnaudited(t *testing.T) {
	f := cancelFixture(start)
	var got appointment.Appointment
	err := inClinic(t, f, clinicA, func(ctx context.Context) error {
		var err error
		got, err = f.svc.Lookup(ctx, apptA)
		return err
	})
	if err != nil || got != f.repo.appts[0] {
		t.Errorf("Lookup() = %+v, %v; want appointment A", got, err)
	}
	err = inClinic(t, f, clinicB, func(ctx context.Context) error {
		_, err := f.svc.Lookup(ctx, apptA)
		return err
	})
	if !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("Lookup() in clinic B = %v, want ErrNotFound", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing", f.audit.events)
	}
}

// The consultation's outcome: a booked appointment becomes fulfilled when the patient joined and noshow when
// they did not, audited as an update by the agent in the appointment's clinic.
func TestRecordConsultationOutcome_MarksABookedAppointmentByWhetherThePatientJoined(t *testing.T) {
	for _, tt := range []struct {
		want   appointment.Status
		joined bool
	}{{appointment.StatusFulfilled, true}, {appointment.StatusNoShow, false}} {
		f := cancelFixture(now.Add(-time.Hour))
		err := inClinic(t, f, clinicA, func(ctx context.Context) error {
			return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA, tt.joined)
		})
		if got := f.repo.appts[0]; err != nil || got.Status != tt.want || !got.LastUpdated.Equal(now) || got.VersionID != 2 {
			t.Errorf("joined %t: RecordConsultationOutcome() = %v, appointment %+v; want %s now at version 2", tt.joined, err, got, tt.want)
		}
		checkEvents(t, "RecordConsultationOutcome", f.audit, jobEvent("consultation_window_close", clinicA, auditevent.CodeUpdate, "Appointment", apptA))
	}
}

// A practitioner who marked it by hand wins, and a cancellation stays a cancellation: nothing changes and
// nothing is audited.
func TestRecordConsultationOutcome_LeavesAnyOtherStatus(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusFulfilled, appointment.StatusNoShow, appointment.StatusCancelled} {
		f := cancelFixture(now.Add(-time.Hour))
		f.repo.appts[0].Status = status
		before := f.repo.appts[0]
		err := inClinic(t, f, clinicA, func(ctx context.Context) error {
			return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA, false)
		})
		if err != nil || f.repo.appts[0] != before {
			t.Errorf("%s: RecordConsultationOutcome() = %v, appointment %+v; want it unchanged", status, err, f.repo.appts[0])
		}
		checkEvents(t, string(status), f.audit)
	}
}

func TestRecordConsultationOutcome_FailsWithTheBackend(t *testing.T) {
	f := cancelFixture(now.Add(-time.Hour))
	f.repo.err = errBackend
	err := inClinic(t, f, clinicA, func(ctx context.Context) error {
		return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA, true)
	})
	if !errors.Is(err, errBackend) {
		t.Errorf("RecordConsultationOutcome() = %v, want the repository's error", err)
	}
}
