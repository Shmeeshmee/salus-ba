package notification_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// stubDevices answers as told and keeps what it was asked.
type stubDevices struct {
	err      error
	list     []notification.Endpoint
	calls    []string
	platform push.Platform
	token    string
	id       string
	in       notification.NewEndpoint
}

func (s *stubDevices) RegisterDevice(_ context.Context, _ notification.Caller, p push.Platform, token string) error {
	s.calls, s.platform, s.token = append(s.calls, "register"), p, token
	return s.err
}

func (s *stubDevices) UnregisterDevice(_ context.Context, _ notification.Caller, p push.Platform, token string) error {
	s.calls, s.platform, s.token = append(s.calls, "unregister"), p, token
	return s.err
}

func (s *stubDevices) Webhooks(context.Context, notification.Caller) ([]notification.Endpoint, error) {
	s.calls = append(s.calls, "list")
	return s.list, s.err
}

func (s *stubDevices) CreateWebhook(_ context.Context, _ notification.Caller, in notification.NewEndpoint) (notification.Endpoint, string, error) {
	s.calls, s.in = append(s.calls, "create"), in
	return notification.Endpoint{ID: hookID, URL: in.URL, Events: in.Events, CreatedAt: chatNow, IncludeClinical: in.IncludeClinical}, "whsec_synthetic", s.err
}

func (s *stubDevices) DeleteWebhook(_ context.Context, _ notification.Caller, id string) error {
	s.calls, s.id = append(s.calls, "delete"), id
	return s.err
}

func serveNotification(t *testing.T, svc *stubDevices, log *bytes.Buffer, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := notification.NewHTTPHandler(svc, func(context.Context) (notification.Caller, bool, error) { return clinicAdminCaller, true, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestHTTPHandler_RegistersAndUnregistersTheDevice(t *testing.T) {
	svc := &stubDevices{}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		rec := serveNotification(t, svc, &bytes.Buffer{}, method, "/me/devices", `{"platform":"ios","token":"synthetic-token"}`)
		if rec.Code != http.StatusNoContent || svc.platform != push.IOS || svc.token != "synthetic-token" {
			t.Errorf("%s = %d %s", method, rec.Code, rec.Body)
		}
	}
	if strings.Join(svc.calls, ",") != "register,unregister" {
		t.Errorf("calls %v", svc.calls)
	}
	for _, body := range []string{`{"platform":"ios"}`, `{"token":"x"}`, `{"platform":"ios","token":"x","extra":1}`, `[]`} {
		if rec := serveNotification(t, &stubDevices{}, &bytes.Buffer{}, http.MethodPut, "/me/devices", body); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %s = %d", body, rec.Code)
		}
	}
}

func TestHTTPHandler_CreatesListsAndDeletesWebhooksAndShowsTheSecretOnce(t *testing.T) {
	svc := &stubDevices{list: []notification.Endpoint{{ID: hookID, URL: "https://hooks.example.test", Events: []string{"message.created"}, CreatedAt: chatNow, SecretSealed: []byte("sealed")}}}
	rec := serveNotification(t, svc, &bytes.Buffer{}, http.MethodPost, "/clinic/webhooks", `{"url":"https://hooks.example.test","events":["message.created"]}`)
	want := `{"created_at":"2026-11-12T10:00:00Z","id":"` + hookID + `","url":"https://hooks.example.test","secret":"whsec_synthetic","events":["message.created"],"include_clinical":false}` + "\n"
	if rec.Code != http.StatusCreated || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "no-store" || svc.in.IncludeClinical {
		t.Errorf("POST = %d %s", rec.Code, rec.Body)
	}
	rec = serveNotification(t, svc, &bytes.Buffer{}, http.MethodGet, "/clinic/webhooks", "")
	want = `{"webhooks":[{"created_at":"2026-11-12T10:00:00Z","id":"` + hookID + `","url":"https://hooks.example.test","events":["message.created"],"include_clinical":false}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s", rec.Code, rec.Body)
	}
	if rec := serveNotification(t, &stubDevices{}, &bytes.Buffer{}, http.MethodGet, "/clinic/webhooks", ""); rec.Body.String() != `{"webhooks":[]}`+"\n" {
		t.Errorf("GET none = %s", rec.Body)
	}
	if rec := serveNotification(t, svc, &bytes.Buffer{}, http.MethodDelete, "/clinic/webhooks/"+hookID, ""); rec.Code != http.StatusNoContent || svc.id != hookID {
		t.Errorf("DELETE = %d", rec.Code)
	}
}

func TestHTTPHandler_MapsTheErrorsAndLogsNoTokenOrURL(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{notification.ErrInvalid, "invalid_request", http.StatusBadRequest},
		{notification.ErrForbidden, "forbidden", http.StatusForbidden},
		{notification.ErrNotFound, "not_found", http.StatusNotFound},
		{errors.New("notification: synthetic failure"), "internal error", http.StatusInternalServerError},
	} {
		var log bytes.Buffer
		for _, r := range []struct{ method, path, body string }{
			{http.MethodPut, "/me/devices", `{"platform":"ios","token":"synthetic-token"}`},
			{http.MethodPost, "/clinic/webhooks", `{"url":"https://hooks.example.test/secret-path","events":["message.created"]}`},
			{http.MethodGet, "/clinic/webhooks", ""},
			{http.MethodDelete, "/clinic/webhooks/" + hookID, ""},
		} {
			rec := serveNotification(t, &stubDevices{err: tc.err}, &log, r.method, r.path, r.body)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), `"`+tc.code+`"`) {
				t.Errorf("%s %s with %v = %d %s", r.method, r.path, tc.err, rec.Code, rec.Body)
			}
		}
		if strings.Contains(log.String(), "synthetic-token") || strings.Contains(log.String(), "secret-path") {
			t.Errorf("the log holds a token or a URL: %s", log.String())
		}
	}
	if rec := serveNotification(t, &stubDevices{}, &bytes.Buffer{}, http.MethodPost, "/clinic/webhooks", `{"events":["message.created"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("POST without url = %d", rec.Code)
	}
}
