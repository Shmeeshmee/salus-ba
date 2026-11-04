package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// directory is a clinic with its admin and one practitioner, both signed in, and a signed-in patient.
type directory struct {
	admin, practitioner, patient *client
	practitionerID, roleID       string
}

func ownerPool(t *testing.T, cfg config.Config) *pgxpool.Pool {
	t.Helper()
	owner, err := postgres.Open(t.Context(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Close)
	return owner
}

func newDirectory(t *testing.T, h http.Handler, owner *pgxpool.Pool) directory {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic')", inviteClinic)
	exec("INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', 'dee@example.test')", inviteClinic)
	d := directory{admin: &client{t: t, h: h}, practitioner: &client{t: t, h: h}, patient: &client{t: t, h: h}}
	d.admin.bearer = accept(t, d.admin, issueToken(t, exec, "dee@example.test"))
	if rec := d.admin.do(http.MethodPost, "/invites", `{"email":"cy@example.test","role":"practitioner","givenName":"Cy","familyName":"Example"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /invites = %d %s", rec.Code, rec.Body)
	}
	d.practitioner.bearer = accept(t, d.practitioner, issueToken(t, exec, "cy@example.test"))

	d.patient.do(http.MethodPost, "/auth/register", `{"email":"ana@example.test","password":"`+testPassword+`"}`)
	rec := d.patient.do(http.MethodPost, "/auth/login", `{"email":"ana@example.test","password":"`+testPassword+`"}`)
	var login struct{ Token string }
	decode(t, rec, &login)
	d.patient.bearer = login.Token

	if err := owner.QueryRow(t.Context(), "SELECT p.id::text, r.id::text FROM practitioner p JOIN practitioner_role r ON r.practitioner_id = p.id").
		Scan(&d.practitionerID, &d.roleID); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestNew_ServesTheDirectoryOverFHIRToStaffOfTheClinicAndAuditsIt(t *testing.T) {
	app, cfg := newApp(t)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	audited := func(where string, args ...any) int {
		t.Helper()
		return count("SELECT count(*) FROM audit_event WHERE type_code = 'rest' AND "+where, args...)
	}
	baseline := audited("true") // the invites, acceptances and directory creates

	org := "/fhir/R4/Organization/" + inviteClinic
	if rec := d.admin.do(http.MethodGet, org, ""); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("GET %s as the admin = %d %s %v, want 200 at version 1", org, rec.Code, rec.Body, rec.Header())
	}
	rename := `{"resourceType":"Organization","id":"` + inviteClinic + `","name":"Renamed synthetic clinic"}`
	d.admin.ifMatch = `W/"1"`
	if rec := d.admin.do(http.MethodPut, org, rename); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` ||
		!strings.Contains(rec.Body.String(), "Renamed synthetic clinic") {
		t.Errorf("PUT %s at version 1 = %d %s, want 200 at version 2", org, rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPut, org, rename); rec.Code != http.StatusPreconditionFailed {
		t.Errorf("PUT %s at the stale version 1 = %d %s, want 412", org, rec.Code, rec.Body)
	}
	d.practitioner.ifMatch = `W/"2"`
	if rec := d.practitioner.do(http.MethodPut, org, rename); rec.Code != http.StatusForbidden {
		t.Errorf("PUT %s as the practitioner = %d %s, want 403", org, rec.Code, rec.Body)
	}

	search := "/fhir/R4/PractitionerRole?organization=" + inviteClinic + "&_include=PractitionerRole:practitioner"
	rec := d.practitioner.do(http.MethodGet, search, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"fullUrl":"`+cfg.FHIRBaseURL+"/Practitioner/"+d.practitionerID+`"`) ||
		!strings.Contains(rec.Body.String(), `"fullUrl":"`+cfg.FHIRBaseURL+"/PractitionerRole/"+d.roleID+`"`) {
		t.Errorf("GET %s = %d %s, want the role and its practitioner", search, rec.Code, rec.Body)
	}
	for _, path := range []string{"/fhir/R4/Practitioner/" + d.practitionerID, "/fhir/R4/PractitionerRole/" + d.roleID} {
		if rec := d.admin.do(http.MethodGet, path, ""); rec.Code != http.StatusOK {
			t.Errorf("GET %s as the admin = %d %s, want 200", path, rec.Code, rec.Body)
		}
	}

	for _, path := range []string{org, "/fhir/R4/Practitioner/" + d.practitionerID, "/fhir/R4/PractitionerRole/" + d.roleID, search} {
		if rec := d.patient.do(http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s as a patient = %d %s, want 404", path, rec.Code, rec.Body)
		}
		if rec := (&client{t: t, h: app.Handler}).do(http.MethodGet, path, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s signed out = %d %s, want 401", path, rec.Code, rec.Body)
		}
	}

	var adminID, practitionerAccount string
	if err := owner.QueryRow(t.Context(), `SELECT (SELECT id::text FROM account WHERE email = 'dee@example.test'),
		(SELECT id::text FROM account WHERE email = 'cy@example.test')`).Scan(&adminID, &practitionerAccount); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, where string
		args        []any
		want        int
	}{
		{"the rename", "subtype_code = 'update' AND action = 'U' AND outcome = 0 AND entity_reference = $1 AND agent_account_id = $2",
			[]any{"Organization/" + inviteClinic, adminID}, 1},
		{"the stale rename", "subtype_code = 'update' AND outcome = 4 AND entity_reference = $1 AND agent_account_id = $2",
			[]any{"Organization/" + inviteClinic, adminID}, 1},
		{"the practitioner's rename", "subtype_code = 'update' AND outcome = 4 AND entity_reference IS NULL AND agent_account_id = $1",
			[]any{practitionerAccount}, 1},
		{"the search's match", "subtype_code = 'search-type' AND action = 'E' AND outcome = 0 AND entity_reference = $1 AND agent_account_id = $2",
			[]any{"PractitionerRole/" + d.roleID, practitionerAccount}, 1},
		{"the search's include", "subtype_code = 'read' AND action = 'R' AND outcome = 0 AND entity_reference = $1 AND agent_account_id = $2",
			[]any{"Practitioner/" + d.practitionerID, practitionerAccount}, 1},
		{"the admin's reads", "subtype_code = 'read' AND outcome = 0 AND entity_reference IN ($1, $2) AND agent_account_id = $3 AND organization_id = $4",
			[]any{"Practitioner/" + d.practitionerID, "PractitionerRole/" + d.roleID, adminID, inviteClinic}, 2},
		{"the patient's refusals", "outcome = 4 AND entity_reference IS NULL AND organization_id IS NULL AND subtype_code IN ('read', 'search-type')", nil, 3},
	} {
		if n := audited(tt.where, tt.args...); n != tt.want {
			t.Errorf("%s: %d audit events, want %d", tt.name, n, tt.want)
		}
	}
	// Nothing else: Organization reads and signed-out requests are not audited.
	if n := audited("true") - baseline; n != 10 {
		t.Errorf("%d audit events for the FHIR requests, want 10", n)
	}
}
