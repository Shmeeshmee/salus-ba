package medicationstatement

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	encounterA  = "f1000000-0000-4000-8000-0000000000f1"
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
		Source:       Source{OrganizationID: clinicA, EncounterID: encounterA, RecorderRoleID: roleA},
		Confirmation: Confirmation{At: asserted, RoleID: roleA, OrganizationID: clinicA},
		Code:         Code{Text: "Placeholder medication"}, Status: StatusActive,
		Dosage: Dosage{DoseValue: 1, DoseUnit: "{tablet}", Frequency: 1, Period: 1, PeriodUnit: "wk"}}
}

// stubReader answers the FHIR handler with what it is given and keeps what it was asked.
type stubReader struct {
	err     error
	caller  Caller
	id      string
	patient string
	out     []Seen
}

func (s *stubReader) Get(_ context.Context, c Caller, id string) (Seen, error) {
	s.caller, s.id = c, id
	if s.err != nil || len(s.out) == 0 {
		return Seen{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) Seen(_ context.Context, c Caller, patientID string) ([]Seen, error) {
	s.caller, s.patient = c, patientID
	return s.out, s.err
}

// assertBody compares a response body with testdata/<name>; -update rewrites the file.
func assertBody(t *testing.T, name string, got []byte) {
	t.Helper()
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
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

func fhirGet(t *testing.T, svc *stubReader, signedIn bool, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (Caller, bool, error) {
		return Caller{AccountID: "x", PatientID: patientA}, signedIn, nil
	}
	NewFHIRHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

// The read answers the entry as the caller may see it, with its ETag; one the caller does not see is a 404
// OperationOutcome; nobody signed in a 401.
func TestFHIRHandler_Read(t *testing.T) {
	svc := &stubReader{out: []Seen{{MedicationStatement: clinicMedication(), Attributed: false}}}
	rec := fhirGet(t, svc, true, "/fhir/R4/MedicationStatement/"+statementID)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || rec.Header().Get("Content-Type") != "application/fhir+json" ||
		svc.id != statementID || svc.caller.PatientID != patientA {
		t.Fatalf("read: %d %v, asked %q", rec.Code, rec.Header(), svc.id)
	}
	assertBody(t, "read_unattributed.json", rec.Body.Bytes())
	rec = fhirGet(t, &stubReader{err: ErrNotFound}, true, "/fhir/R4/MedicationStatement/"+statementID)
	if rec.Code != http.StatusNotFound {
		t.Errorf("not seen: %d", rec.Code)
	}
	assertBody(t, "not_found.json", rec.Body.Bytes())
	if rec := fhirGet(t, &stubReader{}, false, "/fhir/R4/MedicationStatement/"+statementID); rec.Code != http.StatusUnauthorized {
		t.Errorf("nobody: %d", rec.Code)
	}
}

// The search by patient is a searchset of what the caller sees; any other search is a 400.
func TestFHIRHandler_SearchByPatient(t *testing.T) {
	own := MedicationStatement{ID: statementID, PatientID: patientA, DateAsserted: asserted, LastUpdated: asserted, VersionID: 1, InProfile: true,
		Code: Code{Text: "Placeholder medication"}, Status: StatusActive, Dosage: Dosage{AsNeeded: true}}
	theirs := clinicMedication()
	theirs.ID = "b1000000-0000-4000-8000-0000000000b2"
	svc := &stubReader{out: []Seen{{MedicationStatement: own, Attributed: true}, {MedicationStatement: theirs}}}
	rec := fhirGet(t, svc, true, "/fhir/R4/MedicationStatement?patient=Patient/"+patientA)
	if rec.Code != http.StatusOK || svc.patient != patientA {
		t.Fatalf("search: %d, patient %q", rec.Code, svc.patient)
	}
	assertBody(t, "search.json", rec.Body.Bytes())
	for _, q := range []string{"", "?patient=" + patientA + "&patient=" + patientA, "?patient=Practitioner/x", "?code=x&patient=" + patientA} {
		if rec := fhirGet(t, &stubReader{}, true, "/fhir/R4/MedicationStatement"+q); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: %d, want 400", q, rec.Code)
		}
	}
}
