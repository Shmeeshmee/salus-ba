package history_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// Synthetic ids only.
const (
	sharingPatient = "d1000000-0000-4000-8000-0000000000d1"
	sharingAccount = "c1000000-0000-4000-8000-0000000000c1"
	receiving      = "a0000000-0000-4000-8000-00000000000a" // the clinic the patient shares with
	elsewhere      = "b0000000-0000-4000-8000-00000000000b"
	thirdClinic    = "c0000000-0000-4000-8000-00000000000c"
	roleElsewhere  = "b1000000-0000-4000-8000-0000000000b1"
	roleThird      = "c1000000-0000-4000-8000-0000000000c1"
	roleReceiving  = "a1000000-0000-4000-8000-0000000000a1"
)

var sharingCaller = consent.Caller{AccountID: sharingAccount, PatientID: sharingPatient}

func day(d int) time.Time { return time.Date(2026, 10, d, 9, 0, 0, 0, time.UTC) }

// fakeSharingEncounters gives the patient's encounters to the patient, as encounter.Service.Search does.
type fakeSharingEncounters struct {
	err     error
	all     []encounter.Encounter
	callers []appointment.Caller
}

func (f *fakeSharingEncounters) Search(_ context.Context, c appointment.Caller, _ encounter.Filter) ([]encounter.Encounter, error) {
	f.callers = append(f.callers, c)
	return f.all, f.err
}

// fakeSignedDocuments gives each encounter's signed documents, in the order signed.
type fakeSignedDocuments struct {
	err    error
	titles map[string][]string
}

func (f *fakeSignedDocuments) Signed(_ context.Context, c appointment.Caller, encounterID string) ([]composition.Document, error) {
	if c.PatientID != sharingPatient || c.OrganizationID != "" {
		return nil, composition.ErrForbidden
	}
	var out []composition.Document
	for _, t := range f.titles[encounterID] {
		out = append(out, composition.Document{Draft: composition.Draft{Title: t}})
	}
	return out, f.err
}

type fakeSharingRoles struct {
	err   error
	roles map[string]practitionerrole.PractitionerRole
}

func (f *fakeSharingRoles) Lookup(_ context.Context, org, id string) (practitionerrole.PractitionerRole, error) {
	r, ok := f.roles[id]
	if !ok || r.OrganizationID != org {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return r, f.err
}

type fakeSharingClinics struct {
	err   error
	names map[string]string
	asked []string
}

func (f *fakeSharingClinics) Name(_ context.Context, id string) (string, error) {
	f.asked = append(f.asked, id)
	n, ok := f.names[id]
	if !ok {
		return "", organization.ErrNotFound
	}
	return n, f.err
}

type sharingFixture struct {
	encounters *fakeSharingEncounters
	documents  *fakeSignedDocuments
	roles      *fakeSharingRoles
	clinics    *fakeSharingClinics
	svc        *history.Sharing
}

func visitAt(id, clinic, role, status string, start time.Time) encounter.Encounter {
	return encounter.Encounter{ID: id, OrganizationID: clinic, PractitionerRoleID: role, PatientID: sharingPatient, Status: encounter.Status(status), PeriodStart: start}
}

func newSharingFixture() sharingFixture {
	f := sharingFixture{
		encounters: &fakeSharingEncounters{all: []encounter.Encounter{
			visitAt("e1000000-0000-4000-8000-0000000000e1", elsewhere, roleElsewhere, "finished", day(10)),
			visitAt("e2000000-0000-4000-8000-0000000000e2", receiving, roleReceiving, "finished", day(12)), // at the receiving clinic
			visitAt("e3000000-0000-4000-8000-0000000000e3", thirdClinic, roleThird, "finished", day(14)),
			visitAt("e4000000-0000-4000-8000-0000000000e4", elsewhere, roleElsewhere, "in-progress", day(16)), // not finished
			visitAt("e5000000-0000-4000-8000-0000000000e5", elsewhere, roleElsewhere, "finished", day(11)),
		}},
		documents: &fakeSignedDocuments{titles: map[string][]string{
			"e1000000-0000-4000-8000-0000000000e1": {"Placeholder findings", "Placeholder referral"},
		}},
		roles: &fakeSharingRoles{roles: map[string]practitionerrole.PractitionerRole{
			roleElsewhere: {ID: roleElsewhere, OrganizationID: elsewhere, SpecialtyCode: "394579002"},
			roleThird:     {ID: roleThird, OrganizationID: thirdClinic},
			roleReceiving: {ID: roleReceiving, OrganizationID: receiving, SpecialtyCode: "419192003"},
		}},
		clinics: &fakeSharingClinics{names: map[string]string{
			receiving: "Synthetic clinic A", elsewhere: "Synthetic clinic B", thirdClinic: "Synthetic clinic C",
		}},
	}
	f.svc = history.NewSharing(history.SharingPorts{Encounters: f.encounters, Documents: f.documents, Roles: f.roles, Clinics: f.clinics})
	return f
}

// The candidates are the patient's finished visits at the other clinics, newest first, each with what the receiving
// clinic will see: the date, the clinic, the role's specialty and the first signed document's title.
func TestShareable_GivesTheFinishedVisitsAtOtherClinicsAsTheClinicWillSeeThem(t *testing.T) {
	f := newSharingFixture()
	got, err := f.svc.Shareable(t.Context(), sharingCaller, receiving)
	want := []consent.Candidate{
		{Date: day(14), EncounterID: "e3000000-0000-4000-8000-0000000000e3", OrganizationID: thirdClinic, OrganizationName: "Synthetic clinic C"},
		{Date: day(11), EncounterID: "e5000000-0000-4000-8000-0000000000e5", OrganizationID: elsewhere, OrganizationName: "Synthetic clinic B", Specialty: "394579002"},
		{Date: day(10), EncounterID: "e1000000-0000-4000-8000-0000000000e1", OrganizationID: elsewhere, OrganizationName: "Synthetic clinic B", Specialty: "394579002", Title: "Placeholder findings"},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Shareable() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []appointment.Caller{{AccountID: sharingAccount, PatientID: sharingPatient}}; !reflect.DeepEqual(f.encounters.callers, want) {
		t.Errorf("searched as %+v, want the patient %+v", f.encounters.callers, want)
	}
}

func TestShareable_IsEmptyWithoutVisitsAndRefusesAnUnknownClinic(t *testing.T) {
	f := newSharingFixture()
	f.encounters.all = nil
	if got, err := f.svc.Shareable(t.Context(), sharingCaller, receiving); err != nil || got == nil || len(got) != 0 {
		t.Errorf("Shareable() without visits = %#v, %v; want an empty list", got, err)
	}
	if _, err := f.svc.Shareable(t.Context(), sharingCaller, "f0000000-0000-4000-8000-00000000000f"); !errors.Is(err, consent.ErrClinicNotFound) {
		t.Errorf("Shareable() to an unknown clinic = %v, want consent.ErrClinicNotFound", err)
	}
}

func TestShareable_FailsWhenASourceFails(t *testing.T) {
	boom := errors.New("connection reset")
	for name, broken := range map[string]func(f sharingFixture){
		"encounters": func(f sharingFixture) { f.encounters.err = boom },
		"documents":  func(f sharingFixture) { f.documents.err = boom },
		"roles":      func(f sharingFixture) { f.roles.err = boom },
		"clinics":    func(f sharingFixture) { f.clinics.err = boom },
	} {
		f := newSharingFixture()
		broken(f)
		if _, err := f.svc.Shareable(t.Context(), sharingCaller, receiving); !errors.Is(err, boom) {
			t.Errorf("%s broken: Shareable() = %v, want the error", name, err)
		}
	}
}
