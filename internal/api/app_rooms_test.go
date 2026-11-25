package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// Rooms and devices through the real stack (docs/clinic-cms/WAVE6.md U3, U4): a practitioner reads the clinic's
// locations and rooms for their own working hours and changes none; a rule's room is a room of the rule's location;
// another practitioner's hours in a room refuse an overlap with resource_busy; a service takes only a device in use.
func TestNew_RoomsOfALocationHoldOnePractitionersHoursAndAServiceTakesADeviceInUse(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	created := func(c *client, path, body string) string {
		t.Helper()
		rec := c.do(http.MethodPost, path, body)
		var out struct{ ID string }
		decode(t, rec, &out)
		if rec.Code != http.StatusCreated || out.ID == "" {
			t.Fatalf("POST %s = %d %s, want 201", path, rec.Code, rec.Body)
		}
		return out.ID
	}

	// The practitioner reads the locations (00081 gave the clinic its first) and a location's rooms and devices.
	rec := d.practitioner.do(http.MethodGet, "/clinic/locations", "")
	var sites []struct{ ID string }
	decode(t, rec, &sites)
	if rec.Code != http.StatusOK || len(sites) != 1 {
		t.Fatalf("GET /clinic/locations as the practitioner = %d %s, want the clinic's one location", rec.Code, rec.Body)
	}
	first := sites[0].ID
	second := created(d.admin, "/clinic/locations", `{"name":"Synthetic branch","active":true}`)
	roomFirst := created(d.admin, "/clinic/locations/"+first+"/rooms", `{"name":"Room 1","active":true}`)
	roomSecond := created(d.admin, "/clinic/locations/"+second+"/rooms", `{"name":"Room 2","active":true}`)
	for _, path := range []string{"/clinic/locations/" + first + "/rooms", "/clinic/locations/" + first + "/devices"} {
		if rec := d.practitioner.do(http.MethodGet, path, ""); rec.Code != http.StatusOK {
			t.Errorf("GET %s as the practitioner = %d %s, want 200", path, rec.Code, rec.Body)
		}
	}
	if rec := d.practitioner.do(http.MethodPost, "/clinic/locations/"+first+"/rooms", `{"name":"Room 3","active":true}`); rec.Code != http.StatusForbidden {
		t.Errorf("POST a room as the practitioner = %d %s, want 403", rec.Code, rec.Body)
	}

	// A rule's room is a room of the rule's location.
	path := "/practitioner-roles/" + d.roleID + "/availability"
	rule := func(site, room string) string {
		return `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00","mode":"in-person","locationId":"` + site + `","roomId":"` + room + `"}]}`
	}
	d.practitioner.ifMatch = `W/"0"`
	if rec := d.practitioner.do(http.MethodPut, path, rule(second, roomFirst)); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_availability"`) {
		t.Errorf("PUT a rule at the branch in the first location's room = %d %s, want 400 invalid_availability", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodPut, path, rule(second, roomSecond)); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), roomSecond) {
		t.Fatalf("PUT a rule at the branch in its room = %d %s, want 200", rec.Code, rec.Body)
	}

	// A second practitioner's hours in the same room at an overlapping time are refused.
	if rec := d.admin.do(http.MethodPost, "/invites", `{"email":"eli@example.test","role":"practitioner","givenName":"Eli","familyName":"Example"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /invites = %d %s", rec.Code, rec.Body)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	colleague := &client{t: t, h: app.Handler}
	colleague.bearer = accept(t, colleague, issueToken(t, exec, "eli@example.test"))
	var otherRole string
	if err := owner.QueryRow(t.Context(), "SELECT id::text FROM practitioner_role WHERE id <> $1", d.roleID).Scan(&otherRole); err != nil {
		t.Fatal(err)
	}
	colleague.ifMatch = `W/"0"`
	other := "/practitioner-roles/" + otherRole + "/availability"
	if rec := colleague.do(http.MethodPut, other, rule(second, roomSecond)); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"resource_busy"`) {
		t.Errorf("PUT another practitioner's hours in a held room = %d %s, want 409 resource_busy", rec.Code, rec.Body)
	}

	// A service takes a device of the clinic in use, never one out of use.
	device := created(d.admin, "/clinic/locations/"+first+"/devices", `{"name":"Device 1","active":false}`)
	service := `{"name":"Synthetic scan","priceMinor":5000,"modes":["in-person"],"active":true,"translations":[],"deviceId":"` + device + `"}`
	if rec := d.admin.do(http.MethodPost, "/clinic/services", service); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_service"`) {
		t.Errorf("POST a service with a device out of use = %d %s, want 400 invalid_service", rec.Code, rec.Body)
	}
}
