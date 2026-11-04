package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

func TestNew_TheClinicsAdminSetsARolesSpecialtyAuditedAndNobodyElseCan(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	path := "/practitioner-roles/" + d.roleID + "/specialty"
	const cardiology = `{"code":"394579002"}`

	d.admin.ifMatch = `W/"1"`
	rec := d.admin.do(http.MethodPut, path, cardiology)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || !strings.Contains(rec.Body.String(), `"display":"Cardiology"`) {
		t.Errorf("PUT %s as the admin at version 1 = %d %s, want 200 at version 2 with cardiology", path, rec.Code, rec.Body)
	}
	for _, tt := range []struct {
		name   string
		c      *client
		body   string
		status int
	}{
		{"the stale version", d.admin, cardiology, http.StatusPreconditionFailed},
		{"a code not in the list", d.admin, `{"code":"394592004"}`, http.StatusBadRequest},
		{"the practitioner", d.practitioner, cardiology, http.StatusForbidden},
		{"a patient", d.patient, cardiology, http.StatusNotFound},
		{"nobody", &client{t: t, h: app.Handler}, cardiology, http.StatusUnauthorized},
	} {
		tt.c.ifMatch = `W/"1"`
		if tt.c == d.practitioner {
			tt.c.ifMatch = `W/"2"`
		}
		if rec := tt.c.do(http.MethodPut, path, tt.body); rec.Code != tt.status {
			t.Errorf("PUT %s, %s = %d %s, want %d", path, tt.name, rec.Code, rec.Body, tt.status)
		}
	}

	if n := count("SELECT count(*) FROM practitioner_role WHERE id = $1 AND specialty_code = '394579002' AND version_id = 2", d.roleID); n != 1 {
		t.Errorf("the role holds cardiology at version 2 in %d rows, want 1", n)
	}
	for _, tt := range []struct {
		name, where string
		want        int
	}{
		{"the change", "action = 'U' AND outcome = 0 AND entity_reference = 'PractitionerRole/' || $1", 1},
		{"the stale change", "action = 'U' AND outcome = 4 AND entity_reference = 'PractitionerRole/' || $1", 1},
		{"the practitioner's and the patient's refusals", "action = 'U' AND outcome = 4 AND entity_reference IS NULL AND $1 <> ''", 2},
	} {
		if n := count("SELECT count(*) FROM audit_event WHERE subtype_code = 'update' AND "+tt.where, d.roleID); n != tt.want {
			t.Errorf("%s: %d audit events, want %d", tt.name, n, tt.want)
		}
	}
	if strings.Contains(log.String(), "394579002") {
		t.Errorf("the log holds the specialty: %s", log.String())
	}
}

func TestNew_AnyoneListsTheSpecialties(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, slog.New(slog.NewTextHandler(&syncBuffer{}, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)

	rec := (&client{t: t, h: app.Handler}).do(http.MethodGet, "/specialties", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"system":"http://snomed.info/sct","code":"394579002","display":"Cardiology","text":"Cardiology"}`) {
		t.Errorf("GET /specialties signed out = %d %s, want 200 with the list", rec.Code, rec.Body)
	}
}
