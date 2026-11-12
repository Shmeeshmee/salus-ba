package consent_test

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
	"time"

	"github.com/Shmeeshmee/salus/internal/consent"
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

const consentID = "e1000000-0000-4000-8000-0000000000e1"

// stubReader answers with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller consent.Caller
	id     string
	out    consent.Consent
	calls  int
}

func (s *stubReader) Read(_ context.Context, c consent.Caller, id string) (consent.Consent, error) {
	s.caller, s.id = c, id
	s.calls++
	return s.out, s.err
}

var staffCaller = consent.Caller{AccountID: staffA, OrganizationID: clinicA}

func readFHIR(t *testing.T, svc *stubReader, caller func(context.Context) (consent.Caller, bool, error), log *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	consent.NewFHIRHandler(svc, caller, slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fhir/R4/Consent/"+consentID, nil))
	return rec
}

func as(c consent.Caller) func(context.Context) (consent.Caller, bool, error) {
	return func(context.Context) (consent.Caller, bool, error) { return c, true, nil }
}

func TestFHIRHandlerRead_SendsTheConsentAsSPECConsentShapeSaysWithMetaAndETag(t *testing.T) {
	active := consent.Consent{ID: consentID, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindCare, Status: consent.StatusActive,
		DateTime: now.Add(-24 * time.Hour), VersionID: 1, LastUpdated: now.Add(-24 * time.Hour)}
	svc := &stubReader{out: active}
	var log bytes.Buffer

	rec := readFHIR(t, svc, as(staffCaller), &log)

	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d %s %v, want 200 at version 1, not stored", rec.Code, rec.Body, rec.Header())
	}
	if svc.caller != staffCaller || svc.id != consentID {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
	}
	assertGolden(t, "consent.json", rec.Body.Bytes())

	revoked := active
	revoked.Status, revoked.RevokedAt, revoked.VersionID = consent.StatusInactive, now, 2
	rec = readFHIR(t, &stubReader{out: revoked}, as(staffCaller), &log)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"inactive"`) {
		t.Errorf("a revoked consent = %d %s, want status inactive", rec.Code, rec.Body)
	}
}

// The transcription consent is a Consent of its own: information collection (v3-ActCode ICOL) beside the
// patient consent category, the provision permitting the clinic to collect and use what is said for treatment.
func TestFHIRHandlerRead_SendsATranscriptionConsentWithItsCategoryAndActions(t *testing.T) {
	tc := consent.Consent{ID: consentID, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindTranscription,
		Status: consent.StatusActive, DateTime: now.Add(-24 * time.Hour), VersionID: 1, LastUpdated: now.Add(-24 * time.Hour)}
	rec := readFHIR(t, &stubReader{out: tc}, as(me), &bytes.Buffer{})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "transcription_consent.json", rec.Body.Bytes())
}

func TestFHIRHandlerRead_AnswersNotFoundAndAsksForASignIn(t *testing.T) {
	var log bytes.Buffer
	rec := readFHIR(t, &stubReader{err: consent.ErrNotFound}, as(staffCaller), &log)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())

	svc := &stubReader{}
	rec = readFHIR(t, svc, func(context.Context) (consent.Caller, bool, error) { return consent.Caller{}, false, nil }, &log)
	if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
		t.Errorf("signed out: status = %d after %d calls, want 401 and none", rec.Code, svc.calls)
	}
	assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
}

func TestFHIRHandlerRead_SendsAFixedOperationOutcomeOnAnError(t *testing.T) {
	failing := errors.New("db down")
	for name, tc := range map[string]struct {
		svc    *stubReader
		caller func(context.Context) (consent.Caller, bool, error)
	}{
		"the service": {&stubReader{err: failing}, as(staffCaller)},
		"the caller":  {&stubReader{}, func(context.Context) (consent.Caller, bool, error) { return consent.Caller{}, false, failing }},
	} {
		var log bytes.Buffer
		rec := readFHIR(t, tc.svc, tc.caller, &log)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || log.Len() == 0 {
			t.Errorf("%s failing: status = %d %s, log %q; want 500 without the error, the error logged", name, rec.Code, rec.Body, log.String())
		}
	}
}

// The history consent (ADR 0013) is a Consent of its own: information disclosure (v3-ActCode IDSCL) beside the
// patient consent category, the provision permitting the receiving clinic the disclosure, for treatment, of the
// listed visits and what refers to them (meaning dependents: their documents, answers and files), and nothing else.
func TestFHIRHandlerRead_SendsAHistoryConsentWithTheVisitsItShares(t *testing.T) {
	hc := consent.Consent{ID: consentID, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindHistory,
		Status: consent.StatusActive, DateTime: now.Add(-24 * time.Hour), VersionID: 3, LastUpdated: now,
		Shared: []consent.SharedVisit{
			{EncounterID: "e2000000-0000-4000-8000-0000000000e2", AddedAt: now.Add(-24 * time.Hour)},
			{EncounterID: "e3000000-0000-4000-8000-0000000000e3", AddedAt: now},
		}}
	rec := readFHIR(t, &stubReader{out: hc}, as(me), &bytes.Buffer{})
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"3"` {
		t.Fatalf("status = %d %s, want 200 at version 3", rec.Code, rec.Body)
	}
	assertGolden(t, "history_consent.json", rec.Body.Bytes())

	// Revoked, it permits nothing: no visit is listed.
	hc.Status, hc.RevokedAt, hc.Shared = consent.StatusInactive, now, nil
	rec = readFHIR(t, &stubReader{out: hc}, as(me), &bytes.Buffer{})
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"data"`) {
		t.Errorf("a revoked history consent = %d %s, want no data", rec.Code, rec.Body)
	}
}
