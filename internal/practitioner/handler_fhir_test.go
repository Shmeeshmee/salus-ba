package practitioner_test

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

	"github.com/Shmeeshmee/salus/internal/practitioner"
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

// stubReader answers Read with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller practitioner.Caller
	id     string
	p      practitioner.Practitioner
	calls  int
}

func (s *stubReader) Read(_ context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error) {
	s.caller, s.id = c, id
	s.calls++
	return s.p, s.err
}

func signedIn(c practitioner.Caller) func(context.Context) (practitioner.Caller, bool) {
	return func(context.Context) (practitioner.Caller, bool) { return c, true }
}

func nobody(context.Context) (practitioner.Caller, bool) { return practitioner.Caller{}, false }

func serveFHIR(t *testing.T, svc *stubReader, caller func(context.Context) (practitioner.Caller, bool), log *slog.Logger, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	practitioner.NewHandler(svc, caller, log).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestHandlerRead_SendsThePractitionerWithItsMetaAndETag(t *testing.T) {
	svc := &stubReader{p: practitioner.Practitioner{
		ID: practitionerA, OrganizationID: clinicA, GivenName: "Cy", FamilyName: "Example",
		VersionID: 2, LastUpdated: now, Active: true,
	}}

	rec := serveFHIR(t, svc, signedIn(staffA), discard(), "/fhir/R4/Practitioner/"+practitionerA)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if etag := rec.Header().Get("ETag"); etag != `W/"2"` {
		t.Errorf(`ETag = %q, want W/"2"`, etag)
	}
	if svc.caller != staffA || svc.id != practitionerA {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
	}
	assertGolden(t, "practitioner.json", rec.Body.Bytes())
}

func TestHandlerRead_AnswersNotFoundForAPractitionerTheCallerMayNotSee(t *testing.T) {
	svc := &stubReader{err: practitioner.ErrNotFound}

	rec := serveFHIR(t, svc, signedIn(staffA), discard(), "/fhir/R4/Practitioner/"+practitionerB)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())
}

func TestHandlerRead_AsksForASignInWithoutCallingTheService(t *testing.T) {
	svc := &stubReader{}

	rec := serveFHIR(t, svc, nobody, discard(), "/fhir/R4/Practitioner/"+practitionerA)

	if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
		t.Errorf("status = %d after %d service calls, want 401 and none", rec.Code, svc.calls)
	}
	assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
}

func TestHandlerRead_SendsAFixedOperationOutcomeAndLogsNoName(t *testing.T) {
	var logs bytes.Buffer
	svc := &stubReader{err: errors.New("practitioner: read: db down"), p: practitioner.Practitioner{GivenName: "Cy", FamilyName: "Example"}}

	rec := serveFHIR(t, svc, signedIn(staffA), slog.New(slog.NewJSONHandler(&logs, nil)), "/fhir/R4/Practitioner/"+practitionerA)

	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Errorf("response = %d %s, want 500 without the error", rec.Code, rec.Body)
	}
	if logs.Len() == 0 {
		t.Fatal("nothing was logged")
	}
	for _, name := range []string{"Cy", "Example"} {
		if strings.Contains(logs.String(), name) {
			t.Errorf("the log holds %q: %s", name, logs.String())
		}
	}
}
