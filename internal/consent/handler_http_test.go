package consent_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
)

// stubConsents answers with what it is given and keeps what it was asked.
type stubConsents struct {
	err    error
	caller consent.Caller
	arg    string // the clinic granted, or the consent revoked
	out    []consent.Consent
	calls  []string
}

func (s *stubConsents) Grant(_ context.Context, c consent.Caller, organizationID string) (consent.Consent, error) {
	s.caller, s.arg = c, organizationID
	s.calls = append(s.calls, "Grant")
	return s.first(), s.err
}

func (s *stubConsents) Revoke(_ context.Context, c consent.Caller, id string) (consent.Consent, error) {
	s.caller, s.arg = c, id
	s.calls = append(s.calls, "Revoke")
	return s.first(), s.err
}

func (s *stubConsents) Mine(_ context.Context, c consent.Caller) ([]consent.Consent, error) {
	s.caller = c
	s.calls = append(s.calls, "Mine")
	return s.out, s.err
}

func (s *stubConsents) first() consent.Consent {
	if len(s.out) == 0 {
		return consent.Consent{}
	}
	return s.out[0]
}

var (
	activeJSON  = `{"dateTime":"2026-11-01T08:00:00Z","id":"` + consentAA + `","organizationId":"` + clinicA + `","status":"active"}`
	revokedJSON = `{"dateTime":"2026-11-01T08:00:00Z","revokedAt":"2026-11-02T08:00:00Z","id":"` + consentAB + `","organizationId":"` + clinicB + `","status":"inactive"}`
)

// serve runs one request through the handler as caller (nobody when signedIn is false), logging to log.
func serve(t *testing.T, svc *stubConsents, caller consent.Caller, signedIn bool, log *bytes.Buffer, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serveWith(t, svc, func(context.Context) (consent.Caller, bool, error) { return caller, signedIn, nil }, log, method, path, body)
}

func serveWith(t *testing.T, svc *stubConsents, caller func(context.Context) (consent.Caller, bool, error), log *bytes.Buffer, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := consent.NewHandler(svc, caller, slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestHandler_Grants(t *testing.T) {
	svc := &stubConsents{out: []consent.Consent{activeAA}}
	rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodPost, "/consents", `{"organizationId":"`+clinicA+`"}`)
	if rec.Code != http.StatusCreated || rec.Body.String() != activeJSON+"\n" || svc.caller != me || svc.arg != clinicA {
		t.Errorf("POST /consents = %d %s (caller %+v, clinic %q), want 201 %s", rec.Code, rec.Body, svc.caller, svc.arg, activeJSON)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
}

func TestHandler_Revokes(t *testing.T) {
	svc := &stubConsents{out: []consent.Consent{revokedAB}}
	rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodPost, "/consents/"+consentAB+"/revoke", "")
	if rec.Code != http.StatusOK || rec.Body.String() != revokedJSON+"\n" || svc.arg != consentAB {
		t.Errorf("POST revoke = %d %s (consent %q), want 200 %s", rec.Code, rec.Body, svc.arg, revokedJSON)
	}
}

func TestHandler_ListsMine(t *testing.T) {
	for _, tt := range []struct {
		want string
		out  []consent.Consent
	}{
		{out: []consent.Consent{activeAA, revokedAB}, want: `{"consents":[` + activeJSON + `,` + revokedJSON + `]}`},
		{out: []consent.Consent{}, want: `{"consents":[]}`},
	} {
		svc := &stubConsents{out: tt.out}
		rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodGet, "/me/consents", "")
		if rec.Code != http.StatusOK || rec.Body.String() != tt.want+"\n" {
			t.Errorf("GET /me/consents = %d %s, want 200 %s", rec.Code, rec.Body, tt.want)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control %q, want no-store", got)
		}
	}
}

func TestHandler_MapsEveryErrorToItsCode(t *testing.T) {
	tests := []struct {
		err    error
		method string
		path   string
		body   string
		code   string
		status int
	}{
		{err: consent.ErrForbidden, method: http.MethodGet, path: "/me/consents", status: 403, code: "forbidden"},
		{err: consent.ErrActive, method: http.MethodPost, path: "/consents", body: `{"organizationId":"x"}`, status: 409, code: "consent_active"},
		{err: consent.ErrProfileRequired, method: http.MethodPost, path: "/consents", body: `{"organizationId":"x"}`, status: 409, code: "profile_required"},
		{err: consent.ErrClinicNotFound, method: http.MethodPost, path: "/consents", body: `{"organizationId":"x"}`, status: 404, code: "not_found"},
		{err: consent.ErrNotFound, method: http.MethodPost, path: "/consents/x/revoke", status: 404, code: "not_found"},
		{err: consent.ErrInactive, method: http.MethodPost, path: "/consents/x/revoke", status: 409, code: "consent_inactive"},
	}
	for _, tt := range tests {
		rec := serve(t, &stubConsents{err: tt.err}, me, true, &bytes.Buffer{}, tt.method, tt.path, tt.body)
		if want := `{"error":"` + tt.code + `"}` + "\n"; rec.Code != tt.status || rec.Body.String() != want {
			t.Errorf("%s %s with %v = %d %s, want %d %s", tt.method, tt.path, tt.err, rec.Code, rec.Body, tt.status, want)
		}
	}
}

func TestHandler_RefusesBadRequestsAndNobodyBeforeTheService(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		code     string
		status   int
		signedIn bool
	}{
		{name: "nobody grants", method: http.MethodPost, path: "/consents", body: `{"organizationId":"x"}`, status: 401, code: "unauthenticated"},
		{name: "nobody revokes", method: http.MethodPost, path: "/consents/x/revoke", status: 401, code: "unauthenticated"},
		{name: "nobody lists", method: http.MethodGet, path: "/me/consents", status: 401, code: "unauthenticated"},
		{name: "no clinic", method: http.MethodPost, path: "/consents", body: `{}`, status: 400, code: "invalid_request", signedIn: true},
		{name: "an unknown field", method: http.MethodPost, path: "/consents", body: `{"organizationId":"x","scope":"all"}`, status: 400, code: "invalid_request", signedIn: true},
		{name: "not JSON", method: http.MethodPost, path: "/consents", body: `organizationId=x`, status: 400, code: "invalid_request", signedIn: true},
		{name: "two objects", method: http.MethodPost, path: "/consents", body: `{"organizationId":"x"}{}`, status: 400, code: "invalid_request", signedIn: true},
	}
	for _, tt := range tests {
		svc := &stubConsents{}
		rec := serve(t, svc, me, tt.signedIn, &bytes.Buffer{}, tt.method, tt.path, tt.body)
		if want := `{"error":"` + tt.code + `"}` + "\n"; rec.Code != tt.status || rec.Body.String() != want || len(svc.calls) != 0 {
			t.Errorf("%s: %d %s after calls %v, want %d %s before the service", tt.name, rec.Code, rec.Body, svc.calls, tt.status, want)
		}
	}
}

func TestHandler_LogsAnInternalErrorWithoutIDsOrDetail(t *testing.T) {
	var log bytes.Buffer
	boom := errors.New("connection reset")
	rec := serve(t, &stubConsents{err: boom}, me, true, &log, http.MethodPost, "/consents", `{"organizationId":"`+clinicA+`"}`)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "connection reset") {
		t.Errorf("POST = %d %s, want 500 without the error text", rec.Code, rec.Body)
	}
	if !strings.Contains(log.String(), "connection reset") || strings.Contains(log.String(), accountA) {
		t.Errorf("log %q, want the error and not the account", log.String())
	}
}

func TestHandler_FailsWhenTheCallerCannotBeResolved(t *testing.T) {
	var log bytes.Buffer
	svc := &stubConsents{}
	broken := func(context.Context) (consent.Caller, bool, error) {
		return consent.Caller{}, false, errors.New("connection reset")
	}
	for _, path := range []string{"/consents", "/consents/x/revoke"} {
		rec := serveWith(t, svc, broken, &log, http.MethodPost, path, `{"organizationId":"x"}`)
		if rec.Code != http.StatusInternalServerError || len(svc.calls) != 0 {
			t.Errorf("POST %s = %d after calls %v, want 500 before the service", path, rec.Code, svc.calls)
		}
	}
	if rec := serveWith(t, svc, broken, &log, http.MethodGet, "/me/consents", ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /me/consents = %d, want 500", rec.Code)
	}
}
