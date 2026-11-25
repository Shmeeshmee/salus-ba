package location_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/location"
)

// stubRooms answers every call with its values and remembers what it was asked.
type stubRooms struct {
	err      error
	rooms    []location.Room
	location string
	staff    location.Staff
	saved    location.Room
	device   bool
}

func (s *stubRooms) Rooms(_ context.Context, st location.Staff, id string) ([]location.Room, error) {
	s.staff, s.location = st, id
	return s.rooms, s.err
}

func (s *stubRooms) Devices(_ context.Context, st location.Staff, id string) ([]location.Device, error) {
	s.staff, s.location, s.device = st, id, true
	out := make([]location.Device, len(s.rooms))
	for i, r := range s.rooms {
		out[i] = location.Device(r)
	}
	return out, s.err
}

func (s *stubRooms) SaveRoom(_ context.Context, st location.Staff, r location.Room) (location.Room, error) {
	s.staff, s.saved = st, r
	r.VersionID++
	if r.ID == "" {
		r.ID, r.VersionID = "a4000000-0000-4000-8000-000000000001", 1
	}
	return r, s.err
}

func (s *stubRooms) SaveDevice(_ context.Context, st location.Staff, d location.Device) (location.Device, error) {
	out, err := s.SaveRoom(context.Background(), st, location.Room(d))
	s.device = true
	return location.Device(out), err
}

func serveRooms(t *testing.T, svc *stubRooms, signedIn bool, method, path, body string, header ...string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (location.Staff, bool, error) { return roomAdmin, signedIn, nil }
	location.NewRoomHandler(svc, caller, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestRoomHandler_ListsAddsAndChanges(t *testing.T) {
	t.Parallel()
	svc := &stubRooms{rooms: []location.Room{{ID: "a4000000-0000-4000-8000-000000000001", LocationID: siteA, Name: "Room 1", VersionID: 3, Active: true}}}
	w := serveRooms(t, svc, true, http.MethodGet, "/clinic/locations/"+siteA+"/rooms", "")
	want := `[{"id":"a4000000-0000-4000-8000-000000000001","locationId":"` + siteA + `","name":"Room 1","versionId":3,"active":true}]`
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != want || w.Header().Get("Cache-Control") != "no-store" || svc.location != siteA || svc.staff != roomAdmin {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	w = serveRooms(t, svc, true, http.MethodPost, "/clinic/locations/"+siteA+"/devices", `{"name":"Ultrasound","active":true}`)
	if w.Code != http.StatusCreated || w.Header().Get("ETag") != `W/"1"` || !svc.device || svc.saved.LocationID != siteA || svc.saved.Name != "Ultrasound" || !svc.saved.Active {
		t.Fatalf("add: %d %s %+v", w.Code, w.Body, svc.saved)
	}
	w = serveRooms(t, svc, true, http.MethodPut, "/clinic/rooms/a4000000-0000-4000-8000-000000000001", `{"name":"Room 2","active":false}`, "If-Match", `W/"3"`)
	if w.Code != http.StatusOK || w.Header().Get("ETag") != `W/"4"` || svc.saved.ID != "a4000000-0000-4000-8000-000000000001" || svc.saved.VersionID != 3 {
		t.Fatalf("change: %d %s %+v", w.Code, w.Body, svc.saved)
	}
}

func TestRoomHandler_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err                 error
		name, method, path  string
		body, ifMatch, code string
		status              int
		signedOut           bool
	}{
		{name: "signed out", signedOut: true, method: http.MethodGet, path: "/clinic/locations/" + siteA + "/rooms", status: 401, code: "unauthenticated"},
		{name: "not found", err: location.ErrNotFound, method: http.MethodGet, path: "/clinic/locations/" + siteA + "/devices", status: 404, code: "not_found"},
		{name: "forbidden", err: location.ErrForbidden, method: http.MethodPost, path: "/clinic/locations/" + siteA + "/rooms", body: `{"name":"x","active":true}`, status: 403, code: "forbidden"},
		{name: "invalid", err: location.ErrInvalid, method: http.MethodPost, path: "/clinic/locations/" + siteA + "/rooms", body: `{"name":" ","active":true}`, status: 400, code: "invalid_location"},
		{name: "conflict", err: location.ErrVersionConflict, method: http.MethodPut, path: "/clinic/devices/" + siteA, body: `{"name":"x","active":true}`, ifMatch: `W/"1"`, status: 412, code: "version_conflict"},
		{name: "no If-Match", method: http.MethodPut, path: "/clinic/rooms/" + siteA, body: `{"name":"x","active":true}`, status: 400, code: "invalid_request"},
		{name: "a field missing", method: http.MethodPost, path: "/clinic/locations/" + siteA + "/rooms", body: `{"name":"x"}`, status: 400, code: "invalid_request"},
		{name: "an unknown field", method: http.MethodPost, path: "/clinic/locations/" + siteA + "/rooms", body: `{"name":"x","active":true,"locationId":"y"}`, status: 400, code: "invalid_request"},
		{name: "broken", err: errBroken, method: http.MethodGet, path: "/clinic/locations/" + siteA + "/rooms", status: 500, code: "internal error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var header []string
			if tc.ifMatch != "" {
				header = []string{"If-Match", tc.ifMatch}
			}
			w := serveRooms(t, &stubRooms{err: tc.err}, !tc.signedOut, tc.method, tc.path, tc.body, header...)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), `"`+tc.code+`"`) {
				t.Errorf("%d %s, want %d %s", w.Code, w.Body, tc.status, tc.code)
			}
			if errors.Is(tc.err, errBroken) && strings.Contains(w.Body.String(), "broken") {
				t.Error("the internal error reached the client")
			}
		})
	}
}
