package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// SAL-33 through api.New: the patient reads and replaces their notification settings (SMS only with a phone in their
// profile), staff may not; the clinic's admin reads its SMS usage, its practitioner may not. Audited; the log holds no
// zone.
func TestNew_ServesThePatientsNotificationSettingsAndTheSMSUsage(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)

	expectHTTP(t, d.patient, http.MethodGet, "/me/notification-settings", http.StatusOK,
		`"channels":{"push":true,"email":true,"sms":false}`, `"quiet_hours":{"start":"22:00","end":"08:00"}`)
	sms := `{"channels":{"push":true,"email":false,"sms":true},"quiet_hours":null,"time_zone":"Asia/Dubai"}`
	if rec := d.patient.do(http.MethodPut, "/me/notification-settings", sms); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "phone_required") {
		t.Errorf("SMS without a phone = %d %s, want 409 phone_required", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPut, "/me/notification-settings", sms); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Asia/Dubai") {
		t.Errorf("PUT settings = %d %s", rec.Code, rec.Body)
	}
	body := expectHTTP(t, d.patient, http.MethodGet, "/me/notification-settings", http.StatusOK, `"sms":true`, `"time_zone":"Asia/Dubai"`)
	if strings.Contains(body, "quiet_hours") {
		t.Errorf("quiet hours turned off are still there: %s", body)
	}
	for _, staff := range []*client{d.admin, d.practitioner} {
		expectHTTP(t, staff, http.MethodGet, "/me/notification-settings", http.StatusForbidden)
	}
	expectHTTP(t, d.admin, http.MethodGet, "/clinic/sms-usage?from=2026-01&to=2026-12", http.StatusOK, `{"months":[]}`)
	expectHTTP(t, d.practitioner, http.MethodGet, "/clinic/sms-usage?from=2026-01&to=2026-12", http.StatusForbidden)
	expectHTTP(t, d.admin, http.MethodGet, "/clinic/sms-usage?from=2026-12&to=2026-01", http.StatusBadRequest)
	if n := count("SELECT count(*) FROM audit_event WHERE entity_name = 'notification_preference'"); n < 4 {
		t.Errorf("%d audit events of the settings, want every read and change", n)
	}
	if strings.Contains(log.String(), "Asia/Dubai") {
		t.Error("the log holds the patient's zone")
	}
}
