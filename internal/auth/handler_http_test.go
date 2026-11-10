package auth_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

// stubService answers with what it is given and keeps what it was called with.
type stubService struct {
	err                    error
	authErr                error // returned by Authenticate
	account                auth.Account
	result                 auth.LoginResult
	enrolment              auth.Enrolment
	email, password, meFor string
	ip, challenge, code    string
	token                  string       // passed to Authenticate
	session                auth.Session // returned by Authenticate
	gotSession             auth.Session // passed to Enrol, Confirm or Logout
	languages              *[2]string   // chosen and detected, passed to Register, Login or AcceptInvite
	setLanguage            *string      // passed to SetLanguage
	stubInvites
	authenticated int // Authenticate calls
}

func (s *stubService) Login(_ context.Context, email, password, clientIP, language, detected string) (auth.LoginResult, error) {
	s.email, s.password, s.ip, s.languages = email, password, clientIP, &[2]string{language, detected}
	return s.result, s.err
}

func (s *stubService) Enrol(_ context.Context, challenge string, sess auth.Session) (auth.Enrolment, error) {
	s.challenge, s.gotSession = challenge, sess
	return s.enrolment, s.err
}

func (s *stubService) Confirm(_ context.Context, challenge string, sess auth.Session, code string) (auth.LoginResult, error) {
	s.challenge, s.gotSession, s.code = challenge, sess, code
	return s.result, s.err
}

func (s *stubService) Verify(_ context.Context, challenge, code string) (auth.LoginResult, error) {
	s.challenge, s.code = challenge, code
	return s.result, s.err
}

func (s *stubService) Logout(_ context.Context, sess auth.Session) error {
	s.gotSession = sess
	return s.err
}

func (s *stubService) Authenticate(_ context.Context, token string) (auth.Session, error) {
	s.token = token
	s.authenticated++
	return s.session, s.authErr
}

func (s *stubService) Register(_ context.Context, email, password, language, detected string) (auth.Account, error) {
	s.email, s.password, s.languages = email, password, &[2]string{language, detected}
	return s.account, s.err
}

func (s *stubService) Me(_ context.Context, id string) (auth.Account, error) {
	s.meFor = id
	return s.account, s.err
}

func (s *stubService) SetLanguage(_ context.Context, id, language string) (auth.Account, error) {
	s.meFor, s.setLanguage = id, &language
	return s.account, s.err
}

func serve(t *testing.T, svc *stubService, log *slog.Logger, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	auth.NewHandler(svc, log, nil).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

func discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

func registerRequest(t *testing.T, body string) *http.Request {
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", strings.NewReader(body))
}

func meRequest(t *testing.T, accountID string) *http.Request {
	ctx := t.Context()
	if accountID != "" {
		ctx = auth.WithSession(ctx, auth.Session{ID: sessionID, AccountID: accountID})
	}
	return httptest.NewRequestWithContext(ctx, http.MethodGet, "/auth/me", nil)
}

func patientAccount() auth.Account {
	return auth.Account{ID: accountID, Email: "ana@example.test", Role: auth.RolePatient, PasswordHash: "$argon2id$secret"}
}

func TestRegisterHandler_CreatesTheAccount(t *testing.T) {
	svc := &stubService{account: patientAccount()}

	rec := serve(t, svc, discard(), registerRequest(t, `{"email":"Ana@example.test","password":"`+password+`"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d %s, want 201", rec.Code, rec.Body)
	}
	if svc.email != "Ana@example.test" || svc.password != password {
		t.Errorf("service called with %q, %q", svc.email, svc.password)
	}
	want := `{"id":"` + accountID + `","email":"ana@example.test","role":"patient","mfaEnrolled":false}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestHandlers_MapErrorsToFixedCodes(t *testing.T) {
	tests := []struct {
		err     error
		request func(t *testing.T) *http.Request
		name    string
		body    string
		status  int
	}{
		{name: "not JSON", request: func(t *testing.T) *http.Request { return registerRequest(t, `{`) }, status: 400, body: "invalid_request"},
		{name: "unknown field", request: func(t *testing.T) *http.Request {
			return registerRequest(t, `{"email":"a@example.test","password":"x","role":"admin"}`)
		}, status: 400, body: "invalid_request"},
		{name: "two documents", request: func(t *testing.T) *http.Request { return registerRequest(t, `{}{}`) }, status: 400, body: "invalid_request"},
		{name: "too large", request: func(t *testing.T) *http.Request {
			return registerRequest(t, `{"email":"a@example.test","password":"`+strings.Repeat("p", 8192)+`"}`)
		}, status: 400, body: "invalid_request"},
		{name: "invalid e-mail", err: auth.ErrInvalidEmail, request: func(t *testing.T) *http.Request { return registerRequest(t, `{}`) }, status: 400, body: "invalid_email"},
		{name: "invalid password", err: auth.ErrInvalidPassword, request: func(t *testing.T) *http.Request { return registerRequest(t, `{}`) }, status: 400, body: "invalid_password"},
		{name: "taken e-mail", err: auth.ErrEmailTaken, request: func(t *testing.T) *http.Request { return registerRequest(t, `{}`) }, status: 409, body: "email_taken"},
		{name: "register fails", err: errors.New("db down"), request: func(t *testing.T) *http.Request { return registerRequest(t, `{}`) }, status: 500, body: "internal error"},
		{name: "me, nobody signed in", request: func(t *testing.T) *http.Request { return meRequest(t, "") }, status: 401, body: "unauthenticated"},
		{name: "me, account gone", err: auth.ErrNotFound, request: func(t *testing.T) *http.Request { return meRequest(t, accountID) }, status: 401, body: "unauthenticated"},
		{name: "me fails", err: errors.New("db down"), request: func(t *testing.T) *http.Request { return meRequest(t, accountID) }, status: 500, body: "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, &stubService{err: tt.err}, discard(), tt.request(t))
			want := `{"error":"` + tt.body + `"}` + "\n"
			if rec.Code != tt.status || rec.Body.String() != want {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body, tt.status, want)
			}
		})
	}
}

func TestMeHandler_ReturnsTheSignedInAccount(t *testing.T) {
	a := patientAccount()
	a.Role, a.OrganizationID, a.MFAEnrolledAt = auth.RoleAdmin, clinicID, now
	svc := &stubService{account: a}

	rec := serve(t, svc, discard(), meRequest(t, accountID))

	if rec.Code != http.StatusOK || svc.meFor != accountID {
		t.Fatalf("status = %d %s for %q, want 200 for the context's account", rec.Code, rec.Body, svc.meFor)
	}
	want := `{"id":"` + accountID + `","email":"ana@example.test","role":"admin","organizationId":"` + clinicID + `","mfaEnrolled":true}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
}

func TestMeHandler_GivesAPractitionerTheirPractitionerID(t *testing.T) {
	const practitionerID = "a3000000-0000-4000-8000-0000000000a1"
	a := patientAccount()
	a.Role, a.OrganizationID, a.PractitionerID, a.MFAEnrolledAt = auth.RolePractitioner, clinicID, practitionerID, now
	rec := serve(t, &stubService{account: a}, discard(), meRequest(t, accountID))

	want := `{"id":"` + accountID + `","email":"ana@example.test","role":"practitioner","organizationId":"` + clinicID + `","practitionerId":"` + practitionerID + `","mfaEnrolled":true}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET /auth/me = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

// A reviewer of the term list (SAL-14) sees the permission, so the web app shows the review panel's writes.
func TestMeHandler_GivesAReviewerTheTermReviewerFlag(t *testing.T) {
	a := patientAccount()
	a.Role, a.OrganizationID, a.MFAEnrolledAt, a.TermReviewer = auth.RoleAdmin, clinicID, now, true
	rec := serve(t, &stubService{account: a}, discard(), meRequest(t, accountID))

	want := `{"id":"` + accountID + `","email":"ana@example.test","role":"admin","organizationId":"` + clinicID + `","mfaEnrolled":true,"termReviewer":true}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET /auth/me = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

func TestHandlers_LogNoEmailPasswordOrHash(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	leaky := fmt.Errorf("auth: register: %w", errors.New("db down"))
	body := `{"email":"ana@example.test","password":"` + password + `"}`
	for _, svc := range []*stubService{{account: patientAccount()}, {err: auth.ErrEmailTaken}, {err: leaky}} {
		serve(t, svc, log, registerRequest(t, body))
		serve(t, svc, log, meRequest(t, accountID))
	}
	if logs.Len() == 0 {
		t.Fatal("nothing logged: the internal errors should be")
	}
	for _, secret := range []string{"example.test", password, "argon2id"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("the log holds %q:\n%s", secret, logs.String())
		}
	}
}
