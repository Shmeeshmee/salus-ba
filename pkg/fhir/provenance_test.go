package fhir_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// A write's Provenance targets the version written and, for a change, names the version it revised; the agent is
// the role on behalf of its organization; written once (version 1, last updated when recorded).
func TestNewWriteProvenance(t *testing.T) {
	at := time.Date(2030, 1, 7, 9, 45, 0, 0, time.FixedZone("CET", 3600))
	for name, tt := range map[string]struct {
		want string
		w    fhir.Change
	}{
		"a creation": {
			w: fhir.Change{ID: "p1", Resource: "AllergyIntolerance/a1", Version: 1, Recorded: at, Activity: fhir.ActivityCreate,
				AgentType: fhir.AgentAuthor, Who: "PractitionerRole/r1", OnBehalfOf: "Organization/o1"},
			want: `{"resourceType":"Provenance","id":"p1","meta":{"lastUpdated":"2030-01-07T08:45:00Z","versionId":"1"},` +
				`"target":[{"reference":"AllergyIntolerance/a1/_history/1"}],"recorded":"2030-01-07T08:45:00Z",` +
				`"activity":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/v3-DataOperation","code":"CREATE"}]},` +
				`"agent":[{"onBehalfOf":{"reference":"Organization/o1"},"who":{"reference":"PractitionerRole/r1"},` +
				`"type":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/provenance-participant-type","code":"author"}]}}]}`,
		},
		"a confirmation": {
			w: fhir.Change{ID: "p2", Resource: "MedicationStatement/m1", Version: 3, Recorded: at, Activity: fhir.ActivityUpdate,
				AgentType: fhir.AgentVerifier, Who: "PractitionerRole/r1", OnBehalfOf: "Organization/o1"},
			want: `{"resourceType":"Provenance","id":"p2","meta":{"lastUpdated":"2030-01-07T08:45:00Z","versionId":"1"},` +
				`"target":[{"reference":"MedicationStatement/m1/_history/3"}],"recorded":"2030-01-07T08:45:00Z",` +
				`"activity":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/v3-DataOperation","code":"UPDATE"}]},` +
				`"agent":[{"onBehalfOf":{"reference":"Organization/o1"},"who":{"reference":"PractitionerRole/r1"},` +
				`"type":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/provenance-participant-type","code":"verifier"}]}}],` +
				`"entity":[{"role":"revision","what":{"reference":"MedicationStatement/m1/_history/2"}}]}`,
		},
	} {
		got, err := json.Marshal(fhir.NewWriteProvenance(tt.w))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", name, got, tt.want)
		}
	}
}
