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
	return in, s.err
}

func serveRules(t *testing.T, svc *stubRules, signedIn bool, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := organization.NewAppointmentsHandler(svc, func(context.Context) (organization.Caller, bool, error) { return adminCaller, signedIn, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/clinic/appointment-settings", strings.NewReader(body)))
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
		{organization.ErrInvalidAppointmentSettings, http.StatusBadRequest}, {errors.New("down"), http.StatusInternalServerError},
	} {
		if rec := serveRules(t, &stubRules{err: tt.err}, true, http.MethodPut, customJSON); rec.Code != tt.code {
			t.Errorf("%v: %d, want %d", tt.err, rec.Code, tt.code)
		}
	}
	if rec := serveRules(t, svc, false, http.MethodGet, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d", rec.Code)
	}
}
