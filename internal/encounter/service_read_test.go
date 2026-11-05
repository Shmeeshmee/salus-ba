package encounter_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// Read gives the encounter to whoever access.Appointment gives the name or more of its appointment, in their
// own scope, audited.
func TestRead_GivesTheEncounterToWhoeverMaySeeItsAppointment(t *testing.T) {
	for _, tt := range []struct {
		scope  [2]string
		name   string
		caller appointment.Caller
	}{
		{name: "its patient", caller: me, scope: [2]string{patientA, ""}},
		{name: "the practitioner of its role", caller: pract, scope: [2]string{"", clinicA}},
		{name: "the clinic's admin", caller: admin, scope: [2]string{"", clinicA}},
	} {
		f := started()
		got, err := f.svc.Read(t.Context(), tt.caller, encounterOf(apptA))
		if err != nil || got != f.repo.encounters[0] {
			t.Errorf("%s: Read() = %+v, %v; want encounter A", tt.name, got, err)
		}
		if [2]string{f.repo.patient, f.repo.clinic} != tt.scope {
			t.Errorf("%s: read in patient %q, clinic %q; want %q", tt.name, f.repo.patient, f.repo.clinic, tt.scope)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, encounterOf(apptA), auditevent.OutcomeSuccess))
	}
}

// After a revocation the practitioner sees a past appointment at the name level: its encounter too.
func TestRead_GivesTheEncounterAtTheNameLevel(t *testing.T) {
	f := started()
	att := f.appts.attendance[practAcc+apptA]
	att.Level = access.Name
	f.appts.attendance[practAcc+apptA] = att
	if got, err := f.svc.Read(t.Context(), pract, encounterOf(apptA)); err != nil || got.ID != encounterOf(apptA) {
		t.Errorf("Read() = %+v, %v; want encounter A", got, err)
	}
}

// Anyone else gets ErrNotFound, audited without the entity: another patient, a patient without a profile,
// another practitioner whose access does not reach it, another clinic, an unknown id.
func TestRead_IsNotFoundForAnyoneElse(t *testing.T) {
	for _, tt := range []struct {
		name   string
		id     string
		caller appointment.Caller
	}{
		{name: "another patient", caller: other, id: encounterOf(apptA)},
		{name: "no profile", caller: noProfile, id: encounterOf(apptA)},
		{name: "another clinic's admin", caller: adminB, id: encounterOf(apptA)},
		{name: "an unknown id", caller: admin, id: unknownID},
		{name: "no access to the appointment", caller: pract, id: encounterOf(apptB)},
	} {
		f := started()
		if _, err := f.svc.Read(t.Context(), tt.caller, tt.id); !errors.Is(err, encounter.ErrNotFound) {
			t.Errorf("%s: Read() = %v, want ErrNotFound", tt.name, err)
		}
		checkEvents(t, tt.name, f.audit, event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure))
	}
}

func TestRead_FailsWithTheBackend(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"repository":   func(f *fixture) { f.repo.err = errBackend },
		"appointments": func(f *fixture) { f.appts.err = errBackend },
		"audit":        func(f *fixture) { f.audit.err = errBackend },
	} {
		f := started()
		fail(f)
		if _, err := f.svc.Read(t.Context(), me, encounterOf(apptA)); !errors.Is(err, errBackend) {
			t.Errorf("%s: Read() = %v, want its error", name, err)
		}
	}
}

func ids(all []encounter.Encounter) []string {
	out := make([]string, 0, len(all))
	for _, e := range all {
		out = append(out, e.ID)
	}
	return out
}

// Search scopes like the Appointment search: a patient only their own (whatever patient they name), a
// practitioner only their own role's, the admin the clinic's; by appointment, patient and practitioner. Each
// returned encounter is audited; a search that returns none writes one event without an entity.
func TestSearch_ScopesToTheCaller(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller appointment.Caller
		filter encounter.Filter
		want   []string
	}{
		{"the patient", me, encounter.Filter{}, []string{encounterOf(apptA), encounterOf(apptA2)}},
		{"the patient naming another", me, encounter.Filter{PatientID: patientB}, []string{encounterOf(apptA), encounterOf(apptA2)}},
		{"the patient by appointment", me, encounter.Filter{AppointmentID: apptA2}, []string{encounterOf(apptA2)}},
		{"the patient by practitioner", me, encounter.Filter{PractitionerID: practA2}, []string{encounterOf(apptA2)}},
		{"another patient", other, encounter.Filter{}, []string{encounterOf(apptB)}},
		{"no profile", noProfile, encounter.Filter{}, nil},
		{"the practitioner", pract, encounter.Filter{}, []string{encounterOf(apptA)}},
		{"the practitioner naming themselves", pract, encounter.Filter{PractitionerID: practA1}, []string{encounterOf(apptA)}},
		{"the practitioner by patient", pract, encounter.Filter{PatientID: patientA}, []string{encounterOf(apptA)}},
		{"the second practitioner", pract2, encounter.Filter{}, []string{encounterOf(apptA2)}},
		{"the admin", admin, encounter.Filter{}, []string{encounterOf(apptA), encounterOf(apptA2)}},
		{"the admin by practitioner", admin, encounter.Filter{PractitionerID: practA2}, []string{encounterOf(apptA2)}},
		{"the admin by appointment", admin, encounter.Filter{AppointmentID: apptA}, []string{encounterOf(apptA)}},
		{"another clinic's admin", adminB, encounter.Filter{}, nil},
	} {
		f := started()
		got, err := f.svc.Search(t.Context(), tt.caller, tt.filter)
		if err != nil || !slices.Equal(ids(got), tt.want) {
			t.Errorf("%s: Search() = %v, %v; want %v", tt.name, ids(got), err, tt.want)
		}
		var want []auditevent.AuditEvent
		for _, id := range tt.want {
			want = append(want, event(tt.caller, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess))
		}
		if len(want) == 0 {
			want = append(want, event(tt.caller, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
		}
		checkEvents(t, tt.name, f.audit, want...)
	}
}

// A practitioner who names another practitioner is refused, audited.
func TestSearch_RefusesAPractitionerNamingAnother(t *testing.T) {
	f := started()
	if _, err := f.svc.Search(t.Context(), pract, encounter.Filter{PractitionerID: practA2}); !errors.Is(err, encounter.ErrForbidden) {
		t.Errorf("Search() = %v, want ErrForbidden", err)
	}
	checkEvents(t, "Search", f.audit, event(pract, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure))
}

func TestSearch_FailsWithTheBackend(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"repository":   func(f *fixture) { f.repo.err = errBackend },
		"appointments": func(f *fixture) { f.appts.err = errBackend },
		"audit":        func(f *fixture) { f.audit.err = errBackend },
	} {
		f := started()
		fail(f)
		if _, err := f.svc.Search(t.Context(), admin, encounter.Filter{}); !errors.Is(err, errBackend) {
			t.Errorf("%s: Search() = %v, want its error", name, err)
		}
	}
}
