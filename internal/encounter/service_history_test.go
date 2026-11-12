package encounter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// SAL-39: the patient's timeline lists their own visits at every clinic, newest first, after the page's position,
// in their own scope, unaudited (history.Timeline audits the page it returns); another patient's are not theirs.
func TestPatientHistory_GivesThePatientTheirOwnVisitsNewestFirst(t *testing.T) {
	f := started()
	got, err := f.svc.PatientHistory(t.Context(), me, "", keyset.Page{Limit: 10})
	if err != nil || len(got) != 2 || got[0].ID != encounterOf(apptA2) || got[1].ID != encounterOf(apptA) {
		t.Fatalf("PatientHistory() = %+v, %v; want A2 then A", got, err)
	}
	if f.repo.patient != patientA || f.repo.clinic != "" {
		t.Errorf("read in patient %q, clinic %q; want patient A's scope", f.repo.patient, f.repo.clinic)
	}
	checkEvents(t, "PatientHistory", f.audit)
}

// The page's position, range, clinic and limit reach the repository as asked.
func TestPatientHistory_AsksForThePage(t *testing.T) {
	f := started()
	p := keyset.Page{At: start.Add(time.Hour), Ref: "Encounter/" + encounterOf(apptA2), Limit: 5}
	got, err := f.svc.PatientHistory(t.Context(), me, clinicA, p)
	if err != nil || len(got) != 1 || got[0].ID != encounterOf(apptA) {
		t.Fatalf("PatientHistory() = %+v, %v; want A after A2", got, err)
	}
	if f.repo.history.page != p || f.repo.history.org != clinicA || f.repo.history.patient != patientA {
		t.Errorf("asked %+v, want %+v of clinic A for patient A", f.repo.history, p)
	}
}

// None is none, unaudited; a patient without a profile has none and is not scoped at all.
func TestPatientHistory_NoneIsUnaudited(t *testing.T) {
	for name, c := range map[string]appointment.Caller{"no visits": other, "no profile": noProfile} {
		f := newFixture()
		got, err := f.svc.PatientHistory(t.Context(), c, "", keyset.Page{Limit: 10})
		if err != nil || len(got) != 0 {
			t.Errorf("%s: PatientHistory() = %+v, %v; want none", name, got, err)
		}
		checkEvents(t, name, f.audit)
	}
}

// The timeline is the patient's own: staff get ErrForbidden and nothing is read.
func TestPatientHistory_IsForbiddenToStaff(t *testing.T) {
	for name, c := range map[string]appointment.Caller{"practitioner": pract, "admin": admin} {
		f := started()
		if _, err := f.svc.PatientHistory(t.Context(), c, "", keyset.Page{Limit: 10}); !errors.Is(err, encounter.ErrForbidden) {
			t.Errorf("%s: PatientHistory() = %v, want ErrForbidden", name, err)
		}
		if f.repo.history.patient != "" {
			t.Errorf("%s: read %+v, want nothing", name, f.repo.history)
		}
		checkEvents(t, name, f.audit)
	}
}

func TestPatientHistory_FailsWithTheBackend(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"repository": func(f *fixture) { f.repo.err = errBackend },
	} {
		f := started()
		fail(f)
		if _, err := f.svc.PatientHistory(t.Context(), me, "", keyset.Page{Limit: 10}); !errors.Is(err, errBackend) {
			t.Errorf("%s: PatientHistory() = %v, want its error", name, err)
		}
	}
}

// Own gives the patient's own encounters, at one clinic when asked, unaudited, for a service that lists what
// belongs to them and audits its own decision; staff and a patient without a profile get none.
func TestOwn_GivesThePatientsEncountersUnaudited(t *testing.T) {
	f := started()
	f.repo.encounters[1].OrganizationID = clinicB
	got, err := f.svc.Own(t.Context(), me, "")
	if err != nil || len(got) != 2 {
		t.Fatalf("Own() = %+v, %v; want patient A's two", got, err)
	}
	if got, err := f.svc.Own(t.Context(), me, clinicB); err != nil || len(got) != 1 || got[0].ID != encounterOf(apptA2) {
		t.Errorf("Own(clinic B) = %+v, %v; want A2", got, err)
	}
	for name, c := range map[string]appointment.Caller{"staff": pract, "no profile": noProfile} {
		if got, err := f.svc.Own(t.Context(), c, ""); err != nil || len(got) != 0 {
			t.Errorf("%s: Own() = %+v, %v; want none", name, got, err)
		}
	}
	checkEvents(t, "Own", f.audit)
	if len(f.repo.searched) != 0 {
		t.Errorf("Own searched %v, want the patient's own query: the general search is planned again on every call (finding 3)", f.repo.searched)
	}
	f.repo.err = errBackend
	if _, err := f.svc.Own(t.Context(), me, ""); !errors.Is(err, errBackend) {
		t.Errorf("Own() = %v, want its error", err)
	}
}
