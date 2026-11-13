package encounter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// Synthetic ids of the history's fixture: a third appointment of patient A with role A1, and two finished visits
// of patient A at clinic B.
const (
	apptA3    = "e4000000-0000-4000-8000-0000000000a3"
	roleB1    = "r3000000-0000-4000-8000-0000000000b1"
	sharedB   = "c5000000-0000-4000-8000-0000000000b1" // shared with clinic A
	unsharedB = "c6000000-0000-4000-8000-0000000000b2" // not shared
)

// fakeConsents gives each patient's consent state with each clinic (patient id + clinic id).
type fakeConsents struct {
	err   error
	state map[string]access.Consent
}

func (c *fakeConsents) State(ctx context.Context, patientID, org string) (access.Consent, error) {
	if err := inTx(ctx); err != nil {
		return access.Consent{}, err
	}
	return c.state[patientID+org], c.err
}

// fakeVisits gives each patient's appointments in each clinic.
type fakeVisits struct{ visits map[string][]access.Visit }

func (v *fakeVisits) Visits(ctx context.Context, patientID, org string) ([]access.Visit, error) {
	return v.visits[patientID+org], inTx(ctx)
}

// fakeRoles gives each practitioner's role in clinic A.
type fakeRoles struct{ roles map[string]string }

func (r *fakeRoles) OfPractitioner(ctx context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error) {
	if err := inTx(ctx); err != nil {
		return practitionerrole.PractitionerRole{}, err
	}
	id, ok := r.roles[practitionerID]
	if !ok || org != clinicA {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return practitionerrole.PractitionerRole{ID: id, OrganizationID: org, PractitionerID: practitionerID}, nil
}

// historyFixture is patient A's history as clinic A's scope shows it: visit A (role A1) and A2 (role A2) finished,
// A3 (role A1) in progress, and clinic B's two finished visits, of which the patient shared one with clinic A. The
// fake scope shows both of clinic B's, so the rule, not row-level security alone, keeps the unshared one out.
type historyFixture struct {
	*fixture
	consents *fakeConsents
}

func newHistoryFixture() historyFixture {
	f := newFixture()
	finished := func(id, appt, org, role string, at time.Time) encounter.Encounter {
		return encounter.Encounter{
			ID: id, AppointmentID: appt, OrganizationID: org, PatientID: patientA, PractitionerRoleID: role,
			Status: encounter.StatusFinished, PeriodStart: at, PeriodEnd: at.Add(20 * time.Minute), VersionID: 2,
		}
	}
	a3 := finished(encounterOf(apptA3), apptA3, clinicA, roleA1, start.Add(48*time.Hour))
	a3.Status, a3.PeriodEnd = encounter.StatusInProgress, time.Time{}
	f.repo.encounters = []encounter.Encounter{
		finished(encounterOf(apptA), apptA, clinicA, roleA1, start),
		finished(encounterOf(apptA2), apptA2, clinicA, roleA2, start.Add(time.Hour)),
		a3,
		finished(sharedB, "e5000000-0000-4000-8000-0000000000b1", clinicB, roleB1, start.Add(-24*time.Hour)),
		finished(unsharedB, "e6000000-0000-4000-8000-0000000000b2", clinicB, roleB1, start.Add(-48*time.Hour)),
	}
	f.repo.shared = []string{sharedB, unsharedB}
	consents := &fakeConsents{state: map[string]access.Consent{
		patientA + clinicA: {Active: true, History: true, Shared: []string{sharedB}},
	}}
	visits := &fakeVisits{visits: map[string][]access.Visit{
		patientA + clinicA: {{PractitionerRole: roleA1, Start: start}, {PractitionerRole: roleA2, Start: start.Add(time.Hour)}},
		patientB + clinicA: {{PractitionerRole: roleA1, Start: start.Add(2 * time.Hour)}},
	}}
	f.svc = encounter.NewService(encounter.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Appointments: f.appts,
		Consents: consents, Visits: visits, Roles: &fakeRoles{roles: map[string]string{practA1: roleA1, practA2: roleA2}},
	})
	return historyFixture{fixture: f, consents: consents}
}

func visitIDs(vs []encounter.Visit) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}
	return out
}

// A practitioner's history of the patient is the finished visits they held at their clinic and the visits of
// other clinics the patient shared with it, newest first; never a colleague's, one in progress or an unshared one.
// It is the access decision, read in the clinic's scope and not audited: the history audits what it shows.
func TestHistory_GivesTheOwnFinishedVisitsAndTheSharedOnes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller appointment.Caller
		want   []string
	}{
		{name: "practitioner A1", caller: pract, want: []string{encounterOf(apptA), sharedB}},
		{name: "practitioner A2, the colleague", caller: pract2, want: []string{encounterOf(apptA2), sharedB}},
	} {
		f := newHistoryFixture()
		got, err := f.svc.History(t.Context(), tt.caller, patientA)
		if err != nil || !slices.Equal(visitIDs(got), tt.want) {
			t.Errorf("%s: History() = %v, %v; want %v", tt.name, visitIDs(got), err, tt.want)
		}
		for _, v := range got {
			if v.Shared != (v.OrganizationID == clinicB) {
				t.Errorf("%s: visit %s Shared = %v", tt.name, v.ID, v.Shared)
			}
		}
		if f.repo.clinic != clinicA || f.repo.patient != "" {
			t.Errorf("%s: read in patient %q, clinic %q; want clinic A", tt.name, f.repo.patient, f.repo.clinic)
		}
		checkEvents(t, tt.name, f.audit)
	}
}

// Without the history consent, or once it no longer lists a visit, the other clinic's visits are not there.
func TestHistory_LeavesOutWhatTheHistoryConsentDoesNotShare(t *testing.T) {
	for name, st := range map[string]access.Consent{
		"no history consent":      {Active: true},
		"revoked history consent": {Active: true, History: false, Shared: []string{sharedB}},
		"the visit removed":       {Active: true, History: true, Shared: []string{}},
	} {
		f := newHistoryFixture()
		f.consents.state[patientA+clinicA] = st
		got, err := f.svc.History(t.Context(), pract, patientA)
		if err != nil || !slices.Equal(visitIDs(got), []string{encounterOf(apptA)}) {
			t.Errorf("%s: History() = %v, %v; want visit A only", name, visitIDs(got), err)
		}
	}
}

// Nobody but a practitioner of the clinic with an appointment with the patient under an active care consent has
// a history of the patient: ErrNotFound, which discloses nothing.
func TestHistory_IsNotFoundForAnyoneElse(t *testing.T) {
	revoked := func(f historyFixture) {
		f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: start, History: true, Shared: []string{sharedB}}
	}
	for _, tt := range []struct {
		setup   func(historyFixture)
		name    string
		patient string
		caller  appointment.Caller
	}{
		{name: "the clinic's admin", caller: admin, patient: patientA},
		{name: "the patient", caller: me, patient: patientA},
		{name: "staff without a role", caller: appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: "p9000000-0000-4000-8000-0000000000a9"}, patient: patientA},
		{name: "another clinic's practitioner", caller: appointment.Caller{AccountID: practAcc, OrganizationID: clinicB, PractitionerID: practA1}, patient: patientA},
		{name: "no consent", caller: pract, patient: patientB},
		{name: "the care consent revoked", caller: pract, patient: patientA, setup: revoked},
		{name: "not a patient id", caller: pract, patient: "x"},
	} {
		f := newHistoryFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		if got, err := f.svc.History(t.Context(), tt.caller, tt.patient); !errors.Is(err, encounter.ErrNotFound) || got != nil {
			t.Errorf("%s: History() = %v, %v; want ErrNotFound", tt.name, visitIDs(got), err)
		}
		checkEvents(t, tt.name, f.audit)
	}
}

func TestHistory_PassesFailuresOn(t *testing.T) {
	f := newHistoryFixture()
	f.consents.err = errBackend
	if _, err := f.svc.History(t.Context(), pract, patientA); !errors.Is(err, errBackend) {
		t.Errorf("History() = %v, want the backend's error", err)
	}
	f = newHistoryFixture()
	f.repo.err = errBackend
	if _, err := f.svc.History(t.Context(), pract, patientA); !errors.Is(err, errBackend) {
		t.Errorf("History() = %v, want the backend's error", err)
	}
}

// Shared gives another clinic's visit the caller may see in the patient's history, to open what belongs to it;
// not one of the caller's own clinic (its own rule applies), nor an unshared one.
func TestShared_GivesOnlyAVisitSharedWithTheCaller(t *testing.T) {
	for _, tt := range []struct {
		name   string
		id     string
		appt   string
		caller appointment.Caller
		ok     bool
	}{
		{name: "shared", caller: pract, id: sharedB, appt: "e5000000-0000-4000-8000-0000000000b1", ok: true},
		{name: "unshared", caller: pract, id: unsharedB, appt: "e6000000-0000-4000-8000-0000000000b2"},
		{name: "the clinic's own", caller: pract, id: encounterOf(apptA), appt: apptA},
		{name: "the admin", caller: admin, id: sharedB, appt: "e5000000-0000-4000-8000-0000000000b1"},
		{name: "unknown", caller: pract, id: unknownID, appt: unknownID},
	} {
		f := newHistoryFixture()
		got, err := f.svc.Shared(t.Context(), tt.caller, tt.id)
		byAppt, errAppt := f.svc.SharedAppointment(t.Context(), tt.caller, tt.appt)
		switch {
		case tt.ok && (err != nil || got.ID != tt.id || errAppt != nil || byAppt.ID != tt.id):
			t.Errorf("%s: Shared() = %s, %v and %s, %v; want the visit", tt.name, got.ID, err, byAppt.ID, errAppt)
		case !tt.ok && (!errors.Is(err, encounter.ErrNotFound) || !errors.Is(errAppt, encounter.ErrNotFound)):
			t.Errorf("%s: Shared() = %v and %v, want ErrNotFound", tt.name, err, errAppt)
		}
		checkEvents(t, tt.name, f.audit)
	}
}

// Disclosed audits each visit the history shows as the caller's search of it.
func TestDisclosed_AuditsEachVisit(t *testing.T) {
	f := newHistoryFixture()
	if err := f.svc.Disclosed(t.Context(), pract, []string{encounterOf(apptA), sharedB}); err != nil {
		t.Fatalf("Disclosed() = %v", err)
	}
	checkEvents(t, "disclosed", f.audit,
		event(pract, auditevent.CodeSearchType, encounterOf(apptA), auditevent.OutcomeSuccess),
		event(pract, auditevent.CodeSearchType, sharedB, auditevent.OutcomeSuccess))
	f.audit.err = errBackend
	if err := f.svc.Disclosed(t.Context(), pract, []string{sharedB}); !errors.Is(err, errBackend) {
		t.Errorf("Disclosed() = %v, want the auditor's error", err)
	}
}

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
