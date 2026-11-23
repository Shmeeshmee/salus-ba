package signup_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/signup"
)

type stubService struct {
	err     error
	saved   signup.Setup
	setup   signup.Setup
	started signup.Request
	ip      string
	token   string
}

func (s *stubService) Start(_ context.Context, r signup.Request, ip string) error {
	s.started, s.ip = r, ip
	return s.err
}

func (s *stubService) Verify(_ context.Context, token, _ string) (signup.Signup, error) {
	s.token = token
	return signup.Signup{}, s.err
}

func (s *stubService) Setup(context.Context, signup.Caller) (signup.Setup, error) {
	return s.setup, s.err
}

func (s *stubService) SaveSetup(_ context.Context, _ signup.Caller, in signup.Setup) (signup.Setup, error) {
	s.saved = in
	return in, s.err
}

func serve(t *testing.T, svc *stubService, signedIn bool, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (signup.Caller, bool, error) {
		return signup.Caller{AccountID: "a1", OrganizationID: newClinic, Admin: true}, signedIn, nil
	}
	mux := http.NewServeMux()
	signup.NewHandler(svc, caller, slog.New(slog.NewTextHandler(io.Discard, nil)), nil).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandler_SignupAndVerification(t *testing.T) {
	for _, tc := range []struct {
		err        error
		name, path string
		body, want string
		status     int
	}{
		{name: "accepted", path: "/signup", body: `{"email":"admin@example.test","clinicName":"Synthetic clinic","language":"en"}`, status: http.StatusAccepted},
		{name: "invalid", path: "/signup", body: `{"email":"admin","clinicName":"Synthetic clinic","language":"en"}`, err: signup.ErrInvalid, status: http.StatusBadRequest, want: "invalid_signup"},
		{name: "unknown field", path: "/signup", body: `{"email":"a@example.test","clinicName":"x","language":"en","x":1}`, status: http.StatusBadRequest, want: "invalid_request"},
		{name: "limited", path: "/signup", body: `{"email":"admin@example.test","clinicName":"Synthetic clinic","language":"en"}`, err: &signup.RateLimitedError{RetryAfter: time.Minute}, status: http.StatusTooManyRequests, want: "rate_limited"},
		{name: "verified", path: "/signup/verify", body: `{"token":"t"}`, status: http.StatusNoContent},
		{name: "bad token", path: "/signup/verify", body: `{"token":"t"}`, err: signup.ErrInvalidToken, status: http.StatusBadRequest, want: "invalid_verification"},
		{name: "no body", path: "/signup/verify", body: `{`, status: http.StatusBadRequest, want: "invalid_request"},
	} {
		svc := &stubService{err: tc.err}
		rec := serve(t, svc, false, http.MethodPost, tc.path, tc.body)
		if rec.Code != tc.status || (tc.want != "" && !strings.Contains(rec.Body.String(), `"`+tc.want+`"`)) {
			t.Errorf("%s: POST %s = %d %s, want %d %s", tc.name, tc.path, rec.Code, rec.Body, tc.status, tc.want)
		}
	}
	svc := &stubService{}
	serve(t, svc, false, http.MethodPost, "/signup", `{"email":"admin@example.test","clinicName":"Synthetic clinic","language":"en"}`)
	if svc.started != (signup.Request{Email: "admin@example.test", ClinicName: "Synthetic clinic", Language: "en"}) || svc.ip != "192.0.2.1" {
		t.Errorf("Start(%+v, %q)", svc.started, svc.ip)
	}
}

func TestHandler_Setup(t *testing.T) {
	open := signup.Setup{Steps: map[signup.Step]signup.StepState{
		signup.StepDetails: signup.StepDone, signup.StepBilling: signup.StepOpen, signup.StepHours: signup.StepOpen,
		signup.StepPractitioner: signup.StepSkipped, signup.StepServices: signup.StepOpen,
	}}
	svc := &stubService{setup: open}
	rec := serve(t, svc, true, http.MethodGet, "/clinic/setup", "")
	want := `{"details":"done","billing":"open","hours":"open","practitioner":"skipped","services":"open"}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET /clinic/setup = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	rec = serve(t, svc, true, http.MethodPut, "/clinic/setup", want)
	if rec.Code != http.StatusOK || svc.saved.Steps[signup.StepPractitioner] != signup.StepSkipped || len(svc.saved.Steps) != 5 {
		t.Errorf("PUT /clinic/setup = %d %s, saved %+v", rec.Code, rec.Body, svc.saved)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["details"] != "done" {
		t.Errorf("PUT /clinic/setup body %s", rec.Body)
	}
	for _, tc := range []struct {
		err    error
		name   string
		method string
		body   string
		status int
		signIn bool
	}{
		{name: "signed out", method: http.MethodGet, status: http.StatusUnauthorized},
		{name: "no clinic", method: http.MethodGet, err: signup.ErrNotFound, status: http.StatusNotFound, signIn: true},
		{name: "not the admin", method: http.MethodPut, body: want, err: signup.ErrForbidden, status: http.StatusForbidden, signIn: true},
		{name: "missing step", method: http.MethodPut, body: `{"details":"done"}`, status: http.StatusBadRequest, signIn: true},
		{name: "bad state", method: http.MethodPut, body: strings.Replace(want, `"done"`, `"half"`, 1), err: signup.ErrInvalidSetup, status: http.StatusBadRequest, signIn: true},
	} {
		rec := serve(t, &stubService{err: tc.err}, tc.signIn, tc.method, "/clinic/setup", tc.body)
		if rec.Code != tc.status {
			t.Errorf("%s: %s /clinic/setup = %d %s, want %d", tc.name, tc.method, rec.Code, rec.Body, tc.status)
		}
	}
}
