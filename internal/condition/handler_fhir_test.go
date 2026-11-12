package condition_test

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/condition"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// stubReader answers Get with its Condition or error.
type stubReader struct {
	err error
	c   condition.Condition
}

func (s stubReader) Get(context.Context, appointment.Caller, string) (condition.Condition, error) {
	return s.c, s.err
}

func serveFHIR(t *testing.T, svc condition.Reader, signed bool, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (appointment.Caller, bool, error) { return practitioner, signed, nil }
	condition.NewFHIRHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

// GET /fhir/R4/Condition/{id}: the Condition as FHIR R4 JSON, its version as the ETag (testdata/condition.json).
func TestFHIRRead_WritesTheCondition(t *testing.T) {
	c := valid()
	c.ID, c.VersionID, c.LastUpdated = condID, 1, signedAt.Add(time.Second)
	rec := serveFHIR(t, stubReader{c: c}, true, "/fhir/R4/Condition/"+condID)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/fhir+json" || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("GET = %d %v", rec.Code, rec.Header())
	}
	golden := filepath.Join("testdata", "condition.json")
	if *update {
		if err := os.WriteFile(golden, rec.Body.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Body.String() != string(want) {
		t.Errorf("GET =\n%s\nwant\n%s", rec.Body, want)
	}
}

func TestFHIRRead_MapsTheErrors(t *testing.T) {
	for _, tt := range []struct {
		err    error
		signed bool
		want   int
	}{
		{nil, false, http.StatusUnauthorized},
		{condition.ErrNotFound, true, http.StatusNotFound},
		{condition.ErrForbidden, true, http.StatusForbidden},
		{condition.ErrConsentRequired, true, http.StatusForbidden},
		{context.DeadlineExceeded, true, http.StatusInternalServerError},
	} {
		if rec := serveFHIR(t, stubReader{err: tt.err}, tt.signed, "/fhir/R4/Condition/"+condID); rec.Code != tt.want ||
			rec.Header().Get("Content-Type") != "application/fhir+json" {
			t.Errorf("%v: %d %s, want %d as an OperationOutcome", tt.err, rec.Code, rec.Header().Get("Content-Type"), tt.want)
		}
	}
}
