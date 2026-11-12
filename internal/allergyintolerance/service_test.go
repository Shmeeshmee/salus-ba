package allergyintolerance_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Synthetic ids: two clinics, a practitioner of each, a colleague at A, one patient and their visits.
const (
	clinicA      = "0a000000-0000-4000-8000-00000000000a"
	clinicB      = "0b000000-0000-4000-8000-00000000000b"
	roleA        = "1a000000-0000-4000-8000-00000000001a"
	roleC        = "1c000000-0000-4000-8000-00000000001c"
	roleB        = "1b000000-0000-4000-8000-00000000001b"
	practA       = "2a000000-0000-4000-8000-00000000002a"
	practC       = "2c000000-0000-4000-8000-00000000002c"
	patientP     = "3a000000-0000-4000-8000-00000000003a"
	encA         = "4a000000-0000-4000-8000-00000000004a"
	encC         = "4c000000-0000-4000-8000-00000000004c"
	encB         = "4b000000-0000-4000-8000-00000000004b"
	encShared    = "4d000000-0000-4000-8000-00000000004d"
	encFinished  = "4e000000-0000-4000-8000-00000000004e"
	termDrug     = "5a000000-0000-4000-8000-00000000005a"
	termAnatomy  = "5b000000-0000-4000-8000-00000000005b"
	idOwn        = "6a000000-0000-4000-8000-000000000001"
	idOwnByB     = "6a000000-0000-4000-8000-000000000002"
	idVisitA     = "6a000000-0000-4000-8000-000000000003"
	idColleague  = "6a000000-0000-4000-8000-000000000004"
	idVisitB     = "6a000000-0000-4000-8000-000000000005"
	idShared     = "6a000000-0000-4000-8000-000000000006"
	idRemoved    = "6a000000-0000-4000-8000-000000000007"
	accountP     = "7a000000-0000-4000-8000-00000000007a"
	accountPract = "7b000000-0000-4000-8000-00000000007b"
)

var practitionerOfRole = map[string]string{roleA: practA, roleC: practC}

var now = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

var (
	patient      = appointment.Caller{AccountID: accountP, PatientID: patientP}
	practitioner = appointment.Caller{AccountID: accountPract, OrganizationID: clinicA, PractitionerID: practA}
	colleague    = appointment.Caller{AccountID: accountPract, OrganizationID: clinicA, PractitionerID: practC}
	admin        = appointment.Caller{AccountID: accountPract, OrganizationID: clinicA, Admin: true}
)

func entry(id string, src allergyintolerance.Source, inProfile bool, conf allergyintolerance.Confirmation) allergyintolerance.AllergyIntolerance {
	v := allergyintolerance.VerificationUnconfirmed
	if !conf.At.IsZero() {
		v = allergyintolerance.VerificationConfirmed
	}
	return allergyintolerance.AllergyIntolerance{ID: id, PatientID: patientP, RecordedDate: now.Add(-time.Hour), LastUpdated: now.Add(-time.Hour),
		Source: src, InProfile: inProfile, Confirmation: conf, Code: allergyintolerance.Code{Text: "Placeholder " + id[len(id)-1:]},
		ClinicalStatus: allergyintolerance.ClinicalActive, VerificationStatus: v, VersionID: 1}
}

func byA() allergyintolerance.Confirmation {
	return allergyintolerance.Confirmation{At: now.Add(-time.Hour), RoleID: roleA, OrganizationID: clinicA}
}

func byB() allergyintolerance.Confirmation {
	return allergyintolerance.Confirmation{At: now.Add(-time.Hour), RoleID: roleB, OrganizationID: clinicB}
}

// history is the patient's entries: their own (one picked from the terminology), one of theirs a clinic B
// practitioner confirmed, one recorded in each of: the practitioner's visit at A, a colleague's at A, an unshared
// visit at B, a visit at B shared with A; and a removed one.
func history() *fakeRepo {
	own := entry(idOwn, allergyintolerance.Source{}, true, allergyintolerance.Confirmation{})
	own.Code.TermID = termDrug
	removed := entry(idRemoved, allergyintolerance.Source{}, true, allergyintolerance.Confirmation{})
	removed.VerificationStatus = allergyintolerance.VerificationEnteredInError
	return newRepo(own,
		entry(idOwnByB, allergyintolerance.Source{}, true, byB()),
		entry(idVisitA, allergyintolerance.Source{OrganizationID: clinicA, EncounterID: encA, RecorderRoleID: roleA}, false, byA()),
		entry(idColleague, allergyintolerance.Source{OrganizationID: clinicA, EncounterID: encC, RecorderRoleID: roleC}, false,
			allergyintolerance.Confirmation{At: now.Add(-time.Hour), RoleID: roleC, OrganizationID: clinicA}),
		entry(idVisitB, allergyintolerance.Source{OrganizationID: clinicB, EncounterID: encB, RecorderRoleID: roleB}, false, byB()),
		entry(idShared, allergyintolerance.Source{OrganizationID: clinicB, EncounterID: encShared, RecorderRoleID: roleB}, false, byB()),
		removed)
}

type world struct {
	repo  *fakeRepo
	audit *fakeAudit
	svc   *allergyintolerance.Service
}

// newWorld is the service on repo with the patient's consents with clinic A (care active, history sharing
// encShared), the practitioner's appointment with the patient there, and the encounters at A.
func newWorld(repo *fakeRepo, consent access.Consent) world {
	audit := &fakeAudit{}
	svc := allergyintolerance.NewService(allergyintolerance.Ports{
		Repo: repo, Audit: audit, Tx: fakeTx{}, Clock: clock.Fixed{T: now},
		Consents: fakeConsents{clinicA: consent},
		Visits:   fakeVisits{clinicA: {{PractitionerRole: roleA, Start: now}, {PractitionerRole: roleC, Start: now.Add(-48 * time.Hour)}}},
		Roles:    fakeRoles{practA: roleA, practC: roleC},
		Encounters: fakeEncounters{consent: consent.Active, byID: map[string]encounter.Encounter{
			encA:        {ID: encA, OrganizationID: clinicA, PatientID: patientP, PractitionerRoleID: roleA, Status: encounter.StatusInProgress},
			encFinished: {ID: encFinished, OrganizationID: clinicA, PatientID: patientP, PractitionerRoleID: roleA, Status: encounter.StatusFinished},
		}},
		Terms: fakeTerms{termDrug: "J01CA04"},
	})
	return world{repo: repo, audit: audit, svc: svc}
}

func sharing() access.Consent {
	return access.Consent{Active: true, History: true, Shared: []string{encShared}}
}

func ids(seen []allergyintolerance.Seen) []string {
	out := make([]string, len(seen))
	for i, s := range seen {
		out[i] = s.ID
	}
	return out
}

// Each viewer sees what access.ProfileEntry gives them, never a removed entry, attributed as access.Attributed says;
// each entry returned is audited.
func TestSeen_EachViewerSeesWhatTheRuleGives(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		want       []string
		attributed []string
		caller     appointment.Caller
		consent    access.Consent
	}{
		{name: "the patient sees every entry but the removed one", caller: patient, consent: sharing(),
			want: []string{idOwn, idOwnByB, idVisitA, idColleague, idVisitB, idShared}, attributed: []string{idOwn, idOwnByB, idVisitA, idColleague, idVisitB, idShared}},
		{name: "the practitioner: the profile, their own visit, the shared visit", caller: practitioner, consent: sharing(),
			want: []string{idOwn, idOwnByB, idVisitA, idShared}, attributed: []string{idVisitA}},
		{name: "without the history consent no shared visit", caller: practitioner, consent: access.Consent{Active: true, Shared: []string{encShared}},
			want: []string{idOwn, idOwnByB, idVisitA}, attributed: []string{idVisitA}},
		{name: "a colleague: the profile and their own visit", caller: colleague, consent: sharing(),
			want: []string{idOwn, idOwnByB, idColleague, idShared}, attributed: []string{idColleague}},
		{name: "the admin sees nothing", caller: admin, consent: sharing(),
			want: nil, attributed: nil},
		{name: "after a revocation nothing", caller: practitioner, consent: access.Consent{RevokedAt: now},
			want: nil, attributed: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld(history(), tc.consent)
			seen, err := w.svc.Seen(t.Context(), tc.caller, patientP)
			if err != nil {
				t.Fatal(err)
			}
			if got := ids(seen); !slices.Equal(got, tc.want) {
				t.Errorf("seen %v, want %v", got, tc.want)
			}
			var attributed []string
			for _, s := range seen {
				if s.Attributed {
					attributed = append(attributed, s.ID)
				}
			}
			if !slices.Equal(attributed, tc.attributed) {
				t.Errorf("attributed %v, want %v", attributed, tc.attributed)
			}
			var refs []string
			for _, id := range tc.want {
				refs = append(refs, "AllergyIntolerance/"+id)
			}
			if len(refs) == 0 {
				refs = []string{"Patient/" + patientP} // a search that found none names the patient asked about
			}
			if got := w.audit.refs(); !slices.Equal(got, refs) {
				t.Errorf("audited %v, want %v", got, refs)
			}
		})
	}
}

// A term's ATC code comes from the terminology; the search is audited as one, with the caller's purpose.
func TestSeen_CarriesTheATCCodeAndAuditsTheSearch(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	seen, err := w.svc.Seen(t.Context(), patient, patientP)
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].ID != idOwn || seen[0].Code.ATC != "J01CA04" {
		t.Errorf("the own entry's code = %+v, want the term's ATC code", seen[0].Code)
	}
	e := w.audit.events[0]
	if e.Subtype.Code != auditevent.CodeSearchType || e.Action != auditevent.ActionExecute || e.Purpose.Code != auditevent.CodePATRQT ||
		e.AgentAccountID != accountP || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("the event = %+v, want the patient's search", e)
	}
	if w.repo.patient != patientP {
		t.Errorf("the scope = %q, want the patient's", w.repo.patient)
	}
}

// Another patient's id gives the patient nothing.
func TestSeen_APatientSeesNoOtherPatient(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	seen, err := w.svc.Seen(t.Context(), appointment.Caller{AccountID: accountP, PatientID: "3b000000-0000-4000-8000-00000000003b"}, patientP)
	if err != nil || len(seen) != 0 {
		t.Errorf("seen %v, %v; want nothing", ids(seen), err)
	}
}

// Get reads one entry under the same rule: ErrNotFound for one the caller does not see and for a removed one, both
// audited without naming it.
func TestGet_ReadsWhatTheCallerSees(t *testing.T) {
	t.Parallel()
	w := newWorld(history(), sharing())
	got, err := w.svc.Get(t.Context(), practitioner, idVisitA)
	if err != nil || got.ID != idVisitA || !got.Attributed {
		t.Fatalf("the practitioner's own visit's entry: %+v, %v", got, err)
	}
	for _, id := range []string{idColleague, idVisitB, idRemoved, "not-a-uuid"} {
		if _, err := w.svc.Get(t.Context(), practitioner, id); !errors.Is(err, allergyintolerance.ErrNotFound) {
			t.Errorf("Get(%s) = %v, want ErrNotFound", id, err)
		}
	}
	if got := w.audit.refs(); !slices.Equal(got, []string{"AllergyIntolerance/" + idVisitA, "", "", "", ""}) {
		t.Errorf("audited %v", got)
	}
	if e := w.audit.events[1]; e.Outcome != auditevent.OutcomeMinorFailure || e.Purpose.Code != auditevent.CodeTREAT || e.OrganizationID != clinicA {
		t.Errorf("the refusal = %+v", e)
	}
}
