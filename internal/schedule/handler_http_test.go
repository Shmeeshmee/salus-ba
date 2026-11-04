package schedule_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// stubAvailability answers with what it is given and keeps what it was asked.
type stubAvailability struct {
	err     error
	caller  schedule.Caller
	role    string
	called  string
	in      schedule.Availability
	out     schedule.Schedule
	version int32
}

func (s *stubAvailability) Availability(_ context.Context, c schedule.Caller, roleID string) (schedule.Schedule, error) {
	s.called, s.caller, s.role = "Availability", c, roleID
	return s.out, s.err
}

func (s *stubAvailability) SaveAvailability(_ context.Context, c schedule.Caller, roleID string, in schedule.Availability, version int32) (schedule.Schedule, error) {
	s.called, s.caller, s.role, s.in, s.version = "SaveAvailability", c, roleID, in, version
	return s.out, s.err
}

// serve sends method on role A1's availability with body and If-Match ifMatch ("" for none) as
// practitionerA1, or as nobody when signedIn is false, logging to log.
func serve(t *testing.T, svc *stubAvailability, signedIn bool, log *bytes.Buffer, method, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := schedule.NewHTTPHandler(svc, func(context.Context) (schedule.Caller, bool, error) { return practitionerA1, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, "/practitioner-roles/"+roleA1+"/availability", strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const mondays = `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00"}]}`

var savedA1 = schedule.Schedule{ID: "a5000000-0000-4000-8000-0000000000a1", OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 30,
	VersionID: 2, Rules: []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00"}}, LastUpdated: now}

func TestHTTPHandler_GivesTheAvailabilityWithItsVersion(t *testing.T) {
	svc := &stubAvailability{out: savedA1}
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodGet, "", "")

	want := `{"id":"` + savedA1.ID + `","practitionerRoleId":"` + roleA1 + `","availability":[{"start":"09:00","end":"12:00","day":1}],"slotMinutes":30,"versionId":2}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("ETag") != `W/"2"` {
		t.Errorf("GET = %d %s ETag %q, want 200 %s ETag W/\"2\"", rec.Code, rec.Body, rec.Header().Get("ETag"), want)
	}
	if svc.called != "Availability" || svc.caller != practitionerA1 || svc.role != roleA1 {
		t.Errorf("%s(%+v, %s), want Availability for the caller and the URL's role", svc.called, svc.caller, svc.role)
	}
}

func TestHTTPHandler_GivesAnEmptyAvailabilityBeforeTheFirstEdit(t *testing.T) {
	rec := serve(t, &stubAvailability{out: schedule.Schedule{OrganizationID: clinicA, PractitionerRoleID: roleA1}}, true, &bytes.Buffer{}, http.MethodGet, "", "")

	want := `{"practitionerRoleId":"` + roleA1 + `","availability":[],"versionId":0}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("ETag") != `W/"0"` {
		t.Errorf("GET = %d %s ETag %q, want 200 %s ETag W/\"0\"", rec.Code, rec.Body, rec.Header().Get("ETag"), want)
	}
}

func TestHTTPHandler_SavesTheAvailabilityAtTheVersionOfIfMatch(t *testing.T) {
	svc := &stubAvailability{out: savedA1}
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, `W/"1"`, mondays)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"versionId":2`) || rec.Header().Get("ETag") != `W/"2"` {
		t.Errorf("PUT = %d %s ETag %q, want 200 at version 2", rec.Code, rec.Body, rec.Header().Get("ETag"))
	}
	want := schedule.Availability{SlotMinutes: 30, Rules: []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00"}}}
	if svc.called != "SaveAvailability" || svc.caller != practitionerA1 || svc.role != roleA1 || svc.version != 1 ||
		svc.in.SlotMinutes != want.SlotMinutes || !slices.Equal(svc.in.Rules, want.Rules) {
		t.Errorf("%s(%+v, %s, %+v, %d), want SaveAvailability of %+v at version 1", svc.called, svc.caller, svc.role, svc.in, svc.version, want)
	}
	if rec := serve(t, &stubAvailability{out: savedA1}, true, &bytes.Buffer{}, http.MethodPut, `W/"0"`, `{"slotMinutes":30,"availability":[]}`); rec.Code != http.StatusOK {
		t.Errorf("PUT of no rules at version 0 = %d %s, want 200", rec.Code, rec.Body)
	}
}

func TestHTTPHandler_RefusesAnEditItCannotReadBeforeCallingTheService(t *testing.T) {
	tests := []struct {
		name, ifMatch, body string
	}{
		{name: "no If-Match", body: mondays},
		{name: "an If-Match that is not a version", ifMatch: `W/"one"`, body: mondays},
		{name: "no body", ifMatch: `W/"1"`},
		{name: "no slot length", ifMatch: `W/"1"`, body: `{"availability":[]}`},
		{name: "no availability", ifMatch: `W/"1"`, body: `{"slotMinutes":30}`},
		{name: "an unknown field", ifMatch: `W/"1"`, body: `{"slotMinutes":30,"availability":[],"timeZone":"UTC"}`},
		{name: "an unknown rule field", ifMatch: `W/"1"`, body: `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"10:00","note":"x"}]}`},
		{name: "a day that is not a number", ifMatch: `W/"1"`, body: `{"slotMinutes":30,"availability":[{"day":"Monday","start":"09:00","end":"10:00"}]}`},
		{name: "two objects", ifMatch: `W/"1"`, body: mondays + mondays},
		{name: "a body over 64 KiB", ifMatch: `W/"1"`, body: `{"slotMinutes":30,"availability":[` + strings.Repeat(`{"day":1,"start":"09:00","end":"10:00"},`, 2000) + `{}]}`},
	}
	for _, tt := range tests {
		svc := &stubAvailability{}
		rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, tt.ifMatch, tt.body)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"invalid_request"}`+"\n" || svc.called != "" {
			t.Errorf("%s: PUT = %d %s (service called %q), want 400 invalid_request", tt.name, rec.Code, rec.Body, svc.called)
		}
	}
}

func TestHTTPHandler_MapsTheServiceErrors(t *testing.T) {
	tests := []struct {
		err    error
		code   string
		status int
	}{
		{schedule.ErrInvalidSlotMinutes, "invalid_slot_minutes", http.StatusBadRequest},
		{schedule.ErrInvalidAvailability, "invalid_availability", http.StatusBadRequest},
		{schedule.ErrForbidden, "forbidden", http.StatusForbidden},
		{schedule.ErrNotFound, "not_found", http.StatusNotFound},
		{schedule.ErrConflict, "version_conflict", http.StatusPreconditionFailed},
	}
	for _, tt := range tests {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			rec := serve(t, &stubAvailability{err: tt.err}, true, &bytes.Buffer{}, method, `W/"1"`, mondays)
			if rec.Code != tt.status || rec.Body.String() != `{"error":"`+tt.code+`"}`+"\n" {
				t.Errorf("%s with %v = %d %s, want %d %s", method, tt.err, rec.Code, rec.Body, tt.status, tt.code)
			}
		}
	}
}

func TestHTTPHandler_AsksForASignInWithoutCallingTheService(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		svc := &stubAvailability{}
		rec := serve(t, svc, false, &bytes.Buffer{}, method, `W/"1"`, mondays)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":"unauthenticated"}`+"\n" || svc.called != "" {
			t.Errorf("%s signed out = %d %s (service called %q), want 401 unauthenticated", method, rec.Code, rec.Body, svc.called)
		}
	}
}

func TestHTTPHandler_SendsAFixedErrorAndLogsNoAvailability(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		var logs bytes.Buffer
		rec := serve(t, &stubAvailability{err: errors.New("synthetic failure")}, true, &logs, method, `W/"1"`, mondays)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "synthetic") {
			t.Errorf("%s with a failing service = %d %s, want 500 with a fixed message", method, rec.Code, rec.Body)
		}
		if logs.Len() == 0 || strings.Contains(logs.String(), "09:00") {
			t.Errorf("log = %q, want the error logged without the availability", logs.String())
		}
	}

	h := schedule.NewHTTPHandler(&stubAvailability{}, func(context.Context) (schedule.Caller, bool, error) {
		return schedule.Caller{}, false, errors.New("synthetic caller failure")
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/practitioner-roles/"+roleA1+"/availability", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GET when the caller cannot be resolved = %d, want 500", rec.Code)
	}
}
