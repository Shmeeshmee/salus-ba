package api_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// syncBuffer is a log buffer the handler's goroutines may write concurrently.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestNew_APatientSavesAndReadsTheirProfileAuditedAndStaffCannot(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, ownerPool(t, cfg))
	count := openDB(t, cfg)

	if rec := d.patient.do(http.MethodGet, "/me/patient", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET /me/patient before the first save = %d %s, want 404", rec.Code, rec.Body)
	}
	create := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`
	rec := d.patient.do(http.MethodPut, "/me/patient", create)
	var saved struct {
		ID        string `json:"id"`
		VersionID int    `json:"versionId"`
	}
	decode(t, rec, &saved)
	if rec.Code != http.StatusCreated || saved.VersionID != 1 {
		t.Fatalf("PUT /me/patient = %d %s, want 201 at version 1", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", create); rec.Code != http.StatusConflict {
		t.Errorf("a second create = %d %s, want 409", rec.Code, rec.Body)
	}
	update := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-08","versionId":1}`
	if rec := d.patient.do(http.MethodPut, "/me/patient", update); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"versionId":2`) {
		t.Errorf("PUT /me/patient at version 1 = %d %s, want 200 at version 2", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", update); rec.Code != http.StatusConflict {
		t.Errorf("PUT /me/patient at the stale version 1 = %d %s, want 409", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/me/patient", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"birthDate":"1977-07-08"`) ||
		strings.Contains(rec.Body.String(), "phone") {
		t.Errorf("GET /me/patient = %d %s, want the updated profile without a phone", rec.Code, rec.Body)
	}
	for _, staff := range []*client{d.admin, d.practitioner} {
		if rec := staff.do(http.MethodGet, "/me/patient", ""); rec.Code != http.StatusForbidden {
			t.Errorf("GET /me/patient as staff = %d %s, want 403", rec.Code, rec.Body)
		}
		if rec := staff.do(http.MethodPut, "/me/patient", create); rec.Code != http.StatusForbidden {
			t.Errorf("PUT /me/patient as staff = %d %s, want 403", rec.Code, rec.Body)
		}
	}
	if rec := (&client{t: t, h: app.Handler}).do(http.MethodGet, "/me/patient", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /me/patient signed out = %d %s, want 401", rec.Code, rec.Body)
	}

	entity := "Patient/" + saved.ID
	for _, tt := range []struct {
		name, where string
		args        []any
		want        int
	}{
		{"the create", "subtype_code = 'create' AND outcome = 0 AND entity_reference = $1", []any{entity}, 1},
		{"the update", "subtype_code = 'update' AND outcome = 0 AND entity_reference = $1", []any{entity}, 1},
		{"the read", "subtype_code = 'read' AND outcome = 0 AND entity_reference = $1", []any{entity}, 1},
		{"the patient's refusals", "outcome = 4 AND purpose_code = 'PATRQT' AND organization_id IS NULL", nil, 3},
		{"the staff refusals", "outcome = 4 AND purpose_code = 'TREAT' AND organization_id = $1", []any{inviteClinic}, 4},
	} {
		if got := count("SELECT count(*) FROM audit_event WHERE type_code = 'rest' AND "+tt.where, tt.args...); got != tt.want {
			t.Errorf("audit events for %s: %d, want %d", tt.name, got, tt.want)
		}
	}
	for _, field := range []string{"Zorana", "Syntheticova", "1977", "38760000777"} {
		if strings.Contains(log.String(), field) {
			t.Errorf("the log holds %q:\n%s", field, log.String())
		}
	}
}
