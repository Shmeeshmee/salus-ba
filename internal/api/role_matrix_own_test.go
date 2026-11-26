package api_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

// matrixRoles are the columns of roleRoutes.
var matrixRoles = []string{"patient", "practitioner", "admin"}

func TestRoleMatrix_EveryRouteIsClassifiedForEveryRole(t *testing.T) {
	t.Parallel()
	ops := contractOperations(t)
	for op, open := range ops {
		row, ok := roleRoutes[op]
		if !ok {
			t.Errorf("%s is in api/openapi.yaml but not classified in roleRoutes", op)
			continue
		}
		for i, class := range row {
			if open != (class == public) {
				t.Errorf("%s: the %s's class is %d, but the contract says it takes a session: %v", op, matrixRoles[i], class, !open)
			}
		}
	}
	interactions := capabilityInteractions(t)
	for _, op := range interactions {
		if _, ok := roleRoutes["FHIR "+op]; !ok {
			t.Errorf("%s is in the CapabilityStatement but not classified in roleRoutes", op)
		}
	}
	for op := range roleRoutes {
		fhirOp, isFHIR := strings.CutPrefix(op, "FHIR ")
		if _, ok := ops[op]; !ok && (!isFHIR || !slices.Contains(interactions, fhirOp)) {
			t.Errorf("%s is classified in roleRoutes but is not in the contract", op)
		}
	}
}

// classOf is who's class on op: roleRoutes' column, or F1's tables for the receptionist.
func classOf(who, op string) receptionClass {
	if who == "receptionist" {
		if fhirOp, ok := strings.CutPrefix(op, "FHIR "); ok {
			return receptionFHIR[fhirOp]
		}
		return receptionRoutes[op]
	}
	return roleRoutes[op][slices.Index(matrixRoles, who)]
}

// Each of the patient, the practitioner, the admin and the receptionist reaches what its class allows on its own
// clinic's records and is refused the rest, as a stranger: 403 or 404, the very answer an id that does not exist
// gets, and none of a patient's data. Each role has a world of its own, as the routes change what they reach.
func TestRoleMatrix_EachRoleOnItsOwnClinic(t *testing.T) {
	t.Parallel()
	for _, who := range append(slices.Clone(matrixRoles), "receptionist") {
		t.Run(who, func(t *testing.T) {
			t.Parallel()
			m := newRoleWorld(t)
			c := map[string]*client{"patient": m.a.patient.c, "practitioner": m.a.practitioner, "admin": m.a.admin,
				"receptionist": m.a.receptionist}[who]
			var data []string
			for _, cl := range []*matrixClinic{m.a, m.b} {
				data = append(data, cl.patient.data...)
				data = append(data, cl.neighbour.data...)
			}
			for _, op := range matrixOps(t) {
				checkOwn(t, m, who, c, op, data)
			}
		})
	}
}

// checkOwn holds who's answer on op with their own clinic's records to op's class.
func checkOwn(t *testing.T, m *roleWorld, who string, c *client, op string, data []string) {
	t.Helper()
	class := classOf(who, op)
	method, path, body := matrixRequest(t, op, m.a.ids)
	switch class {
	case public:
		return // TestRoleMatrix_ASignedOutCallerReachesOnlyThePublicRoutes
	case stranger: // as an account without a patient profile, on the requests without a body, as F1
		code, got := answer(c, method, path, "", nil)
		if code == http.StatusForbidden || code == http.StatusNotFound {
			checkOwnRefused(t, m, who, c, op, code, got, "", data)
		} else if wc, want := answer(m.stranger, method, path, "", nil); wc != code || want != got {
			t.Errorf("%s as the %s = %d %.200s, want what a stranger gets: %d %.200s", op, who, code, got, wc, want)
		}
		return
	}
	code, got := answer(c, method, path, body, nil)
	switch class {
	case own, allowed:
		if code == http.StatusUnauthorized || code == http.StatusForbidden || code >= http.StatusInternalServerError {
			t.Errorf("%s as the %s = %d %.200s, want the route's answer", op, who, code, got)
		}
	case refused:
		checkOwnRefused(t, m, who, c, op, code, got, body, data)
	case machine:
		if code != http.StatusUnauthorized {
			t.Errorf("%s as the %s = %d %.200s, want 401", op, who, code, got)
		}
	default:
		t.Errorf("%s: no class for the %s", op, who)
	}
}

// checkOwnRefused: a refusal is 403 or 404 (an empty FHIR search, or a request rejected as malformed before any
// access, which reveals nothing either) without a patient's data; for the receptionist, as F1, the same on the
// clinic's own records as on ids that do not exist.
func checkOwnRefused(t *testing.T, m *roleWorld, who string, c *client, op string, code int, got, body string, data []string) {
	t.Helper()
	// Only the two answers given before any access, as F1's checkRefused: a 400 of a validation after the access check
	// (invalid_service, say) is the route's answer, not a refusal.
	malformed := code == http.StatusBadRequest &&
		(got == `{"error":"invalid_request"}`+"\n" || got == `{"error":"websocket_required"}`+"\n")
	if code != http.StatusForbidden && code != http.StatusNotFound && got != "an empty searchset" && !malformed {
		t.Errorf("%s as the %s = %d %.200s, want a refusal", op, who, code, got)
	}
	for _, s := range data {
		if strings.Contains(got, s) {
			t.Errorf("%s as the %s: the refusal holds a patient's data", op, who)
		}
	}
	if who != "receptionist" {
		// Within its own clinic a refusal may say forbidden where an unknown id is not found: the opaque id is the
		// clinic's own. Another clinic's ids are held to the unknown answer by
		// TestRoleMatrix_AnotherClinicsAndPatientsIDsAnswerAsUnknownOnes, the desk's as F1 holds them.
		return
	}
	unknownIDs := withPublic(map[string]string{}, m.a.ids)
	method, path, realBody := matrixRequest(t, op, m.a.ids)
	_, unknown, unknownBody := matrixRequest(t, op, unknownIDs)
	if body == "" {
		unknownBody = ""
	}
	if unknown == path && unknownBody == realBody {
		return
	}
	if uc, ub := answer(c, method, unknown, unknownBody, m.a.ids); uc != code || !sameRefusal(ub, got, m.a.ids) {
		t.Errorf("%s as the %s = %d %.200s on the clinic's own, %d %.200s on unknown ids: the refusal reveals what exists",
			op, who, code, got, uc, ub)
	}
}

// sameRefusal compares two refusals with the clinic's ids taken out of the first.
func sameRefusal(unknown, got string, ids map[string]string) bool {
	for _, id := range ids {
		got = strings.ReplaceAll(got, id, unknownID)
	}
	return unknown == got
}

// first401 is the answer to a request without a session on a route that needs one.
const first401 = `{"error":"unauthenticated"}` + "\n"

// publicFHIR are the FHIR interactions of the public directory, for anyone, signed in or not (docs/booking/SPEC.md,
// "Public browse").
var publicFHIR = map[string]bool{
	"FHIR search-type Organization": true, "FHIR search-type PractitionerRole": true, "FHIR read Schedule": true,
	"FHIR search-type Schedule": true, "FHIR search-type Slot": true,
}

// A signed-out caller reaches the public routes and gets the one 401 on every other route and FHIR interaction, the
// same for a real id as for an unknown one.
func TestRoleMatrix_ASignedOutCallerReachesOnlyThePublicRoutes(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	ops := contractOperations(t)
	for _, op := range matrixOps(t) {
		method, path, body := matrixRequest(t, op, m.b.ids)
		code, got := answer(m.out, method, path, body, nil)
		if open, isRoute := ops[op]; isRoute && open || publicFHIR[op] {
			if got == first401 || code >= http.StatusInternalServerError {
				t.Errorf("%s signed out = %d %.200s, want the public answer", op, code, got)
			}
			continue
		}
		// A live stream refuses a plain request before it looks at the session, which reveals nothing either.
		stream := code == http.StatusBadRequest && got == `{"error":"websocket_required"}`+"\n"
		if !stream && (code != http.StatusUnauthorized || got != first401 && !strings.HasPrefix(op, "FHIR ")) {
			t.Errorf("%s signed out = %d %.200s, want 401 %.200s", op, code, got, first401)
		}
		for _, s := range append(slices.Clone(m.b.patient.data), m.b.neighbour.data...) {
			if strings.Contains(got, s) {
				t.Errorf("%s signed out: the answer holds a patient's data", op)
			}
		}
	}
}
