package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

func TestNew_APatientGrantsListsAndRevokesConsentsAuditedAndStaffCannot(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, ownerPool(t, cfg))
	count := openDB(t, cfg)
	grant := `{"organizationId":"` + inviteClinic + `"}`

	if rec := d.patient.do(http.MethodPost, "/consents", grant); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "profile_required") {
		t.Errorf("POST /consents without a profile = %d %s, want 409 profile_required", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/me/consents", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"consents":[]}`+"\n" {
		t.Errorf("GET /me/consents without a profile = %d %s, want 200 and none", rec.Code, rec.Body)
	}
	profile := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`
	if rec := d.patient.do(http.MethodPut, "/me/patient", profile); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}

	rec := d.patient.do(http.MethodPost, "/consents", grant)
	var first struct{ ID, Status, OrganizationID string }
	decode(t, rec, &first)
	if rec.Code != http.StatusCreated || first.Status != "active" || first.OrganizationID != inviteClinic {
		t.Fatalf("POST /consents = %d %s, want 201 active with the clinic", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", grant); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "consent_active") {
		t.Errorf("a second grant = %d %s, want 409 consent_active", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"f0000000-0000-4000-8000-00000000000f"}`); rec.Code != http.StatusNotFound {
		t.Errorf("a grant to an unknown clinic = %d %s, want 404", rec.Code, rec.Body)
	}
	for _, staff := range []*client{d.admin, d.practitioner} {
		for _, req := range [][2]string{{http.MethodPost, "/consents"}, {http.MethodPost, "/consents/" + first.ID + "/revoke"}, {http.MethodGet, "/me/consents"}} {
			if rec := staff.do(req[0], req[1], grant); rec.Code != http.StatusForbidden {
				t.Errorf("%s %s as staff = %d %s, want 403", req[0], req[1], rec.Code, rec.Body)
			}
		}
	}
	if rec := (&client{t: t, h: app.Handler}).do(http.MethodGet, "/me/consents", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /me/consents signed out = %d %s, want 401", rec.Code, rec.Body)
	}

	if rec := d.patient.do(http.MethodPost, "/consents/"+first.ID+"/revoke", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"status":"inactive"`) || !strings.Contains(rec.Body.String(), `"revokedAt"`) {
		t.Errorf("POST revoke = %d %s, want 200 inactive with revokedAt", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents/"+first.ID+"/revoke", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "consent_inactive") {
		t.Errorf("a second revocation = %d %s, want 409 consent_inactive", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents/f0000000-0000-4000-8000-00000000000f/revoke", ""); rec.Code != http.StatusNotFound {
		t.Errorf("revoking an unknown consent = %d %s, want 404", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", grant); rec.Code != http.StatusCreated || strings.Contains(rec.Body.String(), first.ID) {
		t.Errorf("granting again = %d %s, want 201 with a new consent", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/me/consents", ""); rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"id"`) != 2 {
		t.Errorf("GET /me/consents = %d %s, want both consents", rec.Code, rec.Body)
	}

	checkConsentTrail(t, count, "Consent/"+first.ID, log.String())
}

// checkConsentTrail checks the audit events of TestNew_APatientGrantsListsAndRevokesConsentsAuditedAndStaffCannot
// and that its log holds none of the patient's data.
func checkConsentTrail(t *testing.T, count func(q string, args ...any) int, entity, log string) {
	t.Helper()
	for _, tt := range []struct {
		name, where string
		args        []any
		want        int
	}{
		{"the grant", "subtype_code = 'create' AND outcome = 0 AND entity_reference = $1", []any{entity}, 1},
		{"the revocation", "subtype_code = 'update' AND outcome = 0 AND entity_reference = $1", []any{entity}, 1},
		{"the lists", "subtype_code = 'search-type' AND outcome = 0 AND purpose_code = 'PATRQT'", nil, 3},
		{"the patient's refusals", "entity_reference IS NULL AND outcome = 4 AND purpose_code = 'PATRQT' AND organization_id IS NULL" +
			" AND subtype_code IN ('create', 'update')", nil, 5},
		{"the staff refusals", "outcome = 4 AND purpose_code = 'TREAT' AND organization_id = $1 AND subtype_code <> 'read'", []any{inviteClinic}, 6},
		{"the profile reads that resolve the patient", "subtype_code = 'read' AND entity_reference LIKE 'Patient/%' AND outcome = 0", nil, 8},
	} {
		if got := count("SELECT count(*) FROM audit_event WHERE type_code = 'rest' AND "+tt.where, tt.args...); got != tt.want {
			t.Errorf("audit events for %s: %d, want %d", tt.name, got, tt.want)
		}
	}
	for _, field := range []string{"Zorana", "Syntheticova", "1977", "38760000777", "ana@example.test"} {
		if strings.Contains(log, field) {
			t.Errorf("the log holds %q:\n%s", field, log)
		}
	}
}
