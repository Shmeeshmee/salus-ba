package allergyintolerance

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
	allergyID  = "a1000000-0000-4000-8000-0000000000a1"
	patientA   = "d1000000-0000-4000-8000-0000000000d1"
	clinicA    = "a0000000-0000-4000-8000-00000000000a"
	encounterA = "f1000000-0000-4000-8000-0000000000f1"
	roleA      = "e1000000-0000-4000-8000-0000000000e1"
	termID     = "c1000000-0000-4000-8000-0000000000c1"
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
		Source:       Source{OrganizationID: clinicA, EncounterID: encounterA, RecorderRoleID: roleA},
		Confirmation: Confirmation{At: recorded, RoleID: roleA, OrganizationID: clinicA},
		Code:         Code{Text: "Placeholder substance"}, Category: CategoryFood,
		ClinicalStatus: ClinicalActive, VerificationStatus: VerificationConfirmed}
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

func (s *stubReader) Provenance(context.Context, Caller, string) (Provenance, bool, error) {
	return Provenance{}, false, nil
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
	svc := &stubReader{out: []Seen{{AllergyIntolerance: clinicAllergy(), Attributed: false}}}
	rec := fhirGet(t, svc, true, "/fhir/R4/AllergyIntolerance/"+allergyID)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"3"` || rec.Header().Get("Content-Type") != "application/fhir+json" ||
		svc.id != allergyID || svc.caller.PatientID != patientA {
		t.Fatalf("read: %d %v, asked %q", rec.Code, rec.Header(), svc.id)
	}
	assertBody(t, "read_unattributed.json", rec.Body.Bytes())
	rec = fhirGet(t, &stubReader{err: ErrNotFound}, true, "/fhir/R4/AllergyIntolerance/"+allergyID)
	if rec.Code != http.StatusNotFound {
		t.Errorf("not seen: %d", rec.Code)
	}
	assertBody(t, "not_found.json", rec.Body.Bytes())
	if rec := fhirGet(t, &stubReader{}, false, "/fhir/R4/AllergyIntolerance/"+allergyID); rec.Code != http.StatusUnauthorized {
		t.Errorf("nobody: %d", rec.Code)
	}
}

// The search by patient is a searchset of what the caller sees; any other search is a 400.
func TestFHIRHandler_SearchByPatient(t *testing.T) {
	own := AllergyIntolerance{ID: allergyID, PatientID: patientA, RecordedDate: recorded, LastUpdated: recorded, VersionID: 1, InProfile: true,
		Code: Code{Text: "Placeholder substance"}, ClinicalStatus: ClinicalActive, VerificationStatus: VerificationUnconfirmed}
	theirs := clinicAllergy()
	theirs.ID = "a1000000-0000-4000-8000-0000000000a2"
	svc := &stubReader{out: []Seen{{AllergyIntolerance: own, Attributed: true}, {AllergyIntolerance: theirs}}}
	rec := fhirGet(t, svc, true, "/fhir/R4/AllergyIntolerance?patient=Patient/"+patientA)
	if rec.Code != http.StatusOK || svc.patient != patientA {
		t.Fatalf("search: %d, patient %q", rec.Code, svc.patient)
	}
	assertBody(t, "search.json", rec.Body.Bytes())
	for _, q := range []string{"", "?patient=" + patientA + "&patient=" + patientA, "?patient=Practitioner/x", "?code=x&patient=" + patientA} {
		if rec := fhirGet(t, &stubReader{}, true, "/fhir/R4/AllergyIntolerance"+q); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: %d, want 400", q, rec.Code)
		}
	}
}
