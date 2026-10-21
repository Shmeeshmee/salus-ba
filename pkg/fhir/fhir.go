// Package fhir holds generic FHIR R4 primitives with no knowledge of this product.
package fhir

import (
	"encoding/json"
	"net/http"
)

// ContentType is the media type of every FHIR response.
const ContentType = "application/fhir+json"

// OperationOutcome reports an error on a FHIR endpoint.
type OperationOutcome struct {
	ResourceType string  `json:"resourceType"`
	Issue        []Issue `json:"issue"`
}

// Issue is one problem in an OperationOutcome. Severity and Code come from the FHIR value sets
// issue-severity and issue-type. Diagnostics never carries personal data or internal error text.
type Issue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Diagnostics string `json:"diagnostics,omitempty"`
}

// WriteError sends an OperationOutcome with one error issue.
func WriteError(w http.ResponseWriter, status int, code, diagnostics string) {
	Write(w, status, OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue:        []Issue{{Severity: "error", Code: code, Diagnostics: diagnostics}},
	})
}

// Write sends v as a FHIR JSON response.
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	// The status is already sent; a failed write means the client went away.
	_ = json.NewEncoder(w).Encode(v)
}
