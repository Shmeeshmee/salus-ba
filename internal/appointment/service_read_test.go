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

// minimal is a as a Name-level viewer sees it: the times, the status and the participants.
func minimal(a appointment.Appointment) appointment.Appointment {
	return appointment.Appointment{
		ID: a.ID, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID, OrganizationID: a.OrganizationID,
		Start: a.Start, End: a.End, Status: a.Status, VersionID: a.VersionID, LastUpdated: a.LastUpdated,
	}
}

func TestRead_GivesThePatientTheirAppointmentWithTheirName(t *testing.T) {
	f := cancelFixture(start)
	got, err := f.svc.Read(t.Context(), me, apptA)
	if want := (appointment.Seen{Appointment: f.repo.appts[0], Patient: nameA, Level: access.Full}); err != nil || got != want {
		t.Fatalf("Read() = %+v, %v;\nwant %+v", got, err, want)
	}
	if f.repo.patient != patientA {
		t.Errorf("read in patient %q's scope, want patient A's", f.repo.patient)
	}
	checkEvents(t, "Read", f.audit, event(me, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeRead, "Patient", patientA, auditevent.OutcomeSuccess))
}

func TestRead_GivesStaffWhatAccessAllows(t *testing.T) {
	revoked := now.Add(-time.Minute)
	tests := []struct {
		at      time.Time
		consent access.Consent
		name    string
		caller  appointment.Caller
		level   access.Level
	}{
		{name: "the practitioner, with consent", caller: pract, at: start, consent: access.Consent{Active: true}, level: access.Full},
		{name: "the admin, with consent", caller: admin, at: start, consent: access.Consent{Active: true}, level: access.Full},
		{name: "the practitioner, a past one after the revocation", caller: pract, at: now.Add(-time.Hour), consent: access.Consent{RevokedAt: revoked}, level: access.Name},
		{name: "the admin, a past one after the revocation", caller: admin, at: now.Add(-time.Hour), consent: access.Consent{RevokedAt: revoked}, level: access.Name},
	}
	for _, tt := range tests {
		f := cancelFixture(tt.at)
		f.consents.state[patientA+clinicA] = tt.consent
		got, err := f.svc.Read(t.Context(), tt.caller, apptA)
		want := appointment.Seen{Appointment: f.repo.appts[0], Patient: nameA, Level: tt.level}
		if tt.level == access.Name {
			want.Appointment = minimal(want.Appointment)
		}
		if err != nil || got != want {
			t.Errorf("%s: Read() = %+v, %v;\nwant %+v", tt.name, got, err, want)
			continue
		}
		if f.repo.patient != "" || f.repo.clinic != clinicA {
			t.Errorf("%s: read in patient %q, clinic %q; want clinic A's scope", tt.name, f.repo.patient, f.repo.clinic)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess),
			event(tt.caller, auditevent.CodeRead, "Patient", patientA, auditevent.OutcomeSuccess))
	}
}

func TestRead_HidesTheAppointmentFromEveryoneElseAndAuditsIt(t *testing.T) {
	tests := []struct {
		consent *access.Consent
		name    string
		id      string
		caller  appointment.Caller
	}{
		{name: "another patient", caller: other, id: apptA},
		{name: "a patient without a profile", caller: noProfile, id: apptA},
		{name: "another practitioner", caller: pract2, id: apptA},
		{name: "another clinic", caller: adminB, id: apptA},
		{name: "an unknown one", caller: admin, id: "e9000000-0000-4000-8000-0000000000e9"},
		{name: "a future one after the revocation", caller: pract, id: apptA, consent: &access.Consent{RevokedAt: now.Add(-time.Minute)}},
	}
	for _, tt := range tests {
		f := cancelFixture(start)
		if tt.consent != nil {
			f.consents.state[patientA+clinicA] = *tt.consent
		}
		if _, err := f.svc.Read(t.Context(), tt.caller, tt.id); !errors.Is(err, appointment.ErrNotFound) {
			t.Errorf("%s: Read() error = %v, want ErrNotFound", tt.name, err)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestSearch_APatientFindsOnlyTheirOwnAndEachIsAudited(t *testing.T) {
	f := cancelFixture(start)
	got, err := f.svc.Search(t.Context(), me, appointment.Filter{PatientID: patientB})
	if err != nil || len(got) != 3 {
		t.Fatalf("Search() = %+v, %v; want patient A's three, whatever patient filter is given", got, err)
	}
	var ids []string
	var want []auditevent.AuditEvent
	for _, s := range got {
		ids = append(ids, s.Appointment.ID)
		if s.Patient != nameA || s.Level != access.Full {
			t.Errorf("%s: %+v, want Full with the name", s.Appointment.ID, s)
		}
		want = append(want, event(me, auditevent.CodeSearchType, "Appointment", s.Appointment.ID, auditevent.OutcomeSuccess),
			event(me, auditevent.CodeSearchType, "Patient", patientA, auditevent.OutcomeSuccess))
	}
	if !slices.Equal(sortedCopy(ids), sortedCopy([]string{apptA, apptA2, apptX})) {
		t.Errorf("found %v, want patient A's", ids)
	}
	checkEvents(t, "Search", f.audit, want...)
}

func TestSearch_APractitionerSearchesOnlyTheirOwnRole(t *testing.T) {
	f := cancelFixture(start)
	f.consents.state[patientB+clinicA] = access.Consent{Active: true}
	got, err := f.svc.Search(t.Context(), pract, appointment.Filter{PractitionerRoleID: roleA1, From: start, To: start.Add(90 * time.Minute)})
	if err != nil || len(got) != 2 || got[0].Appointment.ID != apptA || got[1].Appointment.ID != apptB || got[1].Patient.Given != "Bo" {
		t.Fatalf("Search() = %+v, %v; want apptA and apptB with their names", got, err)
	}
	for name, filter := range map[string]appointment.Filter{"no role": {}, "another practitioner's role": {PractitionerRoleID: roleA2}, "another clinic's role": {PractitionerRoleID: roleB}} {
		f := cancelFixture(start)
		if _, err := f.svc.Search(t.Context(), pract, filter); !errors.Is(err, appointment.ErrForbidden) {
			t.Errorf("%s: Search() = %v, want ErrForbidden", name, err)
		}
		checkEvents(t, name, f.audit, event(pract, auditevent.CodeSearchType, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestSearch_TheAdminSearchesTheClinicAsAccessAllows(t *testing.T) {
	revoked := now.Add(-time.Minute)
	past := booked("e5000000-0000-4000-8000-0000000000a5", patientB, roleA1, clinicA, now.Add(-time.Hour))
	f := newFixture(booked(apptA, patientA, roleA1, clinicA, start), booked(apptB, patientB, roleA1, clinicA, start.Add(time.Hour)), past,
		booked(apptX, patientA, roleB, clinicB, start))
	f.consents.state[patientB+clinicA] = access.Consent{RevokedAt: revoked}

	got, err := f.svc.Search(t.Context(), admin, appointment.Filter{Status: appointment.StatusBooked})
	if err != nil || len(got) != 2 {
		t.Fatalf("Search() = %+v, %v; want patient B's past one and patient A's", got, err)
	}
	if got[0].Appointment != minimal(past) || got[0].Level != access.Name || got[0].Patient.Given != "Bo" {
		t.Errorf("patient B's past appointment after the revocation = %+v, want the minimal form with the name", got[0])
	}
	if got[1].Appointment.ID != apptA || got[1].Level != access.Full {
		t.Errorf("patient A's = %+v, want Full", got[1])
	}
}

func TestSearch_FindingNoneAuditsOneEvent(t *testing.T) {
	f := cancelFixture(start)
	got, err := f.svc.Search(t.Context(), admin, appointment.Filter{Status: appointment.StatusNoShow})
	if err != nil || len(got) != 0 {
		t.Fatalf("Search() = %+v, %v; want none", got, err)
	}
	checkEvents(t, "Search", f.audit, event(admin, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess))
}

func TestNotice_GivesTheJobTheAppointmentAndThePatientsAccountAudited(t *testing.T) {
	f := cancelFixture(start)
	f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Minute)} // cancelled by the revocation: still mailed
	got, err := f.svc.Notice(t.Context(), "appointment_cancelled_email", clinicA, apptA)
	if err != nil || got.Appointment != f.repo.appts[0] || got.AccountID != accountA {
		t.Fatalf("Notice() = %+v, %v; want the appointment and patient A's account", got, err)
	}
	if want := []string{"SetClinic", "SetNotice", "ByID"}; !slices.Equal(f.repo.calls, want) || f.repo.notice != apptA {
		t.Errorf("calls %v with notice %q, want %v for the appointment", f.repo.calls, f.repo.notice, want)
	}
	checkEvents(t, "Notice", f.audit, jobEvent("appointment_cancelled_email", clinicA, auditevent.CodeRead, "Appointment", apptA),
		jobEvent("appointment_cancelled_email", clinicA, auditevent.CodeRead, "Patient", patientA))

	f = cancelFixture(start)
	if _, err := f.svc.Notice(t.Context(), "appointment_booked_email", clinicB, apptA); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("Notice() of another clinic's appointment = %v, want ErrNotFound", err)
	}
	failed := jobEvent("appointment_booked_email", clinicB, auditevent.CodeRead, "", "")
	failed.Outcome = auditevent.OutcomeMinorFailure
	checkEvents(t, "Notice of another clinic's", f.audit, failed)
}

func TestFacts_GiveThePatientsVisitsInTheCallersScopeWithoutAnAudit(t *testing.T) {
	f := cancelFixture(start)
	f.repo.clinic = clinicA
	got, err := appointment.NewFacts(f.repo).Visits(t.Context(), patientA, clinicA)
	want := []access.Visit{{PractitionerRole: roleA1, Start: start}, {PractitionerRole: roleA2, Start: start.Add(2 * time.Hour)}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Visits() = %+v, %v; want %+v", got, err, want)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing", f.audit.events)
	}
	f.repo.err = errBackend
	if _, err := appointment.NewFacts(f.repo).Visits(t.Context(), patientA, clinicA); !errors.Is(err, errBackend) {
		t.Errorf("Visits() on a failing repository = %v", err)
	}
}

func TestReadAndSearch_FailOnABackendError(t *testing.T) {
	for name, breakIt := range map[string]func(f *fixture){
		"repository": func(f *fixture) { f.repo.err = errBackend },
		"consents":   func(f *fixture) { f.consents.err = errBackend },
		"patients":   func(f *fixture) { f.patients.err = errBackend },
		"roles":      func(f *fixture) { f.roles.err = errBackend },
		"audit":      func(f *fixture) { f.audit.err = errBackend },
	} {
		f := cancelFixture(start)
		breakIt(f)
		if _, err := f.svc.Read(t.Context(), pract, apptA); !errors.Is(err, errBackend) {
			t.Errorf("%s: Read() = %v, want the backend's error", name, err)
		}
		if _, err := f.svc.Search(t.Context(), pract, appointment.Filter{PractitionerRoleID: roleA1}); !errors.Is(err, errBackend) {
			t.Errorf("%s: Search() = %v, want the backend's error", name, err)
		}
		if _, err := f.svc.Cancel(t.Context(), pract, apptA); !errors.Is(err, errBackend) && name != "patients" {
			t.Errorf("%s: Cancel() = %v, want the backend's error", name, err)
		}
		if name != "roles" {
			if _, err := f.svc.Notice(t.Context(), "appointment_booked_email", clinicA, apptA); !errors.Is(err, errBackend) && name != "consents" {
				t.Errorf("%s: Notice() = %v, want the backend's error", name, err)
			}
		}
	}
}
