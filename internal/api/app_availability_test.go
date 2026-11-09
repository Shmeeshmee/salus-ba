package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

func TestNew_ThePractitionerAndTheAdminEditARolesAvailabilityAuditedAndEnqueueItsSlots(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	path := "/practitioner-roles/" + d.roleID + "/availability"
	const mondays = `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00"}]}`

	rec := d.practitioner.do(http.MethodGet, path, "")
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"0"` || !strings.Contains(rec.Body.String(), `"availability":[]`) {
		t.Errorf("GET %s as the practitioner before any edit = %d %s, want 200 empty at version 0", path, rec.Code, rec.Body)
	}
	d.practitioner.ifMatch = `W/"0"`
	if rec := d.practitioner.do(http.MethodPut, path, mondays); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("PUT %s as the practitioner at version 0 = %d %s, want 200 at version 1", path, rec.Code, rec.Body)
	}
	d.admin.ifMatch = `W/"1"`
	if rec := d.admin.do(http.MethodPut, path, `{"slotMinutes":20,"availability":[]}`); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` {
		t.Errorf("PUT %s as the admin at version 1 = %d %s, want 200 at version 2", path, rec.Code, rec.Body)
	}
	for _, tt := range []struct {
		c                           *client
		name, method, ifMatch, body string
		status                      int
	}{
		{d.practitioner, "the stale version", http.MethodPut, `W/"1"`, mondays, http.StatusPreconditionFailed},
		{d.practitioner, "a length not a multiple of 5", http.MethodPut, `W/"2"`, `{"slotMinutes":32,"availability":[]}`, http.StatusBadRequest},
		{d.admin, "overlapping rules", http.MethodPut, `W/"2"`, `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00"},{"day":1,"start":"11:00","end":"13:00"}]}`, http.StatusBadRequest},
		{d.admin, "an end before the start", http.MethodPut, `W/"2"`, `{"slotMinutes":30,"availability":[{"day":1,"start":"12:00","end":"09:00"}]}`, http.StatusBadRequest},
		{d.patient, "a patient's read", http.MethodGet, "", "", http.StatusNotFound},
		{d.patient, "a patient's edit", http.MethodPut, `W/"2"`, mondays, http.StatusNotFound},
		{&client{t: t, h: app.Handler}, "nobody", http.MethodGet, "", "", http.StatusUnauthorized},
	} {
		tt.c.ifMatch = tt.ifMatch
		if rec := tt.c.do(tt.method, path, tt.body); rec.Code != tt.status {
			t.Errorf("%s %s, %s = %d %s, want %d", tt.method, path, tt.name, rec.Code, rec.Body, tt.status)
		}
	}

	if n := count("SELECT count(*) FROM schedule WHERE practitioner_role_id = $1 AND slot_minutes = 20 AND version_id = 2", d.roleID); n != 1 {
		t.Errorf("the role's schedule is at 20 minutes and version 2 in %d rows, want 1", n)
	}
	// Each edit enqueued the schedule's generation, with the two ids only.
	if n := count(`SELECT count(*) FROM river_job j JOIN schedule s ON j.args = jsonb_build_object('schedule_id', s.id::text, 'organization_id', s.organization_id::text)
		WHERE j.kind = 'slots_generate' AND s.practitioner_role_id = $1`, d.roleID); n != 2 {
		t.Errorf("%d slots_generate jobs for the schedule, want one per edit", n)
	}
	for _, tt := range []struct {
		name, where string
		want        int
	}{
		{"the reads", "subtype_code = 'read' AND entity_reference IS NULL AND outcome = 0", 1},
		{"the create", "subtype_code = 'create' AND entity_reference LIKE 'Schedule/%' AND outcome = 0", 1},
		{"the update", "subtype_code = 'update' AND entity_reference LIKE 'Schedule/%' AND outcome = 0", 1},
		{"the refused edits", "subtype_code = 'update' AND entity_reference IS NULL AND outcome = 4", 2},
		{"the refused read", "subtype_code = 'read' AND entity_reference IS NULL AND outcome = 4 AND agent_account_id IS NOT NULL AND organization_id IS NULL", 1},
	} {
		if n := count("SELECT count(*) FROM audit_event WHERE (entity_reference IS NULL OR entity_reference LIKE 'Schedule/%') AND "+tt.where+" AND $1 <> ''", d.roleID); n != tt.want {
			t.Errorf("%s: %d audit events, want %d", tt.name, n, tt.want)
		}
	}
	if strings.Contains(log.String(), "09:00") {
		t.Errorf("the log holds the availability: %s", log.String())
	}
}
