package allergyintolerance

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares v as JSON with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs:\n got: %s\nwant: %s", path, got, want)
	}
}

// Synthetic ids only.
const (
	allergyID = "a1000000-0000-4000-8000-0000000000a1"
	patientA  = "d1000000-0000-4000-8000-0000000000d1"
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	encounter = "f1000000-0000-4000-8000-0000000000f1"
	roleA     = "e1000000-0000-4000-8000-0000000000e1"
	termID    = "c1000000-0000-4000-8000-0000000000c1"
)

var recorded = time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC)

// The patient's own allergy, picked from the terminology (ATC code), with a reaction: unconfirmed, the patient its
// asserter, no recorder and no encounter.
func TestToFHIR_ThePatientsOwnAllergy(t *testing.T) {
	a := AllergyIntolerance{ID: allergyID, PatientID: patientA, RecordedDate: recorded, LastUpdated: recorded, VersionID: 1, InProfile: true,
		Code:     Code{TermID: termID, ATC: "J01CA04", Text: "Amoxicillin"},
		Category: CategoryMedication, Reaction: "Placeholder reaction", Severity: SeverityModerate,
		ClinicalStatus: ClinicalActive, VerificationStatus: VerificationUnconfirmed}
	assertGolden(t, "allergy_own.json", toFHIR(a, true))
}

// A clinic's allergy as its own clinic sees it (attributed): the encounter and the recorder; confirmed.
func TestToFHIR_AClinicsAllergyAttributed(t *testing.T) {
	assertGolden(t, "allergy_clinic.json", toFHIR(clinicAllergy(), true))
}

// The same allergy where its clinic may not be named (added to the profile, seen by another clinic): no encounter,
// no recorder, no asserter; still confirmed ("confirmed by a clinician").
func TestToFHIR_AClinicsAllergyUnattributed(t *testing.T) {
	assertGolden(t, "allergy_unattributed.json", toFHIR(clinicAllergy(), false))
}

// A typed substance has text only (no free-text code); a mistaken entry has no clinical status (FHIR ait-2), and
// an entry without a reaction no reaction.
func TestToFHIR_TypedTextAndEnteredInError(t *testing.T) {
	a := AllergyIntolerance{ID: allergyID, PatientID: patientA, RecordedDate: recorded, LastUpdated: recorded, VersionID: 2, InProfile: true,
		Code: Code{Text: "Placeholder substance"}, ClinicalStatus: ClinicalActive, VerificationStatus: VerificationEnteredInError}
	assertGolden(t, "allergy_entered_in_error.json", toFHIR(a, true))
}

func clinicAllergy() AllergyIntolerance {
	return AllergyIntolerance{ID: allergyID, PatientID: patientA, RecordedDate: recorded, LastUpdated: recorded, VersionID: 3, InProfile: true,
		Source:       Source{OrganizationID: clinicA, EncounterID: encounter, RecorderRoleID: roleA},
		Confirmation: Confirmation{At: recorded, RoleID: roleA, OrganizationID: clinicA},
		Code:         Code{Text: "Placeholder substance"}, Category: CategoryFood,
		ClinicalStatus: ClinicalActive, VerificationStatus: VerificationConfirmed}
}
