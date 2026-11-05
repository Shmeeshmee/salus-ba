package patient_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
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

// stubReader answers with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller patient.Caller
	id     string
	p      patient.Patient
	calls  int
}

func (s *stubReader) Read(_ context.Context, c patient.Caller, id string) (patient.Patient, error) {
	s.caller, s.id = c, id
	s.calls++
	return s.p, s.err
}

var practitionerA = patient.Caller{AccountID: "a1000000-0000-4000-8000-0000000000a1", OrganizationID: "a0000000-0000-4000-8000-00000000000a",
	PractitionerRoleID: "a4000000-0000-4000-8000-0000000000a1"}

func readFHIR(t *testing.T, svc *stubReader, caller func(context.Context) (patient.Caller, bool, error), log *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	patient.NewFHIRHandler(svc, caller, slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fhir/R4/Patient/"+patientA, nil))
	return rec
}

func as(c patient.Caller) func(context.Context) (patient.Caller, bool, error) {
	return func(context.Context) (patient.Caller, bool, error) { return c, true, nil }
}

func TestFHIRHandlerRead_SendsThePatientWithItsNameBirthDatePhoneMetaAndETag(t *testing.T) {
	svc := &stubReader{p: savedA}
	var log bytes.Buffer

	rec := readFHIR(t, svc, as(practitionerA), &log)

	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || rec.Header().Get("Content-Type") != "application/fhir+json" {
		t.Fatalf("status = %d %s %v, want 200 FHIR JSON at version 2", rec.Code, rec.Body, rec.Header())
	}
	if svc.caller != practitionerA || svc.id != patientA {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
	}
	assertGolden(t, "patient.json", rec.Body.Bytes())

	noPhone := savedA
	noPhone.Phone = ""
	rec = readFHIR(t, &stubReader{p: noPhone}, as(practitionerA), &log)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "telecom") {
		t.Errorf("without a phone = %d %s, want no telecom", rec.Code, rec.Body)
	}
}

func TestFHIRHandlerRead_AnswersNotFoundAndAsksForASignIn(t *testing.T) {
	var log bytes.Buffer
	rec := readFHIR(t, &stubReader{err: patient.ErrNotFound}, as(practitionerA), &log)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())

	svc := &stubReader{}
	rec = readFHIR(t, svc, func(context.Context) (patient.Caller, bool, error) { return patient.Caller{}, false, nil }, &log)
	if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
		t.Errorf("signed out: status = %d after %d calls, want 401 and none", rec.Code, svc.calls)
	}
	assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
}

func TestFHIRHandlerRead_SendsAFixedOperationOutcomeAndLogsNoProfileField(t *testing.T) {
	failing := errors.New("db down")
	for name, tc := range map[string]struct {
		svc    *stubReader
		caller func(context.Context) (patient.Caller, bool, error)
	}{
		"the service": {&stubReader{err: failing, p: savedA}, as(practitionerA)},
		"the caller":  {&stubReader{p: savedA}, func(context.Context) (patient.Caller, bool, error) { return patient.Caller{}, false, failing }},
	} {
		var log bytes.Buffer
		rec := readFHIR(t, tc.svc, tc.caller, &log)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || log.Len() == 0 {
			t.Errorf("%s failing: status = %d %s, log %q; want 500 without the error, the error logged", name, rec.Code, rec.Body, log.String())
		}
		for _, field := range []string{"Ada", "Example", "1990", "38760000001"} {
			if strings.Contains(log.String(), field) {
				t.Errorf("%s failing: the log holds %q", name, field)
			}
		}
	}
}
