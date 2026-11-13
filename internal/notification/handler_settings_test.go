package notification_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// stubSettings answers as told and keeps what it was given.
type stubSettings struct {
	err      error
	from, to string
	in       notification.Preferences
}

func (s *stubSettings) Preferences(context.Context, notification.Caller) (notification.Preferences, error) {
	return notification.DefaultPreferences(), s.err
}

func (s *stubSettings) SetPreferences(_ context.Context, _ notification.Caller, in notification.Preferences) (notification.Preferences, error) {
	s.in = in
	return in, s.err
}

func (s *stubSettings) SMSUsage(_ context.Context, _ notification.Caller, from, to string) ([]notification.UsageMonth, error) {
	s.from, s.to = from, to
	return []notification.UsageMonth{{Month: "2026-10", Messages: 3, Segments: 4}}, s.err
}

func serveSettings(t *testing.T, svc *stubSettings, signedIn bool, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	notification.NewSettingsHandler(svc, func(context.Context) (notification.Caller, bool, error) {
		return notification.Caller{AccountID: patientAccount}, signedIn, nil
	}, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestSettingsHandler_ReadsAndReplacesThePatientsSettings(t *testing.T) {
	rec := serveSettings(t, &stubSettings{}, true, http.MethodGet, "/me/notification-settings", "")
	want := `{"quiet_hours":{"start":"22:00","end":"08:00"},"channels":{"push":true,"email":true,"sms":false}}`
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	svc := &stubSettings{}
	rec = serveSettings(t, svc, true, http.MethodPut, "/me/notification-settings", `{"channels":{"push":false,"email":true,"sms":true},"quiet_hours":null,"time_zone":"Asia/Dubai"}`)
	if rec.Code != 200 || svc.in.Quiet != nil || svc.in.TimeZone != "Asia/Dubai" || !svc.in.Channels.SMS || svc.in.Channels.Push {
		t.Errorf("PUT = %d %s with %+v", rec.Code, rec.Body, svc.in)
	}
	for _, tt := range []struct {
		err              error
		name, body, want string
		status           int
		signedIn         bool
	}{
		{name: "signed out", body: `{}`, status: 401, want: "unauthenticated"},
		{name: "no channels", body: `{"time_zone":"Asia/Dubai"}`, status: 400, want: "invalid_request", signedIn: true},
		{name: "a channel missing", body: `{"channels":{"push":true,"email":true}}`, status: 400, want: "invalid_request", signedIn: true},
		{name: "an unknown field", body: `{"channels":{"push":true,"email":true,"sms":false},"x":1}`, status: 400, want: "invalid_request", signedIn: true},
		{name: "no phone", body: `{"channels":{"push":true,"email":true,"sms":true}}`, err: notification.ErrPhoneRequired, status: 409, want: "phone_required", signedIn: true},
		{name: "staff", body: `{"channels":{"push":true,"email":true,"sms":false}}`, err: notification.ErrForbidden, status: 403, want: "forbidden", signedIn: true},
		{name: "a zone", body: `{"channels":{"push":true,"email":true,"sms":false},"time_zone":"Mars"}`, err: notification.ErrInvalid, status: 400, want: "invalid_request", signedIn: true},
	} {
		rec := serveSettings(t, &stubSettings{err: tt.err}, tt.signedIn, http.MethodPut, "/me/notification-settings", tt.body)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), `"`+tt.want+`"`) {
			t.Errorf("%s: %d %s, want %d %s", tt.name, rec.Code, rec.Body, tt.status, tt.want)
		}
	}
}

func TestSettingsHandler_GivesTheAdminTheSMSUsage(t *testing.T) {
	svc := &stubSettings{}
	rec := serveSettings(t, svc, true, http.MethodGet, "/clinic/sms-usage?from=2026-01&to=2026-12", "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"months":[{"month":"2026-10","messages":3,"segments":4}]}` || svc.from != "2026-01" || svc.to != "2026-12" {
		t.Errorf("GET = %d %s (%s to %s)", rec.Code, rec.Body, svc.from, svc.to)
	}
	for _, tt := range []struct {
		err    error
		status int
	}{{notification.ErrForbidden, 403}, {notification.ErrInvalid, 400}} {
		if rec := serveSettings(t, &stubSettings{err: tt.err}, true, http.MethodGet, "/clinic/sms-usage?from=x&to=y", ""); rec.Code != tt.status {
			t.Errorf("%v: %d, want %d", tt.err, rec.Code, tt.status)
		}
	}
}
