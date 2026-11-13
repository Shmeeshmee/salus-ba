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

// stubRules answers with what it is given and keeps what it was asked.
type stubRules struct {
	err      error
	caller   organization.Caller
	in       organization.AppointmentSettings
	settings organization.AppointmentSettings
}

func (s *stubRules) Settings(_ context.Context, c organization.Caller) (organization.AppointmentSettings, error) {
	s.caller = c
	return s.settings, s.err
}

func (s *stubRules) SetSettings(_ context.Context, c organization.Caller, in organization.AppointmentSettings) (organization.AppointmentSettings, error) {
	s.caller, s.in = c, in
	out := in
	out.Version++
	return out, s.err
}

// serveRules serves one request; a PUT carries If-Match W/"4", as read at version 4.
func serveRules(t *testing.T, svc *stubRules, signedIn bool, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	header := http.Header{}
	if method == http.MethodPut {
		header.Set("If-Match", `W/"4"`)
	}
	return serveRulesWith(t, svc, signedIn, method, body, header)
}

func serveRulesWith(t *testing.T, svc *stubRules, signedIn bool, method, body string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	h := organization.NewAppointmentsHandler(svc, func(context.Context) (organization.Caller, bool, error) { return adminCaller, signedIn, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, "/clinic/appointment-settings", strings.NewReader(body))
	req.Header = header
	mux.ServeHTTP(rec, req)
	return rec
}

const customJSON = `{"reminder_offsets_minutes":[2880,120,15],"reschedule_notice_hours":48,"reschedule_max":1,"waitlist_hold_minutes":30,` +
	`"noshow_flag_threshold":3,"noshow_flag_window_days":365,"sms_reminders":true,"waitlist_enabled":true}`

func TestAppointmentsHandler_ServesTheSettings(t *testing.T) {
	svc := &stubRules{settings: customSet}
	if rec := serveRules(t, svc, true, http.MethodGet, ""); rec.Code != http.StatusOK || rec.Body.String() != customJSON+"\n" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s", rec.Code, rec.Body)
	}
	if rec := serveRules(t, &stubRules{}, true, http.MethodGet, ""); !strings.Contains(rec.Body.String(), `"reminder_offsets_minutes":[]`) {
		t.Errorf("GET of no offsets = %s, want an empty list", rec.Body)
	}
	if rec := serveRules(t, svc, true, http.MethodPut, customJSON); rec.Code != http.StatusOK || !same(svc.in, customSet) || svc.caller != adminCaller {
		t.Errorf("PUT = %d %s, asked %+v as %+v", rec.Code, rec.Body, svc.in, svc.caller)
	}
	for _, body := range []string{`{}`, `{"reminder_offsets_minutes":[60]}`, customJSON[:len(customJSON)-1] + `,"extra":1}`, `not json`, customJSON + `{}`} {
		if rec := serveRules(t, svc, true, http.MethodPut, body); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %s = %d, want 400", body, rec.Code)
		}
	}
	for _, tt := range []struct {
		err  error
		code int
	}{
		{organization.ErrNotFound, http.StatusNotFound}, {organization.ErrForbidden, http.StatusForbidden},
		{organization.ErrInvalidAppointmentSettings, http.StatusBadRequest}, {organization.ErrSMSUnavailable, http.StatusConflict},
		{organization.ErrSettingsChanged, http.StatusPreconditionFailed},
		{errors.New("down"), http.StatusInternalServerError},
	} {
		if rec := serveRules(t, &stubRules{err: tt.err}, true, http.MethodPut, customJSON); rec.Code != tt.code {
			t.Errorf("%v: %d, want %d", tt.err, rec.Code, tt.code)
		}
	}
	if rec := serveRules(t, svc, false, http.MethodGet, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d", rec.Code)
	}
}

// GET names the version read as a weak ETag; PUT needs If-Match with it, passes it to the service, answers with the
// new version, and 412 version_conflict when another save came first.
func TestAppointmentsHandler_VersionsTheSettings(t *testing.T) {
	stored := customSet
	stored.Version = 4
	if rec := serveRules(t, &stubRules{settings: stored}, true, http.MethodGet, ""); rec.Header().Get("ETag") != `W/"4"` {
		t.Errorf("GET ETag = %q, want W/\"4\"", rec.Header().Get("ETag"))
	}
	svc := &stubRules{}
	rec := serveRules(t, svc, true, http.MethodPut, customJSON)
	if rec.Code != http.StatusOK || svc.in.Version != 4 || rec.Header().Get("ETag") != `W/"5"` {
		t.Errorf("PUT = %d, asked version %d, ETag %q; want 200, 4, W/\"5\"", rec.Code, svc.in.Version, rec.Header().Get("ETag"))
	}
	for _, match := range []string{"", "4", `W/"x"`, `"-1"`} {
		h := http.Header{}
		if match != "" {
			h.Set("If-Match", match)
		}
		if rec := serveRulesWith(t, &stubRules{}, true, http.MethodPut, customJSON, h); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT with If-Match %q = %d, want 400", match, rec.Code)
		}
	}
	rec = serveRules(t, &stubRules{err: organization.ErrSettingsChanged}, true, http.MethodPut, customJSON)
	if rec.Code != http.StatusPreconditionFailed || !strings.Contains(rec.Body.String(), `"version_conflict"`) {
		t.Errorf("stale PUT = %d %s, want 412 version_conflict", rec.Code, rec.Body)
	}
}
