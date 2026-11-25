package api_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

// restrictedOpenings are the reads that open a restricted record on purpose (docs/clinic-cms/WAVE5.md S7): the admin's
// hand-over of the record, read with its purpose and audited; and the admin's queue and detail of the clinic's data
// requests, which name the request's patient (its name and birth date only, each read audited as a read of the Patient
// at the patient's request): the request is the stated reason, and the admin reaches a restricted patient only through
// it (wave 5 walk, findings 3 and 5, the owner, 2026-11-25). Nobody else, and no other route, returns its data.
var restrictedOpenings = map[string]bool{
	"admin GET /clinic/patients/{id}/record": true,
	"admin GET /clinic/data-requests":        true,
	"admin GET /clinic/data-requests/{id}":   true,
}

// restrictedViewers are the clinic's staff a restricted record is hidden from.
func restrictedViewers(r reception) map[string]*client {
	return map[string]*client{"practitioner": r.practitioner, "admin": r.admin, "receptionist": r.receptionist}
}

// restrictedReads reads every GET route of the contract and every FHIR read and search as each of the clinic's staff
// and returns the bodies, keyed "viewer GET route".
func restrictedReads(t *testing.T, r reception) map[string]string {
	t.Helper()
	var day string // the visit's day in the clinic's zone, for the desk
	if err := r.owner.QueryRow(t.Context(), `SELECT to_char(lower(period) AT TIME ZONE 'Europe/Sarajevo', 'YYYY-MM-DD') FROM appointment WHERE id = $1`, r.visit).Scan(&day); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	var routes []string
	for op, class := range receptionRoutes {
		if strings.HasPrefix(op, "GET ") && class != public && class != machine {
			routes = append(routes, op)
		}
	}
	slices.Sort(routes)
	ids := map[string]string{"Patient": r.patientID, "Appointment": r.visit, "Organization": inviteClinic,
		"Practitioner": r.practitionerID, "PractitionerRole": r.roleID}
	for name, c := range restrictedViewers(r) {
		for _, op := range routes {
			_, route, _ := strings.Cut(op, " ")
			path := strings.Replace(receptionPath(r, route, true), "/clinic/patients/"+r.registered, "/clinic/patients/"+r.patientID, 1)
			switch route {
			case "/clinic/patients":
				path += "?name=Syntheticova" // the desk's search
			case "/clinic/desk/today":
				path += "?date=" + day // the desk's day of the visit
			}
			out[name+" "+op] = c.do(http.MethodGet, path, "").Body.String()
		}
		for _, op := range capabilityInteractions(t) {
			if method, path := fhirPath(op, ids, true); method == http.MethodGet {
				out[name+" FHIR "+op] = c.do(method, path, "").Body.String()
			}
		}
	}
	return out
}

// A restricted record is hidden from the clinic's everyday screens and searches (S7): route by route, as in F1's test,
// a restricted patient's data reaches neither the practitioner of their visit, nor the admin, nor the desk, where it
// did before the restriction; the patient still reads their own; lifting the restriction gives it back.
func TestRestrictedRoutes_ARestrictedPatientsDataReachesNoStaff(t *testing.T) {
	t.Parallel()
	r, count := newReception(t)
	leaked := shown(restrictedReads(t, r))
	if len(leaked) < 3 {
		t.Fatalf("only %v hold the patient's data before the restriction: the fixture shows nothing to restrict", leaked)
	}
	t.Logf("the patient's data before the restriction: %v", leaked)
	// A request that is not the clinic's and the patient's is refused as not found, audited, and restricts nothing.
	if rec := r.admin.do(http.MethodPut, "/clinic/patients/"+r.patientID+"/restriction", `{"dataRequestId":"`+unknownID+`"}`); rec.Code != http.StatusNotFound {
		t.Errorf("PUT restriction for an unknown request = %d %s, want 404", rec.Code, rec.Body)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_account_id IS NOT NULL AND subtype_code = 'update' AND outcome = 4 AND entity_reference IS NULL`); n == 0 {
		t.Error("the refused restriction was not audited")
	}
	restrict := r.admin.do(http.MethodPut, "/clinic/patients/"+r.patientID+"/restriction", "")
	if restrict.Code != http.StatusNoContent {
		t.Fatalf("PUT restriction as the admin = %d %s", restrict.Code, restrict.Body)
	}
	for key, body := range restrictedReads(t, r) {
		if restrictedOpenings[key] {
			continue
		}
		for _, s := range bookedPatientData {
			if strings.Contains(body, s) {
				t.Errorf("%s returns the restricted patient's data", key)
				break
			}
		}
	}
	if rec := r.patient.do(http.MethodGet, "/me/patient", ""); !strings.Contains(rec.Body.String(), "Syntheticova") {
		t.Errorf("GET /me/patient as the restricted patient = %d %s, want their own profile", rec.Code, rec.Body)
	}
	for _, c := range []*client{r.practitioner, r.receptionist} {
		if rec := c.do(http.MethodPut, "/clinic/patients/"+r.patientID+"/restriction", ""); rec.Code != http.StatusForbidden {
			t.Errorf("PUT restriction as staff who are not the admin = %d %s, want 403", rec.Code, rec.Body)
		}
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE entity_reference = $1 AND subtype_code = 'update' AND outcome = 0`, "Patient/"+r.patientID); n != 1 {
		t.Errorf("%d audited restrictions, want 1", n)
	}
	if rec := r.admin.do(http.MethodDelete, "/clinic/patients/"+r.patientID+"/restriction", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE restriction as the admin = %d %s", rec.Code, rec.Body)
	}
	if again := shown(restrictedReads(t, r)); !slices.Equal(again, leaked) {
		t.Errorf("after lifting, the patient's data is shown on %v, want %v as before", again, leaked)
	}
}

// shown returns, in order, the reads whose body holds the patient's family name.
func shown(reads map[string]string) []string {
	var out []string
	for key, body := range reads {
		if strings.Contains(body, "Syntheticova") {
			out = append(out, key)
		}
	}
	slices.Sort(out)
	return out
}
