package api_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// The data request enums in api/openapi.yaml are the Go constants (docs/clinic-cms WAVE5.md, package 5.0).
func TestClinicWave5Contract_EnumsMatchTheGoConstants(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, want := range map[string][]string{
		"DataRequestKind": {string(datarequest.KindAccess), string(datarequest.KindPortability), string(datarequest.KindRectification),
			string(datarequest.KindRestriction), string(datarequest.KindObjection), string(datarequest.KindErasure)},
		"DataRequestTarget": {string(datarequest.TargetAccount), string(datarequest.TargetClinic)},
		"DataRequestStatus": {string(datarequest.StatusReceived), string(datarequest.StatusInProgress), string(datarequest.StatusFulfilled),
			string(datarequest.StatusRefused), string(datarequest.StatusWithdrawn)},
		"DataRequestAnswerCode": {string(datarequest.AnswerDone), string(datarequest.AnswerDonePartlyKept), string(datarequest.AnswerIdentityUnverified),
			string(datarequest.AnswerManifestlyUnfounded), string(datarequest.AnswerExcessive), string(datarequest.AnswerRetentionLaw),
			string(datarequest.AnswerNoData)},
		"DataRequestExtensionReason": {string(datarequest.ExtensionComplex), string(datarequest.ExtensionManyRequests)},
		"DataRequestExportStatus": {string(datarequest.ExportQueued), string(datarequest.ExportReady), string(datarequest.ExportFailed),
			string(datarequest.ExportExpired)},
	} {
		if got := s[name].Enum; !slices.Equal(got, want) {
			t.Errorf("%s: enum %v, want %v", name, got, want)
		}
	}
	answer := []string{string(datarequest.StatusFulfilled), string(datarequest.StatusRefused)}
	if got := s["DataRequestAnswer"].Properties["status"].Enum; !slices.Equal(got, answer) {
		t.Errorf("DataRequestAnswer.status: enum %v, want %v", got, answer)
	}
	for _, code := range []string{"invalid_data_request", "data_request_closed", "already_extended"} {
		if !slices.Contains(s["Error"].Properties["error"].Enum, code) {
			t.Errorf("Error.error: no %q", code)
		}
	}
}

// Every wave 5 route is in the contract and names the package that builds it; the operator has none (salusctl).
func TestClinicWave5Contract_EveryRouteNamesItsPackage(t *testing.T) {
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
		"get /me/data-requests":                     "cms-d1",
		"post /me/data-requests":                    "cms-d1",
		"post /me/data-requests/{id}/withdraw":      "cms-d1",
		"get /me/data-requests/{id}/export":         "cms-d2",
		"get /me/data-requests/{id}/export/archive": "cms-d2",
		"post /me/account/close":                    "cms-d3",
		"get /clinic/data-requests":                 "cms-d1",
		"get /clinic/data-requests/{id}":            "cms-d1",
		"post /clinic/data-requests/{id}/answer":    "cms-d1",
		"post /clinic/data-requests/{id}/extend":    "cms-d1",
		"post /clinic/patients/{id}/data-requests":  "cms-d1",
		"put /clinic/patients/{id}/restriction":     "cms-d4",
		"delete /clinic/patients/{id}/restriction":  "cms-d4",
	} {
		method, path, _ := strings.Cut(route, " ")
		op, _ := doc.Paths[path][method].(map[string]any)
		if got, _ := op["x-salus-package"].(string); got != want {
			t.Errorf("%s: x-salus-package %q, want %q", route, got, want)
		}
	}
	for path := range doc.Paths {
		if strings.Contains(path, "operator") {
			t.Errorf("%s: the operator has no route (salusctl, D5)", path)
		}
	}
}
