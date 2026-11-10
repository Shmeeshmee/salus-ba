package composition_test

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
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

// stubReader answers every read with doc, or err, and keeps the id asked for.
type stubReader struct {
	err error
	id  string
	doc composition.Document
}

func (s *stubReader) Signed(_ context.Context, _ appointment.Caller, encounterID string) ([]composition.Document, error) {
	s.id = encounterID
	if s.err != nil {
		return nil, s.err
	}
	if s.doc.Draft.ID == "" {
		return []composition.Document{}, nil
	}
	return []composition.Document{s.doc}, nil
}

func (s *stubReader) Composition(_ context.Context, _ appointment.Caller, id string) (composition.Document, error) {
	s.id = id
	return s.doc, s.err
}

func (s *stubReader) DocumentReference(_ context.Context, _ appointment.Caller, id string) (composition.Document, error) {
	s.id = id
	return s.doc, s.err
}

func (s *stubReader) Provenance(_ context.Context, _ appointment.Caller, id string) (composition.Document, error) {
	s.id = id
	return s.doc, s.err
}

const (
	draftA = "d0000000-0000-4000-8000-0000000000a1"
	refA   = "d0c00000-0000-4000-8000-0000000000a1"
	provA  = "9e000000-0000-4000-8000-0000000000a1"
)

// signedDoc is a signed findings report: findings and tests filled, the control date missing, the diagnosis
// coded; text that needs escaping in the narrative. Synthetic.
func signedDoc() composition.Document {
	signed := time.Date(2030, 1, 7, 9, 45, 0, 0, time.UTC)
	return composition.Document{
		Encounter: encounter.Encounter{ID: encA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1},
		Draft: composition.Draft{
			ID: draftA, EncounterID: encA, TemplateID: tplID, Language: "bs", Title: "Placeholder findings report", TypeCode: "11488-4",
			Status: composition.StatusFinal, VersionID: 3, UpdatedAt: signed, SignedAt: signed, SignerRoleID: roleA1,
			DocumentReferenceID: refA, ProvenanceID: provA,
			PDF: composition.PDF{KEKID: "k1", SHA1: bytes.Repeat([]byte{0xab}, 20), Size: 20480},
			Fields: []composition.DraftField{
				{Field: composition.Field{Key: "findings", Value: "Placeholder <findings> & more\nsecond line", SourceSegmentIDs: []string{seg1}}, Type: composition.FieldText, Label: "Placeholder findings"},
				{Field: composition.Field{Key: "tests", Value: "Placeholder one\nPlaceholder two"}, Type: composition.FieldList, Label: "Placeholder tests"},
				{Field: composition.Field{Key: "control", Missing: true}, Type: composition.FieldDate, Label: "Placeholder control date"},
				{Field: composition.Field{Key: "diagnosis", Value: "J06.9"}, Type: composition.FieldCoded, Label: "Placeholder diagnosis"},
			},
		},
	}
}

func getFHIR(t *testing.T, svc *stubReader, signedIn bool, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (appointment.Caller, bool, error) { return pract, signedIn, nil }
	composition.NewFHIRHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func TestFHIR_ReadsTheSignedDocumentsResources(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ path, golden, etag string }{
		{"/fhir/R4/Composition/" + draftA, "composition_final.json", `W/"3"`},
		{"/fhir/R4/DocumentReference/" + refA, "documentreference.json", `W/"1"`},
		{"/fhir/R4/Provenance/" + provA, "provenance.json", `W/"1"`},
	} {
		svc := &stubReader{doc: signedDoc()}
		rec := getFHIR(t, svc, true, tt.path)
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/fhir+json" || rec.Header().Get("ETag") != tt.etag ||
			!strings.HasSuffix(tt.path, svc.id) {
			t.Errorf("%s = %d %v", tt.path, rec.Code, rec.Header())
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
}

// A preliminary Composition has no attester; its author is the encounter's practitioner and its date the
// draft's last change.
func TestFHIR_ReadsAPreliminaryComposition(t *testing.T) {
	t.Parallel()
	doc := signedDoc()
	d := &doc.Draft
	d.Status, d.SignedAt, d.SignerRoleID, d.DocumentReferenceID, d.ProvenanceID, d.PDF = composition.StatusPreliminary, time.Time{}, "", "", "", composition.PDF{}
	d.VersionID, d.UpdatedAt = 2, time.Date(2030, 1, 7, 9, 40, 0, 0, time.UTC)
	rec := getFHIR(t, &stubReader{doc: doc}, true, "/fhir/R4/Composition/"+draftA)
	assertGolden(t, "composition_preliminary.json", rec.Body.Bytes())
}

func TestFHIR_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		code   string
		status int
	}{
		{composition.ErrNotFound, `"code":"not-found"`, 404},
		{composition.ErrForbidden, `"code":"forbidden"`, 403},
		{composition.ErrConsentRequired, `"code":"forbidden"`, 403},
		{errBackend, `"code":"exception"`, 500},
	} {
		rec := getFHIR(t, &stubReader{err: tt.err}, true, "/fhir/R4/Provenance/"+provA)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.code) || !strings.Contains(rec.Body.String(), "OperationOutcome") {
			t.Errorf("%v: %d %s", tt.err, rec.Code, rec.Body)
		}
	}
	if rec := getFHIR(t, &stubReader{}, false, "/fhir/R4/Composition/"+draftA); rec.Code != 401 {
		t.Errorf("signed out = %d, want 401", rec.Code)
	}
}

// SAL-17: DocumentReference?encounter=… is the encounter's signed documents as a searchset, the reference given
// as an id or as Encounter/<id>; any other search is not supported.
func TestFHIR_SearchesTheEncountersSignedDocuments(t *testing.T) {
	t.Parallel()
	for _, q := range []string{encA, "Encounter/" + encA} {
		svc := &stubReader{doc: signedDoc()}
		rec := getFHIR(t, svc, true, "/fhir/R4/DocumentReference?encounter="+q)
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/fhir+json" || rec.Header().Get("Cache-Control") != "no-store" || svc.id != encA {
			t.Errorf("search %s = %d %v, asked for %q", q, rec.Code, rec.Header(), svc.id)
		}
		assertGolden(t, "documentreference_search.json", rec.Body.Bytes())
	}
	if rec := getFHIR(t, &stubReader{}, true, "/fhir/R4/DocumentReference?encounter="+encA); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("search without documents = %d %s", rec.Code, rec.Body)
	}
	for _, q := range []string{"", "?patient=" + patientA, "?encounter=" + encA + "&encounter=" + encA, "?encounter=Patient/" + encA, "?encounter=" + encA + "&status=current"} {
		if rec := getFHIR(t, &stubReader{}, true, "/fhir/R4/DocumentReference"+q); rec.Code != 400 || !strings.Contains(rec.Body.String(), "not-supported") {
			t.Errorf("search %q = %d %s, want 400 not-supported", q, rec.Code, rec.Body)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
	}{{composition.ErrNotFound, 404}, {composition.ErrForbidden, 403}, {composition.ErrConsentRequired, 403}, {errBackend, 500}} {
		if rec := getFHIR(t, &stubReader{err: tt.err}, true, "/fhir/R4/DocumentReference?encounter="+encA); rec.Code != tt.status || !strings.Contains(rec.Body.String(), "OperationOutcome") {
			t.Errorf("%v: %d %s", tt.err, rec.Code, rec.Body)
		}
	}
	if rec := getFHIR(t, &stubReader{}, false, "/fhir/R4/DocumentReference?encounter="+encA); rec.Code != 401 {
		t.Errorf("signed out = %d, want 401", rec.Code)
	}
}
