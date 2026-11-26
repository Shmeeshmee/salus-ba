package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The role matrix with a suspended clinic (docs/clinic-cms/PROGRESS.md, "Readiness follow-ups: suspended clinics"): out
// of the public directory and taking no new booking, while its own staff and its patients keep everything of theirs.

// servedReads are the public directory's reads by id that a clinic whose membership stopped keeps for those it serves,
// its staff and the patients with a care consent with it (organization.Service.Served), read in its own scope: where
// a visit is, the clinic's profile, a schedule. Each answers them as before, body and all.
var servedReads = map[string]bool{
	"FHIR read Schedule": true, "GET /organizations/{id}/locations": true, "GET /organizations/{id}/profile": true,
	"FHIR read Organization": true, "FHIR read PractitionerRole": true,
}

// clinicReads are the reads by id that name a visit's clinic to its patient (the clinic of a care consent, active or
// revoked), whatever its membership, and to its staff; to anyone else they answer as for an unknown id.
var clinicReads = []string{"FHIR read Organization", "FHIR read PractitionerRole"}

// A suspended clinic takes no new bookings, from the patient, the desk or a treatment plan, and nothing else changes for
// its own: every read answers each of its staff and its patient as it did before (the directory's searches show it to
// no one), and care goes on (docs/plans/clinic-cms.md Q4: care never stops).
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
			was := before[who][op]
			if was.code != got.code || servedReads[op] && was.body != got.body {
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
	// A revoked consent still names where the patient's visits were.
	if rec := m.a.neighbour.c.do(http.MethodPost, "/consents/"+m.a.neighbour.ids["consent"]+"/revoke", ""); rec.Code != http.StatusOK {
		t.Fatalf("POST /consents/{id}/revoke = %d %s", rec.Code, rec.Body)
	}
	for op := range servedReads {
		method, path, _ := matrixRequest(t, op, m.a.ids)
		if code, body := answer(m.a.neighbour.c, method, path, "", nil); code != before["patient"][op].code {
			t.Errorf("%s as a patient of the suspended clinic who revoked their consent = %d %.200s, want %d", op, code, body, before["patient"][op].code)
		}
	}
}

// directoryReads are the public directory's reads of a clinic: its listing, roles, schedules, free slots, locations,
// profile and the prices shown at booking.
var directoryReads = []string{
	"FHIR search-type Organization", "FHIR search-type PractitionerRole", "FHIR search-type Schedule", "FHIR read Schedule",
	"FHIR search-type Slot", "GET /organizations/{id}/locations", "GET /organizations/{id}/profile", "GET /organizations/{id}/services",
}

// directoryShows gives, by read, whether c's answer shows clinic c of the matrix.
func directoryShows(t *testing.T, c *client, clinic *matrixClinic) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, op := range directoryReads {
		method, path, _ := matrixRequest(t, op, clinic.ids)
		if op == "GET /organizations/{id}/services" {
			path += "?mode=in-person"
		}
		code, body := answer(c, method, path, "", nil)
		out[op] = code == http.StatusOK && body != "an empty searchset" && body != "[]" &&
			(op != "FHIR search-type Organization" || strings.Contains(body, clinic.org))
	}
	return out
}

// deskSearches are the directory's searches that its staff make in their own clinic's scope, listed or not (the front
// desk, docs/clinic-cms F3): a suspended membership does not change them.
var deskSearches = map[string]bool{"FHIR search-type PractitionerRole": true, "FHIR search-type Schedule": true, "FHIR search-type Slot": true}

// While its membership is suspended or cancelled a clinic is out of the public directory: no read of it shows it to a
// caller it does not serve (signed out, without a patient profile, another clinic's patient or staff), its listing and
// the prices shown at booking show it to no one, and its patients' searches do not show it; nobody books. Once the
// membership is active again, all is back.
func TestRoleMatrix_ASuspendedClinicIsOutOfTheDirectoryUntilItsMembershipReturns(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	others := map[string]*client{
		"a signed-out caller": m.out, "an account without a patient profile": m.stranger,
		"a patient of another clinic": m.b.patient.c, "another clinic's admin": m.b.admin,
	}
	check := func(who string, c *client, want func(op string) bool) {
		t.Helper()
		for op, shown := range directoryShows(t, c, m.a) {
			if shown != want(op) {
				t.Errorf("%s as %s: shows the clinic %v, want %v", op, who, shown, want(op))
			}
		}
	}
	everything := func(string) bool { return true }
	nothing := func(string) bool { return false }
	for who, c := range others {
		check(who, c, everything)
	}
	for _, status := range []string{"suspended", "cancelled"} {
		m.exec(t, `UPDATE clinic_subscription SET status = $2 WHERE organization_id = $1`, m.a.org, status)
		for who, c := range others {
			check(who+", "+status, c, nothing)
		}
		check("its patient, "+status, m.a.patient.c, func(op string) bool { return servedReads[op] })
		for who, c := range staffOf(m.a) {
			check("its "+who+", "+status, c, func(op string) bool { return servedReads[op] || deskSearches[op] })
		}
		m.namesTheClinicToItsOwnOnly(t, others, status)
	}
	m.exec(t, `UPDATE clinic_subscription SET status = 'active' WHERE organization_id = $1`, m.a.org)
	m.namesTheClinicToItsOwnOnly(t, others, "active again")
	for who, c := range others {
		check(who+", active again", c, everything)
	}
	for name, rec := range newBookings(t, m, m.a) {
		if rec.Code >= 400 {
			t.Errorf("%s once the membership is active again = %d %s, want it booked", name, rec.Code, rec.Body)
		}
	}
}

// namesTheClinicToItsOwnOnly holds the reads that name clinic A: they answer A's patient and staff with A's clinic and
// role, and every other caller as for an id that does not exist.
func (m *roleWorld) namesTheClinicToItsOwnOnly(t *testing.T, others map[string]*client, state string) {
	t.Helper()
	own := staffOf(m.a)
	own["patient"] = m.a.patient.c
	for _, op := range clinicReads {
		method, path, _ := matrixRequest(t, op, m.a.ids)
		_, unknown, _ := matrixRequest(t, op, map[string]string{})
		for who, c := range own {
			if code, body := answer(c, method, path, "", nil); code != http.StatusOK || !strings.Contains(body, m.a.org) {
				t.Errorf("%s as its %s, %s = %d %.200s; want the clinic's", op, who, state, code, body)
			}
		}
		for who, c := range others {
			code, body := answer(c, method, path, "", m.a.ids)
			if wantCode, want := answer(c, method, unknown, "", nil); code != wantCode || body != want {
				t.Errorf("%s as %s, %s = %d %.200s; want the unknown id's %d %.200s", op, who, state, code, body, wantCode, want)
			}
		}
	}
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
