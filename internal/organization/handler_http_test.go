package organization_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// stubChat answers with what it is given and keeps what it was asked.
type stubChat struct {
	err      error
	caller   organization.Caller
	id       string
	in       organization.ChatSettings
	settings organization.ChatSettings
	hours    organization.ChatHours
}

func (s *stubChat) Settings(_ context.Context, c organization.Caller) (organization.ChatSettings, error) {
	s.caller = c
	return s.settings, s.err
}

func (s *stubChat) SetSettings(_ context.Context, c organization.Caller, in organization.ChatSettings) (organization.ChatSettings, error) {
	s.caller, s.in = c, in
	return in, s.err
}

func (s *stubChat) Hours(_ context.Context, c organization.Caller, id string) (organization.ChatHours, error) {
	s.caller, s.id = c, id
	return s.hours, s.err
}

var adminCaller = organization.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}

func serve(t *testing.T, svc *stubChat, signedIn bool, log *bytes.Buffer, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := organization.NewHTTPHandler(svc, func(context.Context) (organization.Caller, bool, error) { return adminCaller, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestHTTPHandler_ServesTheChatSettingsAndHours(t *testing.T) {
	svc := &stubChat{
		settings: organization.ChatSettings{Hours: []organization.ChatHour{{Day: "mon", Start: "08:00", End: "16:00"}}, ResponseTimeHours: 8, EmailAfterMinutes: 30},
		hours:    organization.ChatHours{TimeZone: "Europe/Sarajevo", Hours: []organization.ChatHour{}, ResponseTimeHours: 24, OpenNow: true},
	}
	rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodGet, "/clinic/chat-settings", "")
	if want := `{"hours":[{"day":"mon","start":"08:00","end":"16:00"}],"response_time_hours":8,"email_after_minutes":30}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET settings = %d %s", rec.Code, rec.Body)
	}
	rec = serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, "/clinic/chat-settings", `{"hours":[{"day":"tue","start":"09:00","end":"17:00"}],"response_time_hours":4,"email_after_minutes":15}`)
	if rec.Code != http.StatusOK || svc.in.ResponseTimeHours != 4 || svc.in.EmailAfterMinutes != 15 || svc.in.Hours[0].Day != "tue" || svc.caller != adminCaller {
		t.Errorf("PUT settings = %d %s, service got %+v", rec.Code, rec.Body, svc.in)
	}
	rec = serve(t, svc, true, &bytes.Buffer{}, http.MethodGet, "/organizations/"+clinicA+"/chat-hours", "")
	if want := `{"time_zone":"Europe/Sarajevo","hours":[],"response_time_hours":24,"open_now":true}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want || svc.id != clinicA {
		t.Errorf("GET hours = %d %s", rec.Code, rec.Body)
	}
}

func TestHTTPHandler_MapsTheChatErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{organization.ErrNotFound, "not_found", http.StatusNotFound},
		{organization.ErrForbidden, "forbidden", http.StatusForbidden},
		{organization.ErrInvalidChatSettings, "invalid_request", http.StatusBadRequest},
		{errors.New("synthetic database failure"), "internal error", http.StatusInternalServerError},
	} {
		for _, r := range []struct{ method, path, body string }{
			{http.MethodGet, "/clinic/chat-settings", ""},
			{http.MethodPut, "/clinic/chat-settings", `{"hours":[],"response_time_hours":4,"email_after_minutes":15}`},
			{http.MethodGet, "/organizations/" + clinicA + "/chat-hours", ""},
		} {
			rec := serve(t, &stubChat{err: tc.err}, true, &bytes.Buffer{}, r.method, r.path, r.body)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), `"`+tc.code+`"`) {
				t.Errorf("%s %s with %v = %d %s", r.method, r.path, tc.err, rec.Code, rec.Body)
			}
		}
	}
	for _, body := range []string{`{`, `{"hours":[],"response_time_hours":4,"email_after_minutes":15,"extra":1}`, `{"hours":[]}`} {
		svc := &stubChat{}
		if rec := serve(t, svc, true, &bytes.Buffer{}, http.MethodPut, "/clinic/chat-settings", body); rec.Code != http.StatusBadRequest || svc.caller != (organization.Caller{}) {
			t.Errorf("PUT %s = %d, want 400 before the service", body, rec.Code)
		}
	}
	if rec := serve(t, &stubChat{}, false, &bytes.Buffer{}, http.MethodGet, "/clinic/chat-settings", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out = %d, want 401", rec.Code)
	}
}
