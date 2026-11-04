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

	"github.com/Shmeeshmee/salus/internal/auth"
)

// stubInvites is the invite part of stubService.
type stubInvites struct {
	invite     auth.Invite // returned by Invite
	acceptance auth.Acceptance
	gotInvite  auth.Invite
	acceptWith [2]string     // token, password
	invites    []auth.Invite // returned by ListInvites
}

func (s *stubService) Invite(_ context.Context, sess auth.Session, in auth.Invite) (auth.Invite, error) {
	s.gotSession, s.gotInvite = sess, in
	return s.invite, s.err
}

func (s *stubService) ListInvites(_ context.Context, sess auth.Session) ([]auth.Invite, error) {
	s.gotSession = sess
	return s.invites, s.err
}

func (s *stubService) AcceptInvite(_ context.Context, token, password string) (auth.Acceptance, error) {
	s.acceptWith = [2]string{token, password}
	return s.acceptance, s.err
}

func openInvite() auth.Invite {
	return auth.Invite{
		ID: "1a000000-0000-4000-8000-000000000001", OrganizationID: clinicID, InvitedBy: accountID, Email: "cy@example.test",
		Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example", CreatedAt: now, ExpiresAt: expires,
	}
}

const inviteJSON = `{"expiresAt":"2026-11-04T21:30:00Z","id":"1a000000-0000-4000-8000-000000000001","email":"cy@example.test","role":"practitioner","givenName":"Cy","familyName":"Example"}`

func TestInviteHandler_InvitesAsTheSignedInAdmin(t *testing.T) {
	svc := &stubService{stubInvites: stubInvites{invite: openInvite()}}
	r := signedIn(post(t, "/invites", `{"email":"cy@example.test","role":"practitioner","givenName":"Cy","familyName":"Example"}`), adminSession())

	rec := serve(t, svc, discard(), r)

	if rec.Code != http.StatusCreated || strings.TrimSpace(rec.Body.String()) != inviteJSON {
		t.Errorf("POST /invites = %d %s, want 201 %s", rec.Code, rec.Body, inviteJSON)
	}
	want := auth.Invite{Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example"}
	if svc.gotInvite != want || svc.gotSession != adminSession() {
		t.Errorf("Invite() called with %+v as %+v", svc.gotInvite, svc.gotSession)
	}
}

func TestListInvitesHandler_ListsTheOpenInvites(t *testing.T) {
	svc := &stubService{stubInvites: stubInvites{invites: []auth.Invite{openInvite()}}}
	r := signedIn(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/invites", nil), adminSession())

	rec := serve(t, svc, discard(), r)

	if want := `{"invites":[` + inviteJSON + `]}`; rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("GET /invites = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	svc.invites = nil
	if rec := serve(t, svc, discard(), r); strings.TrimSpace(rec.Body.String()) != `{"invites":[]}` {
		t.Errorf("GET /invites with none = %s, want an empty list", rec.Body)
	}
}

func TestInviteHandlers_NeedASession(t *testing.T) {
	for _, r := range []*http.Request{
		post(t, "/invites", `{"email":"cy@example.test","role":"admin"}`),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/invites", nil),
	} {
		if rec := serve(t, &stubService{}, discard(), r); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session = %d, want 401", r.Method, r.URL.Path, rec.Code)
		}
	}
}

func TestAcceptInviteHandler_ReturnsTheEnrolmentAndTheChallenge(t *testing.T) {
	svc := &stubService{stubInvites: stubInvites{acceptance: auth.Acceptance{
		Account: auth.Account{ID: accountID}, Challenge: challenge, Enrolment: auth.Enrolment{Secret: "SECRET", URI: "otpauth://totp/x"},
	}}}

	rec := serve(t, svc, discard(), post(t, "/invites/accept", `{"token":"`+token+`","password":"`+password+`"}`))

	want := `{"challenge":"` + challenge + `","secret":"SECRET","uri":"otpauth://totp/x"}`
	if rec.Code != http.StatusCreated || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("POST /invites/accept = %d %s (Cache-Control %q), want 201 %s, not cached", rec.Code, rec.Body, rec.Header().Get("Cache-Control"), want)
	}
	if svc.acceptWith != [2]string{token, password} {
		t.Errorf("AcceptInvite() called with %v", svc.acceptWith)
	}
}

func TestInviteHandlers_MapErrorsToFixedCodes(t *testing.T) {
	tests := []struct {
		err    error
		code   string
		status int
	}{
		{auth.ErrForbidden, "forbidden", http.StatusForbidden},
		{auth.ErrInvalidEmail, "invalid_email", http.StatusBadRequest},
		{auth.ErrInvalidRole, "invalid_role", http.StatusBadRequest},
		{auth.ErrInvalidName, "invalid_name", http.StatusBadRequest},
		{auth.ErrInvalidPassword, "invalid_password", http.StatusBadRequest},
		{auth.ErrInvalidInvite, "invalid_invite", http.StatusNotFound},
		{auth.ErrEmailTaken, "email_taken", http.StatusConflict},
	}
	for _, tt := range tests {
		for _, r := range []*http.Request{
			signedIn(post(t, "/invites", `{"email":"cy@example.test","role":"admin"}`), adminSession()),
			post(t, "/invites/accept", `{"token":"`+token+`","password":"`+password+`"}`),
		} {
			rec := serve(t, &stubService{err: tt.err}, discard(), r)
			if want := `{"error":"` + tt.code + `"}`; rec.Code != tt.status || strings.TrimSpace(rec.Body.String()) != want {
				t.Errorf("%s with %v = %d %s, want %d %s", r.URL.Path, tt.err, rec.Code, rec.Body, tt.status, want)
			}
		}
	}
	if rec := serve(t, &stubService{}, discard(), post(t, "/invites/accept", `{"token":1}`)); rec.Code != http.StatusBadRequest {
		t.Errorf("POST /invites/accept with a bad body = %d, want 400", rec.Code)
	}
}

func TestInviteHandlers_LogNoEmailNameTokenOrPassword(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	leaky := fmt.Errorf("auth: invite: %w", errors.New("db down"))
	for _, svc := range []*stubService{{stubInvites: stubInvites{invite: openInvite(), invites: []auth.Invite{openInvite()}}}, {err: auth.ErrForbidden}, {err: leaky}} {
		serve(t, svc, log, signedIn(post(t, "/invites", `{"email":"cy@example.test","role":"practitioner","givenName":"Cy","familyName":"Example"}`), adminSession()))
		serve(t, svc, log, signedIn(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/invites", nil), adminSession()))
		serve(t, svc, log, post(t, "/invites/accept", `{"token":"`+token+`","password":"`+password+`"}`))
	}
	if logs.Len() == 0 {
		t.Fatal("nothing logged: the internal errors should be")
	}
	for _, secret := range []string{"example.test", "Example", token, password} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("the log holds %q:\n%s", secret, logs.String())
		}
	}
}
