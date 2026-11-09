package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const otherClinic = "b0000000-0000-4000-8000-00000000000b"

// matrix is the directory's clinic, listed, with a second practitioner, the admin of another clinic, a patient
// with a profile, an active consent and a future and a past appointment with the first practitioner.
type matrix struct {
	directory
	other, otherAdmin, anonymous      *client
	patientID, consentID              string
	future, past, scheduleID, slotID  string
	practitionerAccount, otherAccount string
	cfgBase                           string
}

func newMatrix(t *testing.T, h http.Handler, owner *pgxpool.Pool, base string) matrix {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	m := matrix{directory: newDirectory(t, h, owner), other: &client{t: t, h: h}, otherAdmin: &client{t: t, h: h}, anonymous: &client{t: t, h: h}, cfgBase: base}
	if rec := m.admin.do(http.MethodPost, "/invites", `{"email":"eve@example.test","role":"practitioner","givenName":"Eve","familyName":"Sample"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /invites = %d %s", rec.Code, rec.Body)
	}
	m.other.bearer = accept(t, m.other, issueToken(t, exec, "eve@example.test"))
	exec("INSERT INTO organization (id, name) VALUES ($1, 'Other synthetic clinic')", otherClinic)
	exec("INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', 'fay@example.test')", otherClinic)
	m.otherAdmin.bearer = accept(t, m.otherAdmin, issueToken(t, exec, "fay@example.test"))

	m.slotID = seedSlots(t, owner, m.roleID, 7*24*time.Hour)[0]
	if rec := m.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := m.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	rec := m.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+m.slotID+`"}`)
	var booked struct{ ID string }
	decode(t, rec, &booked)
	m.future = booked.ID
	err := owner.QueryRow(t.Context(), `WITH p AS (SELECT id FROM patient),
		a AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
			SELECT id, $1, $2, gen_random_uuid(), tstzrange(now() - interval '2 hours', now() - interval '90 minutes'), now() - interval '1 day' FROM p RETURNING id)
		SELECT (SELECT id::text FROM a), (SELECT id::text FROM p), (SELECT id::text FROM consent), (SELECT id::text FROM schedule),
			(SELECT id::text FROM account WHERE email = 'cy@example.test'), (SELECT id::text FROM account WHERE email = 'eve@example.test')`,
		m.roleID, inviteClinic).Scan(&m.past, &m.patientID, &m.consentID, &m.scheduleID, &m.practitionerAccount, &m.otherAccount)
	if err != nil || m.future == "" {
		t.Fatalf("seeding the appointments: %v, booking %d %s", err, rec.Code, rec.Body)
	}
	return m
}

// expect checks the status of each GET, and that a 200 holds want (when given).
func expect(t *testing.T, label string, c *client, path string, status int, want ...string) string {
	t.Helper()
	rec := c.do(http.MethodGet, path, "")
	if rec.Code != status {
		t.Errorf("%s: GET %s = %d %s, want %d", label, path, rec.Code, rec.Body, status)
	}
	for _, w := range want {
		if !strings.Contains(rec.Body.String(), w) {
			t.Errorf("%s: GET %s = %s, want it to hold %s", label, path, rec.Body, w)
		}
	}
	return rec.Body.String()
}

// total is a searchset's total.
func total(t *testing.T, body string) int {
	t.Helper()
	var b struct{ Total int }
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatalf("not a Bundle: %s", body)
	}
	return b.Total
}

// The public directory, the access matrix of SPEC.md "Access rule" through FHIR with row-level security on
// (salus_app), and the clinic after a revocation.
func TestNew_ServesTheBookingResourcesOverFHIRAsAccessAllows(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	m := newMatrix(t, app.Handler, owner, cfg.FHIRBaseURL)
	count := openDB(t, cfg)

	checkListing(t, m, count)
	checkWithConsent(t, m, count)
	if rec := m.patient.do(http.MethodPost, "/consents/"+m.consentID+"/revoke", ""); rec.Code != http.StatusOK {
		t.Fatalf("revoking = %d %s", rec.Code, rec.Body)
	}
	checkAfterRevocation(t, m)
	for _, field := range []string{"Zorana", "Syntheticova", "1977", "38760000777", "ana@example.test"} {
		if strings.Contains(log.String(), field) {
			t.Errorf("the log holds %q", field)
		}
	}
}

// checkListing: the admin lists the clinic through FHIR, and anyone then finds it, its practitioner, the
// schedule and the free slots, without an audit event.
func checkListing(t *testing.T, m matrix, count func(string, ...any) int) {
	t.Helper()
	org := "/fhir/R4/Organization/" + inviteClinic
	m.admin.ifMatch = `W/"1"` // seedSlots listed it as the owner, a direct write that keeps the version
	body := `{"resourceType":"Organization","id":"` + inviteClinic + `","name":"Synthetic clinic","extension":[` +
		`{"url":"https://salus.invalid/fhir/StructureDefinition/organization-listed","valueBoolean":true},` +
		`{"url":"https://salus.invalid/fhir/StructureDefinition/organization-timezone","valueCode":"Europe/Vienna"}]}`
	if rec := m.admin.do(http.MethodPut, org, body); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"valueBoolean":true`) ||
		!strings.Contains(rec.Body.String(), "Europe/Vienna") {
		t.Errorf("listing the clinic = %d %s, want it listed in Vienna's zone", rec.Code, rec.Body)
	}
	m.admin.ifMatch = ""
	before := count("SELECT count(*) FROM audit_event")
	for _, c := range []*client{m.anonymous, m.patient} {
		expect(t, "the public directory", c, "/fhir/R4/Organization?listed=true", http.StatusOK, `"id":"`+inviteClinic+`"`)
		body := expect(t, "the public directory", c, "/fhir/R4/PractitionerRole?organization="+inviteClinic+"&_include=PractitionerRole:practitioner",
			http.StatusOK, `"fullUrl":"`+m.cfgBase+"/Practitioner/"+m.practitionerID+`"`, `"family":"Example"`)
		if total(t, body) != 2 {
			t.Errorf("the public roles = %s, want both practitioners' roles", body)
		}
		expect(t, "the public directory", c, "/fhir/R4/Schedule?actor=PractitionerRole/"+m.roleID, http.StatusOK, `"id":"`+m.scheduleID+`"`)
		expect(t, "the public directory", c, "/fhir/R4/Schedule/"+m.scheduleID, http.StatusOK)
		lt := url.QueryEscape("lt" + clock.System{}.Now().Add(30*24*time.Hour).UTC().Format(time.RFC3339))
		if body := expect(t, "the public directory", c, "/fhir/R4/Slot?schedule="+m.scheduleID+"&status=free&start="+lt, http.StatusOK); total(t, body) != 0 {
			t.Errorf("free slots = %s, want none: the one slot is booked", body)
		}
	}
	expect(t, "an unlisted clinic", m.anonymous, "/fhir/R4/PractitionerRole?organization="+otherClinic, http.StatusOK, `"total":0`)
	expect(t, "the unknown search", m.anonymous, "/fhir/R4/Slot?schedule="+m.scheduleID+"&status=free&start=lt2030-01-01T00:00:00Z&_sort=start", http.StatusBadRequest)
	if after := count("SELECT count(*) FROM audit_event"); after != before {
		t.Errorf("the public reads wrote %d audit events, want none", after-before)
	}
}

// checkWithConsent: with the patient's active consent, who sees what (SPEC.md "Access rule"), and the audit
// of a search per returned resource.
func checkWithConsent(t *testing.T, m matrix, count func(string, ...any) int) {
	t.Helper()
	patientPath, future, consentPath := "/fhir/R4/Patient/"+m.patientID, "/fhir/R4/Appointment/"+m.future, "/fhir/R4/Consent/"+m.consentID
	byCy := "/fhir/R4/Appointment?practitioner=Practitioner/" + m.practitionerID
	const name = `"display":"Zorana Syntheticova"`
	for _, tt := range []struct {
		c                                 *client
		label                             string
		patient, appointment, consent, cy int
	}{
		{m.patient, "the patient", http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK},
		{m.practitioner, "their practitioner", http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK},
		{m.other, "another practitioner", http.StatusNotFound, http.StatusNotFound, http.StatusOK, http.StatusForbidden},
		{m.admin, "the admin", http.StatusNotFound, http.StatusOK, http.StatusOK, http.StatusOK},
		{m.otherAdmin, "another clinic", http.StatusNotFound, http.StatusNotFound, http.StatusNotFound, http.StatusOK},
		{m.anonymous, "nobody signed in", http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized, http.StatusUnauthorized},
	} {
		expect(t, tt.label, tt.c, patientPath, tt.patient)
		if body := expect(t, tt.label, tt.c, future, tt.appointment); tt.appointment == http.StatusOK && !strings.Contains(body, name) {
			t.Errorf("%s: the appointment %s, want the patient's name", tt.label, body)
		}
		expect(t, tt.label, tt.c, consentPath, tt.consent)
		body := expect(t, tt.label, tt.c, byCy, tt.cy)
		want := map[string]int{"another clinic": 0}[tt.label]
		if tt.label != "another clinic" {
			want = 2
		}
		if tt.cy == http.StatusOK && total(t, body) != want {
			t.Errorf("%s: %s = %s, want %d appointments", tt.label, byCy, body, want)
		}
	}
	patientRef := "Patient/" + m.patientID
	for _, tt := range []struct {
		name, where string
		args        []any
		want        int
	}{
		{"the practitioner's profile reads (the Patient read, and the name with the appointment's)",
			"subtype_code = 'read' AND outcome = 0 AND entity_reference = $1 AND purpose_code = 'TREAT' AND agent_account_id = $2", []any{patientRef, m.practitionerAccount}, 2},
		{"the other practitioner's refused reads", "subtype_code = 'read' AND outcome = 4 AND entity_reference IS NULL AND agent_account_id = $1", []any{m.otherAccount}, 2},
		{"the practitioner's search, per appointment", "subtype_code = 'search-type' AND outcome = 0 AND entity_reference LIKE 'Appointment/%' AND agent_account_id = $1",
			[]any{m.practitionerAccount}, 2},
		{"the practitioner's search, per name given", "subtype_code = 'search-type' AND outcome = 0 AND entity_reference = $1 AND agent_account_id = $2",
			[]any{patientRef, m.practitionerAccount}, 2},
	} {
		if got := count("SELECT count(*) FROM audit_event WHERE type_code = 'rest' AND "+tt.where, tt.args...); got != tt.want {
			t.Errorf("audit events for %s: %d, want %d", tt.name, got, tt.want)
		}
	}
}

// checkAfterRevocation: once the revocation commits, before its job runs, the clinic keeps its past
// appointment with the name, times and status only, and loses the profile and the future appointment.
func checkAfterRevocation(t *testing.T, m matrix) {
	t.Helper()
	past := "/fhir/R4/Appointment/" + m.past
	for _, c := range []*client{m.practitioner, m.admin} {
		expect(t, "after the revocation", c, "/fhir/R4/Patient/"+m.patientID, http.StatusNotFound)
		expect(t, "after the revocation", c, "/fhir/R4/Appointment/"+m.future, http.StatusNotFound)
		body := expect(t, "after the revocation", c, past, http.StatusOK, `"display":"Zorana Syntheticova"`, `"status":"booked"`)
		if strings.Contains(body, `"slot"`) || strings.Contains(body, `"created"`) {
			t.Errorf("the past appointment after the revocation = %s, want the name, times and status only", body)
		}
		expect(t, "after the revocation", c, "/fhir/R4/Consent/"+m.consentID, http.StatusOK, `"status":"inactive"`)
	}
	if body := expect(t, "after the revocation", m.practitioner, "/fhir/R4/Appointment?practitioner=Practitioner/"+m.practitionerID, http.StatusOK); total(t, body) != 1 ||
		!strings.Contains(body, m.past) {
		t.Errorf("the practitioner's search after the revocation = %s, want the past appointment only", body)
	}
	if body := expect(t, "after the revocation", m.patient, "/fhir/R4/Appointment?patient=Patient/"+m.patientID, http.StatusOK); total(t, body) != 2 {
		t.Errorf("the patient's search after the revocation = %s, want both of their appointments", body)
	}
	expect(t, "after the revocation", m.patient, "/fhir/R4/Patient/"+m.patientID, http.StatusOK, `"birthDate":"1977-07-07"`)
}
