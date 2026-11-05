package appointment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
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
