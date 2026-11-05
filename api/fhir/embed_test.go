package fhir_test

import (
	"encoding/json"
	"os"
	"strings"
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

// TestCapabilityStatement_ListsExactlyTheInteractionsServed: docs/accounts/SPEC.md "FHIR interactions" and
// docs/booking/SPEC.md "FHIR reads and searches".
func TestCapabilityStatement_ListsExactlyTheInteractionsServed(t *testing.T) {
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
		"Appointment":      "read search-type param:patient:reference param:practitioner:reference param:date:date param:status:token ",
		"Consent":          "read ",
		"Organization":     "read update search-type param:listed:token versioning:versioned-update",
		"Patient":          "read ",
		"Practitioner":     "read ",
		"PractitionerRole": "read search-type param:organization:reference include:PractitionerRole:practitioner ",
		"Schedule":         "read search-type param:actor:reference ",
		"Slot":             "search-type param:schedule:reference param:start:date param:status:token ",
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

// The listed search parameter is defined by our SearchParameter, and it and the extensions on Organization are
// the documents in this folder.
func TestCapabilityStatement_DefinesListedByItsSearchParameter(t *testing.T) {
	var sp struct {
		ResourceType string   `json:"resourceType"`
		URL          string   `json:"url"`
		Code         string   `json:"code"`
		Type         string   `json:"type"`
		Expression   string   `json:"expression"`
		Base         []string `json:"base"`
	}
	raw, err := os.ReadFile("SearchParameter-organization-listed.json")
	if err != nil || json.Unmarshal(raw, &sp) != nil {
		t.Fatalf("SearchParameter-organization-listed.json: %v", err)
	}
	const listed = "https://salus.invalid/fhir/StructureDefinition/organization-listed"
	if sp.ResourceType != "SearchParameter" || sp.Code != "listed" || sp.Type != "token" || len(sp.Base) != 1 || sp.Base[0] != "Organization" ||
		!strings.Contains(sp.Expression, listed) {
		t.Errorf("SearchParameter = %+v, want the token listed on Organization over the listed extension", sp)
	}
	if !strings.Contains(fhir.CapabilityStatement(), `"definition": "`+sp.URL+`"`) {
		t.Errorf("the CapabilityStatement does not define listed by %s", sp.URL)
	}
	for _, name := range []string{"listed", "timezone"} {
		var sd struct {
			URL     string `json:"url"`
			Type    string `json:"type"`
			Context []struct {
				Expression string `json:"expression"`
			} `json:"context"`
		}
		raw, err := os.ReadFile("StructureDefinition-organization-" + name + ".json")
		if err != nil || json.Unmarshal(raw, &sd) != nil {
			t.Fatalf("StructureDefinition-organization-%s.json: %v", name, err)
		}
		if sd.URL != "https://salus.invalid/fhir/StructureDefinition/organization-"+name || sd.Type != "Extension" ||
			len(sd.Context) != 1 || sd.Context[0].Expression != "Organization" {
			t.Errorf("organization-%s = %+v, want an extension of Organization at its URL", name, sd)
		}
	}
}
