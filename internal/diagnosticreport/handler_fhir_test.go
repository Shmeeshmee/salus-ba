package diagnosticreport_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
)

func TestFHIR_WritesAReleasedReportWithItsObservationsContained(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	released := time.Date(2026, 11, 24, 15, 30, 0, 0, time.FixedZone("CET", 3600))
	r := diagnosticreport.Report{
		ID: "r1", OrganizationID: "o1", PatientID: "p1", AppointmentID: "a1", Status: diagnosticreport.StatusFinal,
		Title: "Complete blood count", LOINC: "58410-2", EffectiveDate: &day, ReleasedAt: &released, VersionID: 3,
		Results: []diagnosticreport.Result{
			{Test: "Haemoglobin", LOINC: "718-7", Value: ptr(13.5), Unit: "g/dL", ReferenceRange: "12-16", Interpretation: "N"},
			{Test: "Synthetic marker", ValueText: "negative"},
		},
	}
	got, err := json.Marshal(diagnosticreport.FHIR(r))
	want := `{"resourceType":"DiagnosticReport","id":"r1","meta":{"lastUpdated":"2026-11-24T14:30:00Z","versionId":"3"},` +
		`"contained":[` +
		`{"valueQuantity":{"unit":"g/dL","system":"http://unitsofmeasure.org","code":"g/dL","value":13.5},"resourceType":"Observation","id":"result-1",` +
		`"status":"final","code":{"text":"Haemoglobin","coding":[{"system":"http://loinc.org","code":"718-7"}]},"subject":{"reference":"Patient/p1"},` +
		`"interpretation":[{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation","code":"N"}]}],` +
		`"referenceRange":[{"text":"12-16"}]},` +
		`{"resourceType":"Observation","id":"result-2","status":"final","code":{"text":"Synthetic marker"},"subject":{"reference":"Patient/p1"},"valueString":"negative"}],` +
		`"status":"final","code":{"text":"Complete blood count","coding":[{"system":"http://loinc.org","code":"58410-2"}]},` +
		`"subject":{"reference":"Patient/p1"},"effectiveDateTime":"2026-11-20","issued":"2026-11-24T14:30:00Z",` +
		`"performer":[{"reference":"Organization/o1"}],"result":[{"reference":"#result-1"},{"reference":"#result-2"}]}`
	if err != nil || string(got) != want {
		t.Errorf("%s, %v\nwant %s", got, err, want)
	}
}
