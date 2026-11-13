package appointment_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

func checkEvents(t *testing.T, name string, audit *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(audit.events, want) {
		t.Errorf("%s: audit events %+v,\nwant %+v", name, audit.events, want)
	}
}

func TestBook_HoldsTheSlotStoresTheAppointmentEnqueuesItsEmailsAndAuditsIt(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Book(t.Context(), me, slotFree)
	want := appointment.Appointment{
		ID: newID, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA, SlotID: slotFree,
		Start: start, End: start.Add(30 * time.Minute), Created: now, LastUpdated: now, Status: appointment.StatusBooked, VersionID: 1,
		BookedVia: appointment.BookedDirect,
	}
	if err != nil || got != want {
		t.Fatalf("Book() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"SetPatient", "SetClinic", "Create"}; !slices.Equal(f.repo.calls, want) || f.repo.patient != patientA || f.repo.clinic != clinicA {
		t.Errorf("calls %v in patient %q and clinic %q, want %v in a transaction of patient A with clinic A", f.repo.calls, f.repo.patient, f.repo.clinic, want)
	}
	if !slices.Equal(f.slots.held, []string{slotFree}) {
		t.Errorf("held %v, want the slot", f.slots.held)
	}
	wantJobs := []string{"booked " + newID + " " + clinicA, "questionnaire_request " + newID + " " + clinicA, "reminder " + newID + " " + clinicA + " at " + start.Add(-24*time.Hour).Format(time.RFC3339)}
	if !slices.Equal(f.jobs.enqueued, wantJobs) {
		t.Errorf("enqueued %v,\nwant %v", f.jobs.enqueued, wantJobs)
	}
	checkEvents(t, "Book", f.audit, event(me, auditevent.CodeCreate, "Appointment", newID, auditevent.OutcomeSuccess))
}

func TestBook_LessThan24HoursAheadEnqueuesNoReminder(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Book(t.Context(), me, slotSoon); err != nil {
		t.Fatalf("Book() error = %v", err)
	}
	if want := []string{"booked " + newID + " " + clinicA, "questionnaire_request " + newID + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want only the confirmation and the questionnaire request %v", f.jobs.enqueued, want)
	}
}

func TestBook_RefusesAndAuditsEveryCaseOfRefusal(t *testing.T) {
	tests := []struct {
		want     error
		consents map[string]access.Consent
		name     string
		slot     string
		caller   appointment.Caller
		existing []appointment.Appointment
		held     []string
	}{
		{name: "staff", caller: admin, slot: slotFree, want: appointment.ErrForbidden},
		{name: "a patient without a profile", caller: noProfile, slot: slotFree, want: appointment.ErrProfileRequired},
		{name: "an unknown slot", caller: me, slot: "51000000-0000-4000-8000-0000000000ff", want: appointment.ErrSlotUnavailable},
		{name: "a slot that has started", caller: me, slot: slotPast, want: appointment.ErrSlotUnavailable},
		{name: "a slot of an unlisted clinic", caller: me, slot: slotU, want: appointment.ErrSlotUnavailable},
		{name: "a clinic never consented to", caller: me, slot: slotB, want: appointment.ErrConsentRequired},
		{name: "a revoked consent", caller: me, slot: slotB, want: appointment.ErrConsentRequired,
			consents: map[string]access.Consent{patientA + clinicB: {RevokedAt: revokedAt}}},
		{name: "a time the practitioner is booked at, after the rollback", caller: me, slot: slotFree, want: appointment.ErrSlotUnavailable,
			existing: []appointment.Appointment{booked("e1000000-0000-4000-8000-0000000000e1", patientB, roleA1, clinicA, start.Add(15*time.Minute))}},
		{name: "a slot held by another booking after the directory offered it", caller: me, slot: slotFree, want: appointment.ErrSlotUnavailable,
			held: []string{slotFree}},
	}
	for _, tt := range tests {
		f := newFixture(tt.existing...)
		f.slots.held = slices.Clone(tt.held)
		for k, v := range tt.consents {
			f.consents.state[k] = v
		}
		_, err := f.svc.Book(t.Context(), tt.caller, tt.slot)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Book() error = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.jobs.enqueued) != 0 || len(f.slots.held) != len(tt.held) || len(f.repo.appts) != len(tt.existing) {
			t.Errorf("%s: enqueued %v, held %v, %d appointments; want nothing changed", tt.name, f.jobs.enqueued, f.slots.held, len(f.repo.appts))
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestBook_FailsOnABackendErrorWithoutLeakingItsInputs(t *testing.T) {
	for name, breakIt := range map[string]func(f *fixture){
		"repository": func(f *fixture) { f.repo.err = errBackend },
		"consents":   func(f *fixture) { f.consents.err = errBackend },
		"clinics":    func(f *fixture) { f.clinics.err = errBackend },
		"jobs":       func(f *fixture) { f.jobs.err = errBackend },
		"audit":      func(f *fixture) { f.audit.err = errBackend },
		"slots":      func(f *fixture) { f.slots.err = errBackend },
	} {
		f := newFixture()
		breakIt(f)
		_, err := f.svc.Book(t.Context(), me, slotFree)
		if !errors.Is(err, errBackend) || strings.Contains(err.Error(), patientA) || strings.Contains(err.Error(), slotFree) {
			t.Errorf("%s: Book() error = %v, want the backend's error without ids", name, err)
		}
		if len(f.jobs.enqueued) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: enqueued %v, audited %v after the rollback; want nothing", name, f.jobs.enqueued, f.audit.events)
		}
	}
}
