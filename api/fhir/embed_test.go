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

// TestCapabilityStatement_ListsExactlyTheDirectoryInteractions: docs/accounts/SPEC.md "FHIR interactions".
func TestCapabilityStatement_ListsExactlyTheDirectoryInteractions(t *testing.T) {
	var cs struct {
		Rest []struct {
			Mode     string `json:"mode"`
			Resource []struct {
				Type        string `json:"type"`
				Versioning  string `json:"versioning"`
				Interaction []struct {
					Code string `json:"code"`
				} `json:"interaction"`
				SearchInclude []string `json:"searchInclude"`
				SearchParam   []struct {
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"searchParam"`
			} `json:"resource"`
		} `json:"rest"`
	}
	if err := json.Unmarshal([]byte(fhir.CapabilityStatement()), &cs); err != nil {
		t.Fatalf("CapabilityStatement is not JSON: %v", err)
	}
	if len(cs.Rest) != 1 || cs.Rest[0].Mode != "server" {
		t.Fatalf("rest = %+v, want one server entry", cs.Rest)
	}
	got := map[string]string{}
	for _, r := range cs.Rest[0].Resource {
		var s string
		for _, i := range r.Interaction {
			s += i.Code + " "
		}
		for _, p := range r.SearchParam {
			s += "param:" + p.Name + ":" + p.Type + " "
		}
		for _, inc := range r.SearchInclude {
			s += "include:" + inc + " "
		}
		if r.Versioning != "" {
			s += "versioning:" + r.Versioning
		}
		got[r.Type] = s
	}
	want := map[string]string{
		"Organization":     "read update versioning:versioned-update",
		"Practitioner":     "read ",
		"PractitionerRole": "read search-type param:organization:reference include:PractitionerRole:practitioner ",
	}
	if len(got) != len(want) {
		t.Errorf("resources = %v, want %v", got, want)
	}
	for typ, w := range want {
		if got[typ] != w {
			t.Errorf("%s = %q, want %q", typ, got[typ], w)
		}
	}
}
