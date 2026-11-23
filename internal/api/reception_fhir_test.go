package api_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// What a receptionist gets on each FHIR interaction of api/fhir/CapabilityStatement.json (docs/clinic-cms F1, F3; W2):
// the desk reads its own clinic, the clinic's practitioners and their free times; every resource that holds a
// patient's data refuses them as a stranger. An interaction added to the CapabilityStatement fails
// TestReceptionFHIR_EveryInteractionIsClassified until it is classified here.
var receptionFHIR = map[string]receptionClass{
	"read Organization": desk, "search-type Organization": desk, "update Organization": refused,
	"read Practitioner": desk, "read PractitionerRole": desk, "search-type PractitionerRole": desk,
	"read Schedule": desk, "search-type Schedule": desk, "search-type Slot": desk,

	"read Patient": refused, "read Consent": refused,
	"read Appointment": refused, "search-type Appointment": refused,
	"read Encounter": refused, "search-type Encounter": refused, "read Condition": refused,
	"read AllergyIntolerance": refused, "search-type AllergyIntolerance": refused,
	"read MedicationStatement": refused, "search-type MedicationStatement": refused,
	"read Composition": refused, "read DocumentReference": refused, "search-type DocumentReference": refused,
	"read Provenance": refused, "read Questionnaire": refused, "search-type Questionnaire": refused,
	"read QuestionnaireResponse": refused, "search-type QuestionnaireResponse": refused,
}

// capabilityInteractions reads every "interaction Type" of the CapabilityStatement.
func capabilityInteractions(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("../../api/fhir/CapabilityStatement.json")
	if err != nil {
		t.Fatal(err)
	}
	var cs struct {
		Rest []struct {
			Resource []struct {
				Type        string
				Interaction []struct{ Code string }
			}
		}
	}
	if err := json.Unmarshal(b, &cs); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range cs.Rest[0].Resource {
		for _, i := range r.Interaction {
			out = append(out, i.Code+" "+r.Type)
		}
	}
	return out
}

func TestReceptionFHIR_EveryInteractionIsClassified(t *testing.T) {
	t.Parallel()
	all := map[string]bool{}
	for _, op := range capabilityInteractions(t) {
		all[op] = true
		if _, ok := receptionFHIR[op]; !ok {
			t.Errorf("%s is in the CapabilityStatement but not classified for the receptionist in receptionFHIR", op)
		}
	}
	for op := range receptionFHIR {
		if !all[op] {
			t.Errorf("%s is classified but not in the CapabilityStatement", op)
		}
	}
}

// fhirPath gives the request of the interaction op for the receptionist: the clinic's own resources (real) or ids
// that do not exist. ids holds the fixture's ids by resource type.
func fhirPath(op string, ids map[string]string, real bool) (method, path string) {
	interaction, typ, _ := strings.Cut(op, " ")
	id := func(t string) string {
		if v, ok := ids[t]; ok && (real || t == "Organization" || t == "PractitionerRole" || t == "Practitioner" || t == "Schedule") {
			return v // the directory's ids are public: always the clinic's own
		}
		return unknownID
	}
	switch interaction {
	case "read":
		return http.MethodGet, "/fhir/R4/" + typ + "/" + id(typ)
	case "update":
		return http.MethodPut, "/fhir/R4/" + typ + "/" + id(typ)
	}
	end := clock.System{}.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	query := map[string]string{
		"Organization":          "listed=true",
		"PractitionerRole":      "organization=Organization/" + id("Organization") + "&_include=PractitionerRole:practitioner",
		"Schedule":              "actor=PractitionerRole/" + id("PractitionerRole"),
		"Slot":                  "schedule=Schedule/" + id("Schedule") + "&status=free&start=lt" + end,
		"Appointment":           "patient=Patient/" + id("Patient"),
		"Encounter":             "patient=Patient/" + id("Patient"),
		"AllergyIntolerance":    "patient=Patient/" + id("Patient"),
		"MedicationStatement":   "patient=Patient/" + id("Patient"),
		"QuestionnaireResponse": "patient=Patient/" + id("Patient"),
		"DocumentReference":     "encounter=Encounter/" + unknownID,
		"Questionnaire":         "status=active",
	}[typ]
	return http.MethodGet, "/fhir/R4/" + typ + "?" + query
}

// fhirEmpty reports whether body is a searchset Bundle without entries.
func fhirEmpty(body string) bool {
	var b struct {
		ResourceType string
		Entry        []json.RawMessage
	}
	return json.Unmarshal([]byte(body), &b) == nil && b.ResourceType == "Bundle" && len(b.Entry) == 0
}

// A receptionist reads their clinic, its practitioners and their free times over FHIR, and is refused every resource
// that holds a patient's data, as a stranger: 403 or 404, or an empty search, the same for the clinic's own patient
// as for one that does not exist, with none of the patient's data. The desk's reads of the clinic and its roles are
// audited with the receptionist as the agent.
func TestReceptionFHIR_AReceptionistReadsTheDesksDirectoryAndNoPatientsRecord(t *testing.T) {
	r, count := newReception(t)
	ids := map[string]string{"Organization": inviteClinic, "Practitioner": r.practitionerID, "PractitionerRole": r.roleID,
		"Patient": r.patientID, "Appointment": r.visit}
	ids["Schedule"], ids["Consent"] = fixtureIDs(t, r)
	for _, op := range capabilityInteractions(t) {
		class := receptionFHIR[op]
		method, path := fhirPath(op, ids, true)
		r.receptionist.ifMatch = ""
		if method == http.MethodPut {
			r.receptionist.ifMatch = `W/"1"`
		}
		body := ""
		if method == http.MethodPut {
			body = `{"resourceType":"Organization","id":"` + inviteClinic + `","name":"Renamed by the desk"}`
		}
		rec := r.receptionist.do(method, path, body)
		got := rec.Body.String()
		switch class {
		case desk:
			if rec.Code != http.StatusOK || fhirEmpty(got) && op != "search-type Organization" {
				t.Errorf("%s %s as the receptionist = %d %s, want the clinic's own", method, path, rec.Code, got)
			}
		case refused:
			checkFHIRRefused(t, r, op, ids, rec.Code, got)
		}
	}
	for _, tt := range []struct{ name, entity string }{
		{"the clinic's read", "Organization/" + inviteClinic},
		{"the roles' search", "PractitionerRole/" + r.roleID},
	} {
		if n := count(`SELECT count(*) FROM audit_event WHERE agent_account_id = $1 AND outcome = 0 AND entity_reference = $2`,
			r.receptionAccount, tt.entity); n == 0 {
			t.Errorf("%s by the receptionist is not audited", tt.name)
		}
	}
}

// checkFHIRRefused holds a refused FHIR interaction to a stranger's answer.
func checkFHIRRefused(t *testing.T, r reception, op string, ids map[string]string, code int, body string) {
	t.Helper()
	empty := code == http.StatusOK && fhirEmpty(body)
	if code != http.StatusForbidden && code != http.StatusNotFound && !empty {
		t.Errorf("%s as the receptionist = %d %s, want 403, 404 or an empty search", op, code, body)
	}
	for _, s := range bookedPatientData {
		if strings.Contains(body, s) {
			t.Errorf("%s as the receptionist: the body holds the patient's data", op)
		}
	}
	method, real := fhirPath(op, ids, true)
	if _, unknown := fhirPath(op, ids, false); unknown != real {
		rec := r.receptionist.do(method, unknown, "")
		if rec.Code != code || empty != (rec.Code == http.StatusOK && fhirEmpty(rec.Body.String())) ||
			!empty && rec.Body.String() != body {
			t.Errorf("%s as the receptionist = %d %s for the clinic's own, %d %s for an unknown id: the answer reveals what exists",
				op, code, body, rec.Code, rec.Body)
		}
	}
}

// fixtureIDs reads the fixture's schedule and the patient's care consent.
func fixtureIDs(t *testing.T, r reception) (schedule, consent string) {
	t.Helper()
	seedFree(t, r, 2)
	if err := r.owner.QueryRow(t.Context(), `SELECT (SELECT id::text FROM schedule), (SELECT id::text FROM consent WHERE patient_id = $1 LIMIT 1)`,
		r.patientID).Scan(&schedule, &consent); err != nil {
		t.Fatal(err)
	}
	return schedule, consent
}

// seedFree adds a free in-person slot days ahead to the fixture's schedule (the booked visit holds the other one).
func seedFree(t *testing.T, r reception, days int) {
	t.Helper()
	if _, err := r.owner.Exec(t.Context(), `INSERT INTO slot (schedule_id, organization_id, period, visit_mode)
		SELECT id, organization_id, tstzrange(date_trunc('minute', now()) + make_interval(days => $1), date_trunc('minute', now()) + make_interval(days => $1, mins => 30)), 'in-person'
		FROM schedule`, days); err != nil {
		t.Fatal(err)
	}
}
