package api_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The role matrix in the states that change what a role reaches (docs/clinic-cms/WAVE7.md RD1): a suspended clinic,
// a restricted record, a closed account and a revoked consent.

// staffOrder are the staff roles in the order the state tests take them.
var staffOrder = []string{"practitioner", "receptionist", "admin"}

// staffOf are a clinic's staff by role.
func staffOf(c *matrixClinic) map[string]*client {
	return map[string]*client{"practitioner": c.practitioner, "admin": c.admin, "receptionist": c.receptionist}
}

type reply struct {
	body string
	code int
}

// readsOf gives c's answer on every read of the contract and of FHIR on ids, by op.
func readsOf(t *testing.T, c *client, ids map[string]string) map[string]reply {
	t.Helper()
	out := map[string]reply{}
	for _, op := range matrixOps(t) {
		method, path, _ := matrixRequest(t, op, ids)
		if method != http.MethodGet {
			continue
		}
		code, body := answer(c, method, path, "", nil)
		out[op] = reply{body: body, code: code}
	}
	return out
}

// directoryOnly are the reads a suspended membership changes on purpose: the public directory stops listing the
// clinic (docs/clinic-cms I), and its public reads answer as for a clinic that is not listed, to everyone.
var directoryOnly = map[string]bool{
	"FHIR search-type Organization": true, "FHIR read Schedule": true, "GET /organizations/{id}/locations": true,
	"GET /organizations/{id}/profile": true,
}

// A suspended clinic takes no new bookings, from the patient, the desk or a treatment plan, and nothing else changes:
// every read answers each role as it did before, and care goes on (docs/plans/clinic-cms.md Q4: care never stops).
func TestRoleMatrix_ASuspendedClinicTakesNoNewBookingsAndCareGoesOn(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	callers := staffOf(m.a)
	callers["patient"] = m.a.patient.c
	before := map[string]map[string]reply{}
	for who, c := range callers {
		before[who] = readsOf(t, c, m.a.ids)
	}
	m.exec(t, `UPDATE clinic_subscription SET status = 'suspended' WHERE organization_id = $1`, m.a.org)
	for who, c := range callers {
		for op, got := range readsOf(t, c, m.a.ids) {
			if was := before[who][op]; was.code != got.code && !directoryOnly[op] {
				t.Errorf("%s as the %s of a suspended clinic = %d %.200s, was %d %.200s", op, who, got.code, got.body, was.code, was.body)
			}
		}
	}
	for _, c := range []*matrixClinic{m.b, m.a} { // B, whose membership is in its trial, books: the same requests
		for name, rec := range newBookings(t, m, c) {
			if refused := rec.Code >= 400; refused != (c == m.a) {
				t.Errorf("%s at the clinic %s = %d %s, refused %v", name, c.tag, rec.Code, rec.Body, c == m.a)
			}
		}
	}
	// Care goes on: the visit's practitioner records a lab result, the desk issues a receipt.
	created(t, m.a.practitioner, http.StatusCreated, http.MethodPost, "/clinic/appointments/"+m.a.neighbour.ids["visit"]+"/diagnostic-reports",
		`{"title":"Placeholder blood count","results":[]}`)
	created(t, m.a.receptionist, http.StatusCreated, http.MethodPost, "/clinic/receipts",
		`{"kind":"receipt","appointmentId":"`+m.a.neighbour.ids["visit"]+`","paymentMethod":"cash","lines":[{"serviceId":"`+m.a.ids["service"]+`","quantity":1}]}`)
}

// newBookings makes a booking of each kind at clinic c for its neighbour patient: theirs in the app, the desk's, a
// waitlist entry and a treatment plan's; it gives the answers by kind.
func newBookings(t *testing.T, m *roleWorld, c *matrixClinic) map[string]*httptest.ResponseRecorder {
	t.Helper()
	p := c.neighbour
	slots := []string{m.slot(t, c, "120 hours"), m.slot(t, c, "121 hours"), m.slot(t, c, "122 hours")}
	plan := created(t, c.practitioner, http.StatusCreated, http.MethodPost, "/clinic/care-plans",
		`{"patientId":"`+p.ids["patient"]+`","title":"Synthetic plan","steps":[{"serviceId":"`+c.ids["service"]+`","dayOffset":0,"visitMode":"in-person"}]}`)
	startsOn := m.id(t, `SELECT to_char(lower(period) AT TIME ZONE 'Europe/Sarajevo', 'YYYY-MM-DD') FROM slot WHERE id = $1`, slots[2])
	out := map[string]*httptest.ResponseRecorder{}
	p.c.do(http.MethodDelete, "/waitlist/"+p.ids["waitlist"], "") // the patient's entry, so that they may join again
	out["the patient's booking"] = p.c.do(http.MethodPost, "/appointments", `{"slotId":"`+slots[0]+`"}`)
	out["the desk's booking"] = c.receptionist.do(http.MethodPost, "/clinic/appointments", `{"patientId":"`+p.ids["patient"]+`","slotId":"`+slots[1]+`"}`)
	out["a waitlist entry"] = p.c.do(http.MethodPost, "/waitlist",
		`{"organization_id":"`+c.org+`","practitioner_role_id":"`+c.ids["role"]+`","earliest":"`+day(5)+`","latest":"`+day(20)+`"}`)
	c.practitioner.ifMatch = `W/"1"`
	out["a treatment plan's booking"] = c.practitioner.do(http.MethodPost, "/clinic/care-plans/"+plan+"/book",
		`{"startsOn":"`+startsOn+`","slotIds":["`+slots[2]+`"]}`)
	c.practitioner.ifMatch = ""
	return out
}

// staffWrites sends every write of the contract and of FHIR that names a patient's record, as c on ids, and gives the
// answers that succeed, by op. A conflict (409, 412) is not counted: the state of the fixture's record, which the
// route's own tests hold.
func staffWrites(t *testing.T, c *client, ids map[string]string) map[string]reply {
	t.Helper()
	out := map[string]reply{}
	for _, op := range matrixOps(t) {
		method, path, body := matrixRequest(t, op, ids)
		_, none, noneBody := matrixRequest(t, op, withoutPatient(ids))
		if method == http.MethodGet || path == none && body == noneBody || op == "DELETE /clinic/patients/{id}/restriction" {
			continue // a read, a write that names no patient's record, or the lifting of the state under test
		}
		if code, got := answer(c, method, path, body, nil); code < 300 {
			out[op] = reply{body: got, code: code}
		}
	}
	return out
}

// patientKinds are the kinds of records that name a patient.
var patientKinds = []string{"patient", "visit", "receipt", "quote", "report", "plan", "response", "conversation",
	"waitlist", "encounter", "allergy", "medication", "request", "consent", "binary", "message", "attachment"}

// withoutPatient gives ids without the records that name a patient.
func withoutPatient(ids map[string]string) map[string]string {
	out := maps.Clone(ids)
	for _, k := range patientKinds {
		delete(out, k)
	}
	return out
}

// shows reports the reads in reads whose answer holds any of data.
func shows(reads map[string]reply, data []string) []string {
	var out []string
	for op, r := range reads {
		for _, s := range data {
			if strings.Contains(r.body, s) {
				out = append(out, op)
				break
			}
		}
	}
	return out
}

// A restricted record (docs/clinic-cms/WAVE5.md S7) is hidden from the clinic's staff on every read but the admin's
// openings on purpose, and no write of theirs reaches it.
func TestRoleMatrix_ARestrictedRecordReachesNoStaffAndTakesNoWrite(t *testing.T) {
	t.Parallel()
	for _, who := range staffOrder {
		t.Run(who, func(t *testing.T) {
			t.Parallel()
			m := newRoleWorld(t)
			if rec := m.a.admin.do(http.MethodPut, "/clinic/patients/"+m.a.ids["patient"]+"/restriction", ""); rec.Code != http.StatusNoContent {
				t.Fatalf("PUT restriction as the admin = %d %s", rec.Code, rec.Body)
			}
			c := staffOf(m.a)[who]
			for _, op := range shows(readsOf(t, c, m.a.ids), m.a.patient.data) {
				if !restrictedOpenings[who+" "+op] {
					t.Errorf("%s as the %s holds the restricted patient's data", op, who)
				}
			}
			for op, r := range staffWrites(t, c, m.a.ids) {
				if !restrictedWrites[op] {
					t.Errorf("%s as the %s on a restricted record = %d %.200s, want a refusal", op, who, r.code, r.body)
				}
			}
		})
	}
}

// restrictedWrites are the writes a restricted record still takes (S7): those that end or lessen its use (a
// cancellation, a withdrawal from the waitlist), the clinic's handling of the patient's request and of the
// restriction itself, and the receipts the fiscal law asks for a visit (D4's open point, for the owner).
var restrictedWrites = map[string]bool{
	"POST /appointments/{id}/cancel": true, "POST /clinic/appointments/{id}/cancel": true, "DELETE /waitlist/{id}": true,
	"POST /clinic/care-plans/{id}/cancel": true, "POST /clinic/data-requests/{id}/answer": true,
	"POST /clinic/data-requests/{id}/extend": true, "PUT /clinic/patients/{id}/restriction": true,
	"POST /clinic/receipts": true, "POST /clinic/receipts/{id}/reverse": true, "POST /clinic/receipts/{id}/retry": true,
}

// revokedWrites are the writes a revoked consent still allows the clinic (docs/booking/SPEC.md: the past stays the
// clinic's record): those that end or lessen a use, the clinic's handling of the patient's request, a restriction,
// and a past receipt's reversal or retried report.
var revokedWrites = map[string]bool{
	"POST /clinic/care-plans/{id}/cancel": true, "POST /clinic/data-requests/{id}/answer": true,
	"POST /clinic/data-requests/{id}/extend": true, "PUT /clinic/patients/{id}/restriction": true,
	"POST /clinic/receipts/{id}/reverse": true, "POST /clinic/receipts/{id}/retry": true,
}

// A revoked consent (docs/booking/SPEC.md "Past and future after revocation") ends the clinic's access to the
// patient's profile at once: no read of its staff gives their birth date or phone.
func TestRoleMatrix_ARevokedConsentEndsTheClinicsAccessToTheProfile(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	if rec := m.a.patient.c.do(http.MethodPost, "/consents/"+m.a.ids["consent"]+"/revoke", ""); rec.Code != http.StatusOK {
		t.Fatalf("POST /consents/{id}/revoke = %d %s", rec.Code, rec.Body)
	}
	for _, who := range staffOrder { // the admin's writes last: a restriction of theirs only refuses more
		c := staffOf(m.a)[who]
		for _, op := range shows(readsOf(t, c, m.a.ids), m.a.patient.data[1:]) {
			if !restrictedOpenings[who+" "+op] {
				t.Errorf("%s as the %s holds the profile of a patient who revoked their consent", op, who)
			}
		}
		for op, r := range staffWrites(t, c, m.a.ids) {
			if !revokedWrites[op] {
				t.Errorf("%s as the %s after the revocation = %d %.200s, want a refusal", op, who, r.code, r.body)
			}
		}
	}
}

// A closed account (docs/clinic-cms/WAVE5.md S5) signs out at once: its session gets the one 401 on every route
// that takes one, it signs in no more, and its clinics lose its profile as on a revocation.
func TestRoleMatrix_AClosedAccountIsSignedOutEverywhere(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	p := m.a.patient.c
	if rec := p.do(http.MethodPost, "/me/account/close", `{"password":"`+testPassword+`"}`); rec.Code >= 300 {
		t.Fatalf("POST /me/account/close = %d %s", rec.Code, rec.Body)
	}
	ops := contractOperations(t)
	for _, op := range matrixOps(t) {
		if open := ops[op]; open || publicFHIR[op] {
			continue
		}
		method, path, body := matrixRequest(t, op, m.a.ids)
		code, got := answer(p, method, path, body, nil)
		stream := code == http.StatusBadRequest && strings.Contains(got, `"websocket_required"`)
		if !stream && (code != http.StatusUnauthorized || got != first401 && !strings.HasPrefix(op, "FHIR ")) {
			t.Errorf("%s with a closed account's session = %d %.200s, want 401", op, code, got)
		}
	}
	login := `{"email":"alfa.patient@example.test","password":"` + testPassword + `"}`
	if rec := m.out.do(http.MethodPost, "/auth/login", login); rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /auth/login of a closed account = %d %s, want 401", rec.Code, rec.Body)
	}
	for _, who := range staffOrder {
		for _, op := range shows(readsOf(t, staffOf(m.a)[who], m.a.ids), m.a.patient.data[1:]) {
			if !restrictedOpenings[who+" "+op] {
				t.Errorf("%s as the %s holds the profile of a patient who closed their account", op, who)
			}
		}
	}
}
