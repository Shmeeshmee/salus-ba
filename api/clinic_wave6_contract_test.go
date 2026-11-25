package api_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/careplan"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/internal/location"
)

// The wave 6 schemas in api/openapi.yaml have the Go models' fields, required exactly when Go always encodes them,
// and the Go constants' enums (docs/clinic-cms/WAVE6.md, package 6.0).
func TestClinicWave6Contract_SchemasMatchTheGoModels(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, value := range map[string]any{
		"Location": location.Location{}, "Room": location.Room{}, "Device": location.Device{},
		"DiagnosticReport": diagnosticreport.Report{}, "DiagnosticReportResult": diagnosticreport.Result{},
		"CarePlan": careplan.Plan{}, "CarePlanStep": careplan.Step{},
	} {
		all, required := jsonFields(value)
		var props []string
		for p := range s[name].Properties {
			props = append(props, p)
		}
		slices.Sort(props)
		if !slices.Equal(props, all) {
			t.Errorf("%s: properties %v, want the Go fields %v", name, props, all)
		}
		if req := slices.Sorted(slices.Values(s[name].Required)); !slices.Equal(req, required) {
			t.Errorf("%s: required %v, want %v", name, req, required)
		}
	}
	var flags []string
	for _, f := range diagnosticreport.Interpretations {
		flags = append(flags, string(f))
	}
	for name, want := range map[string][]string{
		"DiagnosticReportStatus":    {string(diagnosticreport.StatusPreliminary), string(diagnosticreport.StatusFinal)},
		"ObservationInterpretation": flags,
		"CarePlanStatus": {string(careplan.StatusDraft), string(careplan.StatusActive), string(careplan.StatusCompleted),
			string(careplan.StatusRevoked)},
	} {
		if got := s[name].Enum; !slices.Equal(got, want) {
			t.Errorf("%s: enum %v, want %v", name, got, want)
		}
	}
	for _, code := range []string{"invalid_location", "resource_busy", "invalid_diagnostic_report", "report_released",
		"invalid_care_plan", "care_plan_not_draft", "care_plan_ended"} {
		if !slices.Contains(s["Error"].Properties["error"].Enum, code) {
			t.Errorf("Error.error: no %q", code)
		}
	}
}

// Every wave 6 route is in the contract and names the package that builds it.
func TestClinicWave6Contract_EveryRouteNamesItsPackage(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for route, want := range map[string]string{
		"get /clinic/locations":                             "cms-lc1",
		"post /clinic/locations":                            "cms-lc1",
		"put /clinic/locations/{id}":                        "cms-lc1",
		"get /organizations/{id}/locations":                 "cms-lc1",
		"get /clinic/locations/{id}/rooms":                  "cms-rm1",
		"post /clinic/locations/{id}/rooms":                 "cms-rm1",
		"put /clinic/rooms/{id}":                            "cms-rm1",
		"get /clinic/locations/{id}/devices":                "cms-rm1",
		"post /clinic/locations/{id}/devices":               "cms-rm1",
		"put /clinic/devices/{id}":                          "cms-rm1",
		"get /clinic/appointments/{id}/diagnostic-reports":  "cms-lb1",
		"post /clinic/appointments/{id}/diagnostic-reports": "cms-lb1",
		"get /clinic/diagnostic-reports/{id}":               "cms-lb1",
		"put /clinic/diagnostic-reports/{id}":               "cms-lb1",
		"get /clinic/diagnostic-reports/{id}/pdf":           "cms-lb1",
		"put /clinic/diagnostic-reports/{id}/pdf":           "cms-lb1",
		"post /clinic/diagnostic-reports/{id}/release":      "cms-lb1",
		"get /me/diagnostic-reports":                        "cms-lb1",
		"get /me/diagnostic-reports/{id}":                   "cms-lb1",
		"get /me/diagnostic-reports/{id}/pdf":               "cms-lb1",
		"post /clinic/care-plans":                           "cms-tp1",
		"get /clinic/patients/{id}/care-plans":              "cms-tp1",
		"get /clinic/care-plans/{id}":                       "cms-tp1",
		"put /clinic/care-plans/{id}":                       "cms-tp1",
		"post /clinic/care-plans/{id}/book":                 "cms-tp1",
		"post /clinic/care-plans/{id}/cancel":               "cms-tp1",
		"get /me/care-plans":                                "cms-tp1",
	} {
		method, path, _ := strings.Cut(route, " ")
		op, _ := doc.Paths[path][method].(map[string]any)
		if got, _ := op["x-salus-package"].(string); got != want {
			t.Errorf("%s: x-salus-package %q, want %q", route, got, want)
		}
	}
}
