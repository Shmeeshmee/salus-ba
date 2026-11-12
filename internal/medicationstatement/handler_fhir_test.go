package medicationstatement

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
	statementID = "b1000000-0000-4000-8000-0000000000b1"
	patientA    = "d1000000-0000-4000-8000-0000000000d1"
	clinicA     = "a0000000-0000-4000-8000-00000000000a"
	encounter   = "f1000000-0000-4000-8000-0000000000f1"
	roleA       = "e1000000-0000-4000-8000-0000000000e1"
)

var asserted = time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC)

// The patient's own medication with a dose and how often: the patient its information source, the dose as a UCUM
// quantity, the timing as frequency per period; unconfirmed, so no confirmation extension.
func TestToFHIR_ThePatientsOwnMedication(t *testing.T) {
	m := MedicationStatement{ID: statementID, PatientID: patientA, DateAsserted: asserted, LastUpdated: asserted, VersionID: 1, InProfile: true,
		Code: Code{ATC: "N02BE01", Text: "Paracetamol"}, Status: StatusActive,
		Dosage: Dosage{DoseValue: 500, DoseUnit: "mg", Frequency: 2, Period: 1, PeriodUnit: "d"}}
	assertGolden(t, "medication_own.json", toFHIR(m, true))
}

// A clinic's medication attributed: the encounter as context and the recorder as information source; confirmed.
func TestToFHIR_AClinicsMedicationAttributed(t *testing.T) {
	assertGolden(t, "medication_clinic.json", toFHIR(clinicMedication(), true))
}

// Unattributed: no context and no information source, still confirmed.
func TestToFHIR_AClinicsMedicationUnattributed(t *testing.T) {
	assertGolden(t, "medication_unattributed.json", toFHIR(clinicMedication(), false))
}

// As needed, with no dose or timing: a dosage with asNeededBoolean only; typed text without a code.
func TestToFHIR_AsNeededWithoutDose(t *testing.T) {
	m := MedicationStatement{ID: statementID, PatientID: patientA, DateAsserted: asserted, LastUpdated: asserted, VersionID: 1, InProfile: true,
		Code: Code{Text: "Placeholder medication"}, Status: StatusStopped, Dosage: Dosage{AsNeeded: true}}
	assertGolden(t, "medication_as_needed.json", toFHIR(m, true))
}

func clinicMedication() MedicationStatement {
	return MedicationStatement{ID: statementID, PatientID: patientA, DateAsserted: asserted, LastUpdated: asserted, VersionID: 2, InProfile: false,
		Source:       Source{OrganizationID: clinicA, EncounterID: encounter, RecorderRoleID: roleA},
		Confirmation: Confirmation{At: asserted, RoleID: roleA, OrganizationID: clinicA},
		Code:         Code{Text: "Placeholder medication"}, Status: StatusActive,
		Dosage: Dosage{DoseValue: 1, DoseUnit: "{tablet}", Frequency: 1, Period: 1, PeriodUnit: "wk"}}
}
