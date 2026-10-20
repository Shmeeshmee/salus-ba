package fhir_test

import (
	"encoding/json"
	"testing"

	"github.com/Shmeeshmee/salus/api/fhir"
)

func TestCapabilityStatement_IsAnR4CapabilityStatement(t *testing.T) {
	var cs struct {
		ResourceType string   `json:"resourceType"`
		FHIRVersion  string   `json:"fhirVersion"`
		Format       []string `json:"format"`
	}
	if err := json.Unmarshal([]byte(fhir.CapabilityStatement()), &cs); err != nil {
		t.Fatalf("CapabilityStatement is not JSON: %v", err)
	}

	if cs.ResourceType != "CapabilityStatement" {
		t.Errorf("resourceType = %q, want CapabilityStatement", cs.ResourceType)
	}
	if cs.FHIRVersion != "4.0.1" {
		t.Errorf("fhirVersion = %q, want 4.0.1", cs.FHIRVersion)
	}
	if len(cs.Format) != 1 || cs.Format[0] != "application/fhir+json" {
		t.Errorf("format = %q, want [application/fhir+json]", cs.Format)
	}
}
