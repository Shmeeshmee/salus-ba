package api_test

import (
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// What a receptionist gets on a route of api/openapi.yaml (docs/clinic-cms, F1; W2: the front desk sees a patient's
// name, birth date and phone, the appointments with their service and visit type, the price list and the receipts,
// and never questionnaire answers, documents, history, transcripts, chat or analytics).
type receptionClass int

const (
	// public: the route takes no session (security: [] in the contract); a receptionist is anyone.
	public receptionClass = iota + 1
	// own: the account's own session and sign-in, and what every signed-in account reads (the explanations of
	// medical terms).
	own
	// desk: the front desk's; a receptionist is not refused.
	desk
	// refused: a receptionist is refused as a stranger: 403 or 404, the same answer for the clinic's own patient or
	// appointment as for an id that does not exist, and none of the patient's data.
	refused
	// stranger: a patient's own routes; a receptionist is refused (as refused) or gets the very answer a signed-in
	// account without a patient profile gets.
	stranger
	// machine: a machine's credential, not a session (the video provider's signed webhook): a receptionist's session
	// is unauthenticated.
	machine
)

// receptionRoutes classifies every operation of api/openapi.yaml for the receptionist. A route added to the contract
// fails TestReceptionRoutes_EveryRouteOfTheContractIsClassified until it is classified here.
var receptionRoutes = map[string]receptionClass{
	"GET /healthz": public, "GET /readyz": public, "POST /auth/register": public, "POST /auth/login": public,
	"POST /auth/mfa/verify": public, "POST /invites/accept": public, "GET /specialties": public,
	"GET /a/{token}": public, "POST /a/{token}": public, "GET /organizations/{id}/services": public,
	"POST /signup": public, "POST /signup/verify": public, "GET /organizations/{id}/profile": public,

	"GET /auth/me": own, "PATCH /auth/me": own, "POST /auth/mfa/enrol": own, "POST /auth/mfa/confirm": own,
	"POST /auth/logout": own,

	// The price list (read), and the desk's routes of package F2.
	"GET /clinic/services": desk,
	"GET /clinic/patients": desk, "POST /clinic/patients": desk, "POST /clinic/appointments": desk,
	// R2: the desk corrects a profile the clinic registered and records the withdrawal of its consent; the record is the
	// admin's to hand over.
	"PUT /clinic/patients/{id}": desk, "POST /clinic/patients/{id}/consent": desk, "POST /clinic/patients/{id}/consent-withdrawal": desk, "GET /clinic/patients/{id}/record": refused,
	"POST /clinic/appointments/{id}/reschedule": desk, "POST /clinic/appointments/{id}/cancel": desk,
	"POST /clinic/appointments/{id}/arrive": desk, "GET /clinic/desk/today": desk,

	// Staff administration and the clinic's settings: the admin's.
	"POST /invites": refused, "GET /invites": refused,
	"PUT /practitioner-roles/{id}/specialty": refused, "GET /practitioner-roles/{id}/availability": refused,
	"PUT /practitioner-roles/{id}/availability": refused,
	"GET /clinic/chat-settings":                 refused, "PUT /clinic/chat-settings": refused,
	"GET /clinic/webhooks": refused, "POST /clinic/webhooks": refused, "DELETE /clinic/webhooks/{id}": refused,
	"GET /clinic/appointment-settings": refused, "PUT /clinic/appointment-settings": refused,
	"GET /clinic/sms-usage": refused, "GET /clinic/waitlist": refused, "PUT /clinic/waitlist/{id}/priority": refused,
	"GET /clinic/noshow-flags": refused, "POST /clinic/services": refused, "PUT /clinic/services/{id}": refused,
	"GET /clinic/subscription": refused, "GET /clinic/subscription/invoices": refused,
	"GET /clinic/subscription/invoices/{id}/pdf": refused, "GET /clinic/billing-details": refused,
	"PUT /clinic/billing-details": refused, "GET /clinic/setup": refused, "PUT /clinic/setup": refused,
	"POST /clinic/services/import": refused, "POST /clinic/export": refused, "GET /clinic/export/{id}": refused,
	"GET /clinic/export/{id}/archive": refused, "PUT /clinic/profile": refused,
	"POST /clinic/patients/import": refused,
	// The clinic's profile for patients abroad: public data (GET /organizations/{id}/profile), read by any staff (X2).
	"GET /clinic/profile": desk,

	// Receipts (W2): the desk issues, reads and prints the receipts it issued; the list of all of them, a reversal, a
	// retried report and the reports stay the admin's.
	"POST /clinic/receipts": desk, "GET /clinic/receipts/{id}": desk, "GET /clinic/receipts/{id}/pdf": desk,
	"GET /clinic/receipts": refused, "POST /clinic/receipts/{id}/reverse": refused,
	"POST /clinic/receipts/{id}/retry": refused, "GET /clinic/reports/receipts": refused,
	"GET /clinic/reports/practice": refused,

	// Quotes: a practitioner's treatment plan.
	"GET /clinic/quotes": refused, "POST /clinic/quotes": refused, "GET /clinic/quotes/{id}": refused,
	"GET /clinic/quotes/{id}/pdf": refused,

	// Questionnaires and their answers.
	"POST /questionnaires": refused, "PUT /questionnaires/{id}": refused, "DELETE /questionnaires/{id}": refused,
	"POST /questionnaires/{id}/publish": refused, "POST /questionnaires/{id}/retire": refused,
	"POST /questionnaires/{id}/revise": refused, "GET /practitioner-roles/{id}/questionnaire": refused,
	"PUT /practitioner-roles/{id}/questionnaire": refused, "DELETE /practitioner-roles/{id}/questionnaire": refused,
	"POST /appointments/{id}/questionnaires": refused, "POST /questionnaire-responses/{id}/start": refused,
	"PUT /questionnaire-responses/{id}/answers": refused, "POST /questionnaire-responses/{id}/submit": refused,
	"POST /questionnaire-responses/{id}/withdraw":                   refused,
	"POST /questionnaire-responses/{id}/items/{linkId}/attachments": refused,

	// The patient's appointments through the patient's and the practitioner's routes, and the consultation.
	"POST /appointments": refused, "POST /appointments/{id}/cancel": refused, "POST /appointments/{id}/status": refused,
	"GET /appointments/{id}/changes": refused, "POST /appointments/{id}/confirm": refused,
	"POST /appointments/{id}/reschedule": refused, "GET /appointments/{id}/actions": refused,
	"GET /appointments/{id}/consultation": refused, "POST /appointments/{id}/consultation/join": refused,
	"POST /appointments/{id}/consultation/end": refused, "GET /encounters/{id}/events": refused,
	"GET /encounters/{id}/transcript": refused, "POST /webhooks/livekit": machine,

	// Documents, files and the record.
	"GET /document-templates": refused, "POST /document-templates": refused, "PUT /document-templates/{id}": refused,
	"POST /encounters/{id}/drafts": refused, "GET /encounters/{id}/drafts": refused, "POST /drafts/{id}/sign": refused,
	"GET /document-references/{id}/content": refused, "GET /compositions/{id}/terms": refused,
	"GET /binaries/{id}": refused, "DELETE /binaries/{id}": refused, "GET /binaries/{id}/content": refused,
	"GET /patients/{id}/history": refused, "GET /patients/{id}/history/summary": refused,
	"POST /encounters/{id}/allergies": refused, "PUT /allergies/{id}": refused, "DELETE /allergies/{id}": refused,
	"POST /allergies/{id}/confirm": refused, "POST /allergies/{id}/profile": refused,
	"POST /encounters/{id}/medications": refused, "PUT /medications/{id}": refused, "DELETE /medications/{id}": refused,
	"POST /medications/{id}/confirm": refused, "POST /medications/{id}/profile": refused,

	// The term list (the practitioners' cheatsheet and its review) and analytics.
	"GET /terminology/search": refused, "GET /terminology/explanations": own,
	"GET /terminology/diagnosis-groups": refused, "GET /places": refused,
	"GET /terminology/review/diagnosis-groups": refused, "PUT /terminology/diagnosis-groups/{code}/names/{language}": refused,
	"POST /terminology/diagnosis-groups/{code}/names/{language}/review": refused, "GET /terminology/review": refused,
	"GET /terminology/review/progress": refused, "PUT /terminology/terms/{id}/translations/{language}": refused,
	"POST /terminology/terms/{id}/translations/{language}/review": refused,
	"PUT /terminology/terms/{id}/explanations/{language}":         refused,
	"POST /terminology/terms/{id}/explanations/{language}/review": refused,
	"GET /analytics/platform/months":                              refused, "GET /analytics/platform/cells": refused,
	"GET /analytics/platform/heatmap": refused, "GET /analytics/platform/rises": refused,
	"GET /analytics/clinic/appointments": refused, "GET /analytics/clinic/diagnoses": refused,

	// Chat.
	"POST /conversations": refused, "GET /conversations": refused, "GET /conversations/{id}": refused,
	"GET /conversations/{id}/messages": refused, "POST /conversations/{id}/messages": refused,
	"PATCH /conversations/{id}/messages/{message_id}":         refused,
	"GET /conversations/{id}/messages/{message_id}/revisions": refused, "PUT /conversations/{id}/read": refused,
	"POST /conversations/{id}/attachments":                          refused,
	"GET /conversations/{id}/attachments/{attachment_id}/content":   refused,
	"GET /conversations/{id}/attachments/{attachment_id}/thumbnail": refused,
	"GET /conversations/{id}/search":                                refused, "GET /conversations/search": refused,
	"POST /conversations/{id}/summary": refused, "GET /conversations/{id}/events": refused,
	"POST /conversations/{id}/participants": refused, "DELETE /conversations/{id}/participants/{participant_id}": refused,
	"POST /conversations/{id}/threads": refused, "GET /conversations/{id}/threads": refused,
	"GET /organizations/{id}/chat-hours": refused,

	// A patient's own routes: a receptionist has no patient profile, and gets what an account without one gets.
	"GET /me/patient": stranger, "PUT /me/patient": stranger, "POST /consents": stranger,
	"POST /consents/{id}/revoke": stranger, "GET /me/consents": stranger, "GET /me/history": stranger,
	"POST /me/allergies": stranger, "POST /me/medications": stranger, "GET /me/history-sharing": stranger,
	"GET /history-sharing/{organizationId}": stranger, "PUT /history-sharing/{organizationId}": stranger,
	"DELETE /history-sharing/{organizationId}": stranger, "GET /me/chat-invitations": stranger,
	"POST /me/chat-invitations/{id}/decision": stranger, "POST /me/chat-invitations/{id}/revoke": stranger,
	"PUT /me/devices": stranger, "DELETE /me/devices": stranger, "GET /me/notification-settings": stranger,
	"PUT /me/notification-settings": stranger, "POST /waitlist": stranger, "GET /me/waitlist": stranger,
	"DELETE /waitlist/{id}": stranger, "POST /waitlist/offers/{id}/accept": stranger,
	"POST /waitlist/offers/{id}/decline": stranger, "GET /me/receipts": stranger, "GET /me/receipts/{id}/pdf": stranger,
	"GET /me/quotes": stranger, "GET /me/quotes/{id}/pdf": stranger,
	// Wave 5 (docs/clinic-cms/WAVE5.md): the patient's requests and closure are theirs; the clinic's queue and the
	// restriction are the admin's; the desk files on behalf of a patient the clinic registered (S10).
	"GET /me/data-requests": stranger, "POST /me/data-requests": stranger, "POST /me/data-requests/{id}/withdraw": stranger,
	"GET /me/data-requests/{id}/export": stranger, "GET /me/data-requests/{id}/export/archive": stranger,
	"POST /me/account/close": stranger, "GET /clinic/data-requests": refused, "GET /clinic/data-requests/{id}": refused,
	"POST /clinic/data-requests/{id}/answer": refused, "POST /clinic/data-requests/{id}/extend": refused,
	"POST /clinic/patients/{id}/data-requests": desk, "PUT /clinic/patients/{id}/restriction": refused,
	"DELETE /clinic/patients/{id}/restriction": refused, "GET /clinic/patients/{id}/restriction": refused,
	// Wave 6 (docs/clinic-cms/WAVE6.md): the desk reads the locations, rooms and devices (its day by location, why a
	// device is busy), the admin writes them; lab results and treatment plans are clinical content, never the desk's.
	"GET /clinic/locations": desk, "POST /clinic/locations": refused, "PUT /clinic/locations/{id}": refused,
	"GET /organizations/{id}/locations": public, "GET /clinic/locations/{id}/rooms": desk,
	"POST /clinic/locations/{id}/rooms": refused, "PUT /clinic/rooms/{id}": refused, "GET /clinic/locations/{id}/devices": desk,
	"POST /clinic/locations/{id}/devices": refused, "PUT /clinic/devices/{id}": refused,
	"GET /clinic/appointments/{id}/diagnostic-reports": refused, "POST /clinic/appointments/{id}/diagnostic-reports": refused,
	"GET /clinic/diagnostic-reports/{id}": refused, "PUT /clinic/diagnostic-reports/{id}": refused,
	"GET /clinic/diagnostic-reports/{id}/pdf": refused, "PUT /clinic/diagnostic-reports/{id}/pdf": refused,
	"POST /clinic/diagnostic-reports/{id}/release": refused, "GET /me/diagnostic-reports": stranger,
	"GET /me/diagnostic-reports/{id}": stranger, "GET /me/diagnostic-reports/{id}/pdf": stranger,
	"POST /clinic/care-plans": refused, "GET /clinic/patients/{id}/care-plans": refused, "GET /clinic/care-plans/{id}": refused,
	"PUT /clinic/care-plans/{id}": refused, "POST /clinic/care-plans/{id}/book": refused,
	"POST /clinic/care-plans/{id}/cancel": refused, "GET /me/care-plans": stranger,
}

// notServedYet are the routes in the contract that are not mounted: wave 6's, declared by its contract (6.0) and
// served by LC1, RM1, LB1 and TP1. A route declared before it is built goes here and answers the router's own 404;
// when it is mounted, the test fails until it is taken off this list, and the route is then held to its class above.
var notServedYet = []string{
	"GET /clinic/locations", "POST /clinic/locations", "PUT /clinic/locations/{id}", "GET /organizations/{id}/locations",
	"GET /clinic/locations/{id}/rooms", "POST /clinic/locations/{id}/rooms", "PUT /clinic/rooms/{id}",
	"GET /clinic/locations/{id}/devices", "POST /clinic/locations/{id}/devices", "PUT /clinic/devices/{id}",
	"GET /clinic/appointments/{id}/diagnostic-reports", "POST /clinic/appointments/{id}/diagnostic-reports",
	"GET /clinic/diagnostic-reports/{id}", "PUT /clinic/diagnostic-reports/{id}", "GET /clinic/diagnostic-reports/{id}/pdf",
	"PUT /clinic/diagnostic-reports/{id}/pdf", "POST /clinic/diagnostic-reports/{id}/release", "GET /me/diagnostic-reports",
	"GET /me/diagnostic-reports/{id}", "GET /me/diagnostic-reports/{id}/pdf", "POST /clinic/care-plans",
	"GET /clinic/patients/{id}/care-plans", "GET /clinic/care-plans/{id}", "PUT /clinic/care-plans/{id}",
	"POST /clinic/care-plans/{id}/book", "POST /clinic/care-plans/{id}/cancel", "GET /me/care-plans",
}

// contractOperations reads every operation of api/openapi.yaml as "METHOD /path", with whether it takes no session.
func contractOperations(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	ops := map[string]bool{}
	for path, methods := range doc.Paths {
		for method, op := range methods {
			switch method {
			case "get", "post", "put", "patch", "delete":
				var o struct {
					Security []map[string][]string `yaml:"security"`
				}
				if err := op.Decode(&o); err != nil {
					t.Fatal(err)
				}
				ops[strings.ToUpper(method)+" "+path] = len(o.Security) == 0
			}
		}
	}
	return ops
}

func TestReceptionRoutes_EveryRouteOfTheContractIsClassified(t *testing.T) {
	t.Parallel()
	ops := contractOperations(t)
	for op, open := range ops {
		class, ok := receptionRoutes[op]
		switch {
		case !ok:
			t.Errorf("%s is in api/openapi.yaml but not classified for the receptionist in receptionRoutes", op)
		case open != (class == public):
			t.Errorf("%s: class %d, but the contract says it takes a session: %v", op, class, !open)
		}
	}
	for op := range receptionRoutes {
		if _, ok := ops[op]; !ok {
			t.Errorf("%s is classified but not in api/openapi.yaml", op)
		}
	}
	for _, op := range notServedYet {
		if _, ok := ops[op]; !ok {
			t.Errorf("%s is not served yet but not in api/openapi.yaml", op)
		}
	}
}

// reception is a clinic with its admin, practitioner and receptionist signed in, and a patient with a care consent
// and a booked visit.
type reception struct {
	directory
	owner                                       *pgxpool.Pool
	receptionist, stranger                      *client // stranger: a signed-in account with no patient profile
	visit, patientID, receptionAccount, receipt string  // receipt: one the receptionist issued for the visit
	registered                                  string  // a patient the receptionist registered (docs/clinic-cms R2)
	request                                     string  // the booked patient's data request to the clinic, which names them (wave 5)
}

func newReception(t *testing.T) (reception, func(string, ...any) int) {
	t.Helper()
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	count := openDB(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL) // row-level security on, as in production
	cfg.Analytics.PlatformEnabled = true                // the platform view's routes are served, to be refused
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	r := reception{directory: newDirectory(t, app.Handler, owner), owner: owner, receptionist: &client{t: t, h: app.Handler}}
	r.visit = bookedVisit(t, r.directory, owner)
	if rec := r.admin.do(http.MethodPost, "/invites", `{"email":"Rea@example.test","role":"reception"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /invites for a receptionist = %d %s", rec.Code, rec.Body)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	r.receptionist.bearer = accept(t, r.receptionist, issueToken(t, exec, "rea@example.test"))
	r.stranger = &client{t: t, h: app.Handler}
	r.stranger.do(http.MethodPost, "/auth/register", `{"email":"sam@example.test","password":"`+testPassword+`"}`)
	var login struct{ Token string }
	decode(t, r.stranger.do(http.MethodPost, "/auth/login", `{"email":"sam@example.test","password":"`+testPassword+`"}`), &login)
	r.stranger.bearer = login.Token
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT id::text FROM patient), (SELECT id::text FROM account WHERE email = 'rea@example.test')`).
		Scan(&r.patientID, &r.receptionAccount); err != nil {
		t.Fatal(err)
	}
	r.receipt = deskReceipt(t, r, exec)
	rec := r.receptionist.do(http.MethodPost, "/clinic/patients", `{"givenName":"Rina","familyName":"Registeredova","birthDate":"1966-06-06","consentGivenInPerson":true}`)
	var reg struct{ ID string }
	decode(t, rec, &reg)
	r.registered = reg.ID
	r.request = receptionRequests(t, r)
	return r, count
}

// receptionRequests has the booked patient file an access request to the clinic and the receptionist file a
// restriction request at the desk for the patient they registered (S10): the clinic's queue then names both patients,
// to its admin only (wave 5 walk, finding 3). It returns the booked patient's request.
func receptionRequests(t *testing.T, r reception) string {
	t.Helper()
	rec := r.patient.do(http.MethodPost, "/me/data-requests", `{"kind":"access","target":"clinic","organizationId":"`+inviteClinic+`"}`)
	var filed struct{ ID string }
	decode(t, rec, &filed)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /me/data-requests = %d %s", rec.Code, rec.Body)
	}
	if rec := r.receptionist.do(http.MethodPost, "/clinic/patients/"+r.registered+"/data-requests", `{"kind":"restriction"}`); rec.Code != http.StatusCreated {
		t.Fatalf("the receptionist files on behalf = %d %s", rec.Code, rec.Body)
	}
	if rec := r.admin.do(http.MethodGet, "/clinic/data-requests", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"familyName":"Syntheticova"`) || !strings.Contains(rec.Body.String(), `"familyName":"Registeredova"`) {
		t.Fatalf("the admin's queue = %d %s, want both patients named", rec.Code, rec.Body)
	}
	return filed.ID
}

// deskReceipt gives the clinic its billing details and address and a service, and has the receptionist issue a receipt
// for the visit; it returns the receipt's id.
func deskReceipt(t *testing.T, r reception, exec func(string, ...any)) string {
	t.Helper()
	exec(`UPDATE organization SET legal_name = 'Synthetic clinic d.o.o.', tax_id = '4200000000001', region_id = 'BA-09',
		city_id = 'Q1270190', location_since = now(), address_line = 'Synthetic street 1' WHERE id = $1`, inviteClinic)
	exam := saveService(t, r.directory, `{"name":"Synthetic exam","priceMinor":5000,"modes":["in-person"],"active":true,"translations":[]}`)
	rec := r.receptionist.do(http.MethodPost, "/clinic/receipts", `{"kind":"receipt","appointmentId":"`+r.visit+`","paymentMethod":"cash","lines":[{"serviceId":"`+exam+`","quantity":1}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/receipts as the receptionist = %d %s", rec.Code, rec.Body)
	}
	var out struct{ ID string }
	decode(t, rec, &out)
	return out.ID
}

const unknownID = "0f000000-0000-4000-8000-0000000000f0"

// receptionQueries are the query parameters a route requires, so that it reaches its access check.
var receptionQueries = map[string]string{
	"/analytics/clinic/appointments": "from=2026-01-01&to=2026-06-30",
	"/analytics/clinic/diagnoses":    "from=2026-01-01&to=2026-06-30&by=group",
	"/analytics/platform/cells":      "table=gd&from=2026-01&to=2026-06&by=entity",
	"/analytics/platform/heatmap":    "quarter=2026-Q1",
	"/clinic/noshow-flags":           "from=2026-01-05T00:00:00Z&to=2026-01-12T00:00:00Z",
	"/clinic/receipts":               "from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z",
	"/clinic/reports/practice":       "from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z",
	"/clinic/reports/receipts":       "from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z",
	"/conversations/search":          "q=synthetic",
	"/conversations/{id}/search":     "q=synthetic",
	"/terminology/explanations":      "language=bs",
	"/terminology/search":            "q=synthetic",
}

// receptionPath fills route's parameters and its query. The patient's appointment and profile, and the desk's receipt,
// are the clinic's own (real) or ids that do not exist; the clinic and its practitioner's role, which the public directory lists, are
// always the clinic's own; every other id does not exist.
func receptionPath(r reception, route string, real bool) string {
	visit, patient, receipt, registered, request := r.visit, r.patientID, r.receipt, r.registered, r.request
	if !real {
		visit, patient, receipt, registered, request = unknownID, unknownID, unknownID, unknownID, unknownID
	}
	path := route
	for prefix, id := range map[string]string{
		"/appointments/{id}": visit, "/clinic/appointments/{id}": visit, "/patients/{id}": patient,
		"/practitioner-roles/{id}": r.roleID, "/organizations/{id}": inviteClinic, "/history-sharing/{organizationId}": inviteClinic,
		"/clinic/receipts/{id}": receipt, "/clinic/patients/{id}": registered, "/clinic/data-requests/{id}": request,
	} {
		if strings.HasPrefix(path, prefix) {
			path = strings.Replace(path, prefix[strings.LastIndex(prefix, "{"):], id, 1)
		}
	}
	path = strings.NewReplacer("{id}", unknownID, "{message_id}", unknownID, "{attachment_id}", unknownID,
		"{participant_id}", unknownID, "{token}", "synthetic-token", "{language}", "bs", "{code}", "A00", "{linkId}", "q1").Replace(path)
	if q, ok := receptionQueries[route]; ok {
		path += "?" + q
	}
	return path
}

// The patient's data the bookedVisit fixture holds: none of it may reach a receptionist on a refused route.
var bookedPatientData = []string{"Zorana", "Syntheticova", "1977-07-07", "+38760000777"}

// The data of the patient the receptionist registered: the desk's to see, but not on a refused route (the clinic's
// queue of data requests names them, to its admin only).
var registeredPatientData = []string{"Rina", "Registeredova", "1966-06-06"}

// A receptionist reaches the desk's routes and is refused, as a stranger, on every other route of the contract
// (docs/clinic-cms, F1).
func TestReceptionRoutes_AReceptionistReachesTheDeskAndIsRefusedEverythingElse(t *testing.T) {
	r, count := newReception(t)
	unserved := map[string]bool{}
	for _, op := range notServedYet {
		unserved[op] = true
	}
	routes := make([]string, 0, len(receptionRoutes))
	for op := range receptionRoutes {
		routes = append(routes, op)
	}
	slices.Sort(routes) // in order, so that a route changing state (a logout) can be placed last
	for _, op := range routes {
		class := receptionRoutes[op]
		if class == public || op == "POST /auth/logout" {
			continue
		}
		method, route, _ := strings.Cut(op, " ")
		r.receptionist.ifMatch = ""
		if method == http.MethodPatch || method == http.MethodPut {
			r.receptionist.ifMatch = `W/"1"` // past the precondition, to the access check
		}
		rec := r.receptionist.do(method, receptionPath(r, route, true), "")
		body := rec.Body.String()
		routerNotFound := (rec.Code == http.StatusNotFound && body == "404 page not found\n") ||
			(rec.Code == http.StatusMethodNotAllowed && body == "Method Not Allowed\n")
		if routerNotFound != unserved[op] {
			t.Errorf("%s: served %v, want %v (take a route the package has mounted off notServedYet)", op, !routerNotFound, !unserved[op])
			continue
		}
		if unserved[op] {
			continue
		}
		checkClass(t, r, op, class, rec.Code, body)
	}
	if rec := r.receptionist.do(http.MethodPost, "/auth/logout", ""); rec.Code != http.StatusNoContent {
		t.Errorf("POST /auth/logout as the receptionist = %d %s, want 204", rec.Code, rec.Body)
	}
	// Every refused read of a patient's data is audited with the receptionist as the agent, as any staff read.
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_account_id = $1 AND outcome = 4`, r.receptionAccount); n == 0 {
		t.Error("no refusal of the receptionist was audited")
	}
}

// checkClass holds the receptionist's answer on op (code, body) to the route's class.
func checkClass(t *testing.T, r reception, op string, class receptionClass, code int, body string) {
	t.Helper()
	method, route, _ := strings.Cut(op, " ")
	switch class {
	case own, desk:
		if code == http.StatusUnauthorized || code == http.StatusForbidden || code == http.StatusNotFound || code >= 500 {
			t.Errorf("%s as the receptionist = %d %s, want the desk's answer", op, code, body)
		}
	case refused:
		checkRefused(t, r, op, code, body)
	case stranger:
		if code == http.StatusForbidden || code == http.StatusNotFound {
			checkRefused(t, r, op, code, body)
		} else if want := r.stranger.do(method, receptionPath(r, route, true), ""); want.Code != code || want.Body.String() != body {
			t.Errorf("%s as the receptionist = %d %s, want what a stranger gets: %d %s", op, code, body, want.Code, want.Body)
		}
	case machine:
		if code != http.StatusUnauthorized {
			t.Errorf("%s as the receptionist = %d %s, want 401", op, code, body)
		}
	}
}

// checkRefused: a refused route answers the receptionist 403 or 404, the same for the clinic's own resources as for
// ids that do not exist, with none of the patient's data.
func checkRefused(t *testing.T, r reception, op string, code int, body string) {
	t.Helper()
	method, route, _ := strings.Cut(op, " ")
	// A write without a body is rejected before any access (invalid_request): that reveals nothing either, and is
	// held to the same comparison below.
	malformed := code == http.StatusBadRequest && body == `{"error":"invalid_request"}`+"\n" && method != http.MethodGet
	// A live stream refuses a plain request before any access too; the stream's own access is its domain's tests'.
	malformed = malformed || (code == http.StatusBadRequest && body == `{"error":"websocket_required"}`+"\n")
	if code != http.StatusForbidden && code != http.StatusNotFound && !malformed {
		t.Errorf("%s as the receptionist = %d %s, want 403 or 404", op, code, body)
	}
	for _, s := range append(slices.Clone(bookedPatientData), registeredPatientData...) {
		if strings.Contains(body, s) {
			t.Errorf("%s as the receptionist: the body holds the patient's data", op)
		}
	}
	if real, unknown := receptionPath(r, route, true), receptionPath(r, route, false); real != unknown {
		rec := r.receptionist.do(method, unknown, "")
		if rec.Code != code || rec.Body.String() != body {
			t.Errorf("%s as the receptionist = %d %s for the clinic's own, %d %s for an unknown id: the answer reveals what exists",
				op, code, body, rec.Code, rec.Body)
		}
	}
}

// The receptionist's own session says what they are, and the price list is theirs to read.
func TestReceptionRoutes_AReceptionistSeesTheirRoleAndThePriceList(t *testing.T) {
	r, _ := newReception(t)
	if rec := r.receptionist.do(http.MethodGet, "/auth/me", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"role":"reception"`) {
		t.Errorf("GET /auth/me as the receptionist = %d %s, want the reception role", rec.Code, rec.Body)
	}
	saveService(t, r.directory, `{"name":"Synthetic exam","priceMinor":5000,"modes":["in-person"],"active":true,"translations":[]}`)
	if rec := r.receptionist.do(http.MethodGet, "/clinic/services", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Synthetic exam") {
		t.Errorf("GET /clinic/services as the receptionist = %d %s, want the price list", rec.Code, rec.Body)
	}
}
