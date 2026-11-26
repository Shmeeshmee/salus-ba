package api_test

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The role matrix (docs/clinic-cms/WAVE7.md RD1): every route of api/openapi.yaml and every interaction of the FHIR
// CapabilityStatement, for each of the patient, the practitioner, the receptionist, the admin and a signed-out caller,
// on the caller's own clinic and data, on another clinic's, and for a patient on another patient's. The routes come
// from the contract, so a route added later is held here without a line, and fails until it is classified.

// pathKinds names the fixture's id kind a path parameter takes, by the path segment before it.
var pathKinds = map[string]string{
	"appointments": "visit", "patients": "patient", "practitioner-roles": "role", "organizations": "org",
	"history-sharing": "org", "receipts": "receipt", "data-requests": "request", "locations": "location",
	"rooms": "room", "devices": "device", "quotes": "quote", "diagnostic-reports": "report", "care-plans": "plan",
	"questionnaires": "questionnaire", "questionnaire-responses": "response", "conversations": "conversation",
	"webhooks": "webhook", "document-templates": "template", "waitlist": "waitlist", "export": "export",
	"encounters": "encounter", "allergies": "allergy", "medications": "medication", "invoices": "invoice",
	"consents": "consent", "services": "service", "invites": "invite", "binaries": "binary", "messages": "message",
	"attachments": "attachment",
}

// matrixPath fills route's parameters from ids (by kind; unknownID for a kind ids lacks) and adds its query.
func matrixPath(route string, ids map[string]string) string {
	segs := strings.Split(route, "/")
	for i, s := range segs {
		if !strings.HasPrefix(s, "{") {
			continue
		}
		switch s {
		case "{token}":
			segs[i] = "synthetic-token"
		case "{language}":
			segs[i] = "bs"
		case "{code}":
			segs[i] = "A00"
		case "{linkId}":
			segs[i] = "q1"
		default:
			segs[i] = unknownID
			if id, ok := ids[pathKinds[segs[i-1]]]; ok {
				segs[i] = id
			}
		}
	}
	path := strings.Join(segs, "/")
	if q, ok := receptionQueries[route]; ok {
		path += "?" + q
	}
	return path
}

// fhirKinds names the fixture's id kind of a FHIR resource type.
var fhirKinds = map[string]string{
	"Patient": "patient", "Appointment": "visit", "Organization": "org", "Practitioner": "practitioner",
	"PractitionerRole": "role", "Schedule": "schedule", "Consent": "consent", "Encounter": "encounter",
	"AllergyIntolerance": "allergy", "MedicationStatement": "medication", "Questionnaire": "questionnaire",
	"QuestionnaireResponse": "response",
}

// fhirRequest gives the request of the FHIR interaction op ("read Patient") on ids.
func fhirRequest(op string, ids map[string]string) (method, path, body string) {
	interaction, typ, _ := strings.Cut(op, " ")
	id := func(typ string) string {
		if v, ok := ids[fhirKinds[typ]]; ok {
			return v
		}
		return unknownID
	}
	switch interaction {
	case "read":
		return http.MethodGet, "/fhir/R4/" + typ + "/" + id(typ), ""
	case "update":
		return http.MethodPut, "/fhir/R4/" + typ + "/" + id(typ), `{"resourceType":"` + typ + `","id":"` + id(typ) + `","name":"Renamed synthetic clinic"}`
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
		"DocumentReference":     "encounter=Encounter/" + id("Encounter"),
		"Questionnaire":         "status=active",
	}[typ]
	return http.MethodGet, "/fhir/R4/" + typ + "?" + query, ""
}

// matrixOps are every operation of the contract and every FHIR interaction, in order ("FHIR read Patient" for
// these), without the two that end the caller's session.
func matrixOps(t *testing.T) []string {
	t.Helper()
	var ops []string
	for op := range contractOperations(t) {
		if op != "POST /auth/logout" && op != "POST /me/account/close" {
			ops = append(ops, op)
		}
	}
	for _, op := range capabilityInteractions(t) {
		ops = append(ops, "FHIR "+op)
	}
	slices.Sort(ops)
	return ops
}

// matrixRequest gives op's request on ids: its path and its body built from the contract.
func matrixRequest(t *testing.T, op string, ids map[string]string) (method, path, body string) {
	t.Helper()
	if fhirOp, ok := strings.CutPrefix(op, "FHIR "); ok {
		return fhirRequest(fhirOp, ids)
	}
	method, route, _ := strings.Cut(op, " ")
	return method, matrixPath(route, ids), requestBody(t, op, ids)
}

// answer is a caller's answer, with the ids it was asked about put back to unknownID, so that the answers on real and
// unknown ids compare.
func answer(c *client, method, path, body string, ids map[string]string) (int, string) {
	c.ifMatch = ""
	if method != http.MethodGet {
		c.ifMatch = `W/"1"` // past the precondition, to the access check
	}
	rec := c.do(method, path, body)
	out := rec.Body.String()
	if fhirEmpty(out) {
		out = "an empty searchset"
	}
	for _, id := range ids {
		out = strings.ReplaceAll(out, id, unknownID)
	}
	return rec.Code, out
}

// TestRoleMatrix_AnotherClinicsAndPatientsIDsAnswerAsUnknownOnes holds every route and FHIR interaction, called by
// each of a clinic's staff and its patient with the ids of another clinic's records (and for the patient, with their
// neighbour's at the same clinic), to the very answer an id that does not exist gets: a refusal reveals nothing, and
// none of the other's data.
func TestRoleMatrix_AnotherClinicsAndPatientsIDsAnswerAsUnknownOnes(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	type target struct {
		ids  map[string]string
		data []string
		own  bool // the public directory's ids are the caller's own clinic's
	}
	other := func(c *matrixClinic) target {
		return target{ids: c.ids, data: append(slices.Clone(c.patient.data), c.neighbour.data...)}
	}
	// B's records with A's directory: A's slot, service and role with B's patient or visit, as a caller of A would
	// name them in a write of their own clinic.
	mixed := func(of target) target { return target{ids: withPublic(of.ids, m.a.ids), data: of.data, own: true} }
	b := other(m.b)
	neighbour := mixed(target{ids: m.a.neighbour.ids, data: m.a.neighbour.data})
	callers := []struct {
		name string
		c    *client
		of   []target
	}{
		{"admin", m.a.admin, []target{b, mixed(b)}},
		{"practitioner", m.a.practitioner, []target{b, mixed(b)}},
		{"receptionist", m.a.receptionist, []target{b, mixed(b)}},
		{"patient", m.a.patient.c, []target{b, mixed(b), neighbour}},
	}
	for _, op := range matrixOps(t) {
		for _, caller := range callers {
			for _, of := range caller.of {
				checkOthers(t, op, caller.name, caller.c, of.ids, of.data, of.own)
			}
		}
	}
}

// withPublic gives ids with the public directory's ids taken from public.
func withPublic(ids, public map[string]string) map[string]string {
	out := maps.Clone(ids)
	for _, k := range publicKinds {
		out[k] = public[k]
	}
	return out
}

// publicKinds are the kinds of the public directory: the clinics, their practitioners and roles, schedules and free
// slots, their locations and their price lists, which anyone may read.
var publicKinds = []string{"org", "role", "practitioner", "schedule", "slot", "location", "service"}

// checkOthers holds who's answer on op with the records ids of another clinic or patient: none of their data, and
// the answer an id that does not exist gets; on the public directory alone, which anyone reads, no write by staff.
func checkOthers(t *testing.T, op, who string, c *client, ids map[string]string, data []string, own bool) {
	t.Helper()
	unknownIDs := map[string]string{}
	for _, k := range publicKinds {
		if id, ok := ids[k]; ok {
			unknownIDs[k] = id
		}
	}
	method, real, realBody := matrixRequest(t, op, ids)
	_, unknown, unknownBody := matrixRequest(t, op, unknownIDs)
	_, none, _ := matrixRequest(t, op, nil)
	if real == unknown && realBody == unknownBody && none == real {
		return // op names none of the other's records
	}
	if real == unknown && realBody == unknownBody && own {
		return // op names only the caller's own directory: the own-clinic test holds it
	}
	code, body := answer(c, method, real, realBody, ids)
	for _, s := range data {
		if strings.Contains(body, s) {
			t.Errorf("%s as A's %s on another's records: the answer holds their data", op, who)
		}
	}
	if real == unknown && realBody == unknownBody {
		// Staff do not write another clinic's directory: a write of their own clinic ignores a foreign clinic in its
		// body, and a patient's own route answers staff as an account without a patient profile.
		mine := who == "patient" || receptionRoutes[op] == stranger
		if !mine && method != http.MethodGet && code < 300 && (real != none || strings.Contains(body, ids["org"])) {
			t.Errorf("%s as A's %s on another clinic's directory = %d %.200s, want a refusal", op, who, code, body)
		}
		return
	}
	if uc, ub := answer(c, method, unknown, unknownBody, ids); uc != code || ub != body {
		t.Errorf("%s as A's %s = %d %.200s on another's records, %d %.200s on unknown ids: the answer reveals what exists",
			op, who, code, body, uc, ub)
	}
}

// TestRoleMatrix_AReceptionistBookingWithARealBodyIsAnAccountWithoutAPatientProfile holds F1's stranger class of POST
// /appointments to the answer with a real body, which the matrix's stranger check (no body) does not reach: the
// desk's account books nothing as a patient and gets what any account without a patient profile gets.
func TestRoleMatrix_AReceptionistBookingWithARealBodyIsAnAccountWithoutAPatientProfile(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	method, path, body := matrixRequest(t, "POST /appointments", m.a.ids)
	if !strings.Contains(body, m.a.ids["slot"]) {
		t.Fatalf("the booking's body %s names no real slot of the clinic", body)
	}
	before := m.count(`SELECT count(*) FROM appointment`)
	code, got := answer(m.a.receptionist, method, path, body, nil)
	if code != http.StatusConflict || got != `{"error":"profile_required"}`+"\n" {
		t.Errorf("POST /appointments with a real slot as the receptionist = %d %.200s, want 409 profile_required", code, got)
	}
	if sc, sg := answer(m.stranger, method, path, body, nil); sc != code || sg != got {
		t.Errorf("the stranger gets %d %.200s, the receptionist %d %.200s: want the same", sc, sg, code, got)
	}
	if after := m.count(`SELECT count(*) FROM appointment`); after != before {
		t.Errorf("appointments %d before, %d after: want none booked", before, after)
	}
}
