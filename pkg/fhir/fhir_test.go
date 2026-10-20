package fhir_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

func TestWriteError_SendsOperationOutcomeWithOneErrorIssue(t *testing.T) {
	rec := httptest.NewRecorder()

	fhir.WriteError(rec, http.StatusNotFound, "not-found", "No such resource.")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("Content-Type = %q, want application/fhir+json", ct)
	}
	want := `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"No such resource."}]}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestWriteError_OmitsEmptyDiagnostics(t *testing.T) {
	rec := httptest.NewRecorder()

	fhir.WriteError(rec, http.StatusBadRequest, "invalid", "")

	want := `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"invalid"}]}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}
