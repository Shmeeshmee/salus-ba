package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// The clinic's statistics (SAL-20): its admin gets the breakdown per practitioner, its practitioner the clinic's
// figures without it and their own diagnoses; a patient is refused, nobody is unauthenticated; every read and
// refusal is audited on the clinic.
func TestNew_ServesTheClinicsStatisticsToItsStaffAndAuditsThem(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	const appts = "/analytics/clinic/appointments?from=2026-01-01&to=2026-12-31"
	const diags = "/analytics/clinic/diagnoses?from=2026-01-01&to=2026-12-31&by=group"

	if rec := d.admin.do(http.MethodGet, appts, ""); rec.Code != http.StatusOK || rec.Body.String() != `{"practitioners":[],"series":[],"heatmap":[]}`+"\n" {
		t.Errorf("GET appointments as the admin = %d %s", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodGet, appts, ""); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "practitioners") {
		t.Errorf("GET appointments as the practitioner = %d %s, want no practitioners", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodGet, diags+"&practitionerRole="+d.roleID, ""); rec.Code != http.StatusOK || rec.Body.String() != `{"rows":[]}`+"\n" {
		t.Errorf("GET their own diagnoses as the practitioner = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, diags, ""); rec.Code != http.StatusForbidden {
		t.Errorf("GET diagnoses as a patient = %d %s, want 403", rec.Code, rec.Body)
	}
	if rec := (&client{t: t, h: app.Handler}).do(http.MethodGet, diags, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET diagnoses signed out = %d, want 401", rec.Code)
	}
	var read, refused int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE outcome = 0 AND entity_reference = 'Organization/' || $1),
			count(*) FILTER (WHERE outcome = 4)
		FROM audit_event WHERE subtype_code = 'search-type' AND purpose_code = 'HOPERAT'`, inviteClinic).Scan(&read, &refused); err != nil || read != 3 || refused != 1 {
		t.Errorf("audited %d reads of the clinic and %d refusals, %v; want 3 and 1", read, refused, err)
	}
}
