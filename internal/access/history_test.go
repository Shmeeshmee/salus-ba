package access_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Synthetic ids of the history's visits.
const (
	encounterOwn       = "f1000000-0000-4000-8000-0000000000f1" // a visit at clinic A with the viewing practitioner
	encounterColleague = "f2000000-0000-4000-8000-0000000000f2" // a visit at clinic A with another practitioner
	encounterShared    = "f3000000-0000-4000-8000-0000000000f3" // a visit at clinic B, listed by the history consent
	encounterUnshared  = "f4000000-0000-4000-8000-0000000000f4" // a visit at clinic B, not listed
	roleB1             = "e3000000-0000-4000-8000-0000000000e3" // a practitioner's role in clinic B
)

var start = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

// The visits of patient A's history, at clinic A (the viewer's) and B.
var records = map[string]access.Record{
	"own":       {Encounter: encounterOwn, Clinic: clinicA, PractitionerRole: roleA1, Start: start},
	"colleague": {Encounter: encounterColleague, Clinic: clinicA, PractitionerRole: roleA2, Start: start},
	"shared":    {Encounter: encounterShared, Clinic: clinicB, PractitionerRole: roleB1, Start: start},
	"unshared":  {Encounter: encounterUnshared, Clinic: clinicB, PractitionerRole: roleB1, Start: start},
}

// historyFacts are patient A's facts in clinic A: the care consent, the history consent sharing encounterShared,
// and the viewing practitioner's appointment (any; a booking).
func historyFacts(care, history bool, visits []access.Visit) access.Facts {
	c := access.Consent{Active: care, History: history}
	if !care {
		c.RevokedAt = revokedAt
	}
	if history {
		c.Shared = []string{encounterShared}
	}
	return access.Facts{Patient: patientA, Clinic: clinicA, Consent: c, Visits: visits}
}

var booked = []access.Visit{{PractitionerRole: roleA1, Start: revokedAt.Add(48 * time.Hour)}}

// ADR 0013 "Access model": the patient sees their whole history; a practitioner sees, at their own clinic, only the
// visits they held themselves, with an active care consent; another clinic's visit only while the patient's history
// consent with the practitioner's clinic lists it, the care consent is active and the practitioner has an
// appointment with the patient; the admin and everyone else nothing.
func TestHistoryVisit_OverViewerConsentAndVisit(t *testing.T) {
	for _, tc := range []struct {
		name          string
		viewer        string
		visits        []access.Visit
		want          []string // the records seen; every other is not
		care, history bool
	}{
		{name: "the patient, whatever the consents", viewer: "patient", want: []string{"own", "colleague", "shared", "unshared"}},
		{name: "another patient", viewer: "other patient", care: true, history: true, visits: booked},
		{name: "the practitioner with both consents", viewer: "practitioner", care: true, history: true, visits: booked, want: []string{"own", "shared"}},
		{name: "the practitioner without the history consent", viewer: "practitioner", care: true, visits: booked, want: []string{"own"}},
		{name: "the practitioner after the care consent's revocation", viewer: "practitioner", history: true, visits: booked},
		{name: "the practitioner without an appointment", viewer: "practitioner", care: true, history: true, want: []string{"own"}},
		{name: "the admin", viewer: "admin", care: true, history: true, visits: booked},
	} {
		f := historyFacts(tc.care, tc.history, tc.visits)
		for name, r := range records {
			want := slices.Contains(tc.want, name)
			if got := access.HistoryVisit(viewers[tc.viewer], f, r); got != want {
				t.Errorf("%s: HistoryVisit(%s) = %v, want %v", tc.name, name, got, want)
			}
		}
	}
}

// The practitioner of another clinic never sees clinic A's visits, shared or not: a history consent shares with its
// own clinic only.
func TestHistoryVisit_AnotherClinicsPractitionerSeesNone(t *testing.T) {
	v := access.Viewer{Role: access.RolePractitioner, Clinic: clinicB, PractitionerRole: roleB1}
	f := historyFacts(true, true, booked)
	for name, r := range records {
		if access.HistoryVisit(v, f, r) {
			t.Errorf("clinic B's practitioner sees %s in clinic A's facts", name)
		}
	}
}

// A profile entry (in the profile) is seen under Profile's Full; an entry recorded in a visit and not added to the
// profile wherever that visit is seen.
func TestProfileEntry_TheProfileUnderProfileAVisitsEntryUnderHistoryVisit(t *testing.T) {
	f := historyFacts(true, true, booked)
	for _, tc := range []struct {
		viewer, record string
		inProfile      bool
		want           bool
	}{
		{"practitioner", "unshared", true, true}, // added to the profile: seen as the profile is
		{"practitioner", "unshared", false, false},
		{"practitioner", "shared", false, true},
		{"practitioner", "own", false, true},
		{"practitioner", "colleague", false, false},
		{"admin", "own", true, false}, // the admin sees the name only, never the profile's content
		{"patient", "colleague", false, true},
		{"other patient", "own", true, false},
	} {
		if got := access.ProfileEntry(viewers[tc.viewer], f, tc.inProfile, records[tc.record]); got != tc.want {
			t.Errorf("ProfileEntry(%s, %s, in profile %v) = %v, want %v", tc.viewer, tc.record, tc.inProfile, got, tc.want)
		}
	}
	// No appointment with the practitioner: not even the profile.
	if access.ProfileEntry(viewers["practitioner"], historyFacts(true, true, nil), true, records["own"]) {
		t.Error("ProfileEntry without an appointment = true, want false")
	}
}

// Which clinic recorded or confirmed an entry is shown to the patient and to staff of that clinic only: anywhere
// else it would tell that the patient was treated there ("Confirmed by a clinician", ADR 0013).
func TestAttributed_OnlyThePatientAndTheRecordingClinic(t *testing.T) {
	f := historyFacts(true, true, booked)
	for _, tc := range []struct {
		viewer, clinic string
		want           bool
	}{
		{"patient", clinicB, true},
		{"practitioner", clinicA, true},
		{"admin", clinicA, true},
		{"practitioner", clinicB, false},
		{"other patient", clinicA, false},
	} {
		if got := access.Attributed(viewers[tc.viewer], f, tc.clinic); got != tc.want {
			t.Errorf("Attributed(%s, %s) = %v, want %v", tc.viewer, tc.clinic, got, tc.want)
		}
	}
}
