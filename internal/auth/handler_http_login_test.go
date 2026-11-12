package auth_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

const (
	token     = "c2Vzc2lvbiB0b2tlbiBvZiAzMiBzeW50aGV0aWMgYnl0ZXM" // synthetic
	challenge = "Y2hhbGxlbmdlIG9mIDMyIHN5bnRoZXRpYyBieXRlcyEhIQ"  // synthetic
)

var expires = time.Date(2026, 11, 4, 21, 30, 0, 0, time.UTC)

func post(t *testing.T, path, body string) *http.Request {
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
}

// fromWeb marks r as the web app's: it sends X-Salus-CSRF on every write.
func fromWeb(r *http.Request) *http.Request {
	r.Header.Set("X-Salus-CSRF", "1")
	return r
}

func signedIn(r *http.Request, s auth.Session) *http.Request {
	return r.WithContext(auth.WithSession(r.Context(), s))
}

func doneResult() auth.LoginResult {
	return auth.LoginResult{Next: auth.NextDone, Token: token, Session: auth.Session{ID: sessionID, AccountID: accountID, ExpiresAt: expires}}
}

func TestLoginHandler_GivesTheMobileAppTheTokenInTheBody(t *testing.T) {
	svc := &stubService{result: doneResult()}
	r := post(t, "/auth/login", `{"email":"Ana@example.test","password":"`+password+`"}`)
	r.RemoteAddr = "192.0.2.10:51000"

	rec := serve(t, svc, discard(), r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rec.Code, rec.Body)
	}
	if svc.email != "Ana@example.test" || svc.password != password || svc.ip != "192.0.2.10" {
		t.Errorf("service called with %q, %q from %q", svc.email, svc.password, svc.ip)
	}
	want := `{"expiresAt":"2026-11-04T21:30:00Z","next":"done","token":"` + token + `"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("the mobile app got a cookie")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
}

func TestLoginHandler_GivesTheWebAppAStrictHttpOnlyCookieAndNoToken(t *testing.T) {
	svc := &stubService{result: doneResult()}

	rec := serve(t, svc, discard(), fromWeb(post(t, "/auth/login", `{"email":"ana@example.test","password":"x"}`)))

	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), token) {
		t.Fatalf("= %d %s, want 200 without the token", rec.Code, rec.Body)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("%d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != "salus_session" || c.Value != token || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || !c.Expires.Equal(expires) {
		t.Errorf("cookie %+v, want salus_session=<token>; Path=/; Expires; HttpOnly; Secure; SameSite=Strict", c)
	}
	if want := `{"expiresAt":"2026-11-04T21:30:00Z","next":"done"}` + "\n"; rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
}

func TestLoginHandler_ReturnsTheChallengeForTheNextStep(t *testing.T) {
	for _, next := range []auth.Next{auth.NextVerify, auth.NextEnrol} {
		svc := &stubService{result: auth.LoginResult{Next: next, Challenge: challenge}}
		rec := serve(t, svc, discard(), fromWeb(post(t, "/auth/login", `{"email":"ben@example.test","password":"x"}`)))
		want := `{"next":"` + string(next) + `","challenge":"` + challenge + `"}` + "\n"
		if rec.Code != http.StatusOK || rec.Body.String() != want || len(rec.Result().Cookies()) != 0 {
			t.Errorf("next %s: = %d %s with %d cookies, want %s and no cookie", next, rec.Code, rec.Body, len(rec.Result().Cookies()), want)
		}
	}
}

func TestLoginHandler_AnswersTooManyAttemptsWithRetryAfter(t *testing.T) {
	for wait, want := range map[time.Duration]string{14*time.Minute + 59500*time.Millisecond: "900", 200 * time.Millisecond: "1", 0: "1"} {
		svc := &stubService{err: fmt.Errorf("auth: login: %w", &auth.RateLimitedError{RetryAfter: wait})}
		rec := serve(t, svc, discard(), post(t, "/auth/login", `{"email":"ana@example.test","password":"x"}`))
		if rec.Code != http.StatusTooManyRequests || rec.Body.String() != `{"error":"rate_limited"}`+"\n" || rec.Header().Get("Retry-After") != want {
			t.Errorf("wait %v: = %d %s Retry-After %q, want 429 rate_limited %s", wait, rec.Code, rec.Body, rec.Header().Get("Retry-After"), want)
		}
	}
}

func TestEnrolHandler_PassesTheChallengeOrTheSessionAndReturnsTheSecret(t *testing.T) {
	svc := &stubService{enrolment: auth.Enrolment{Secret: "SYNTHETICSECRET", URI: "otpauth://totp/Salus:ben@example.test?secret=SYNTHETICSECRET"}}
	rec := serve(t, svc, discard(), post(t, "/auth/mfa/enrol", `{"challenge":"`+challenge+`"}`))
	want := `{"secret":"SYNTHETICSECRET","uri":"otpauth://totp/Salus:ben@example.test?secret=SYNTHETICSECRET"}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || svc.challenge != challenge {
		t.Errorf("= %d %s for challenge %q, want %s", rec.Code, rec.Body, svc.challenge, want)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("the secret may be cached")
	}

	sess := auth.Session{ID: sessionID, AccountID: accountID}
	svc = &stubService{}
	serve(t, svc, discard(), signedIn(post(t, "/auth/mfa/enrol", `{}`), sess))
	if svc.challenge != "" || svc.gotSession != sess {
		t.Errorf("enrol from a session: service got challenge %q, session %+v", svc.challenge, svc.gotSession)
	}
}

func TestConfirmHandler_OpensTheSessionFromAChallengeAndAnswers204FromASession(t *testing.T) {
	svc := &stubService{result: doneResult()}
	rec := serve(t, svc, discard(), post(t, "/auth/mfa/confirm", `{"challenge":"`+challenge+`","code":"123456"}`))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), token) || svc.code != "123456" || svc.challenge != challenge {
		t.Errorf("from a challenge: = %d %s, service got %q, %q", rec.Code, rec.Body, svc.challenge, svc.code)
	}

	sess := auth.Session{ID: sessionID, AccountID: accountID}
	svc = &stubService{}
	rec = serve(t, svc, discard(), signedIn(post(t, "/auth/mfa/confirm", `{"code":"123456"}`), sess))
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || svc.gotSession != sess {
		t.Errorf("from a session: = %d %q, service got %+v; want 204", rec.Code, rec.Body, svc.gotSession)
	}
}

func TestVerifyHandler_OpensTheSession(t *testing.T) {
	svc := &stubService{result: doneResult()}
	rec := serve(t, svc, discard(), fromWeb(post(t, "/auth/mfa/verify", `{"challenge":"`+challenge+`","code":"123456"}`)))
	if rec.Code != http.StatusOK || len(rec.Result().Cookies()) != 1 || svc.challenge != challenge || svc.code != "123456" {
		t.Errorf("= %d %s with %d cookies, service got %q, %q", rec.Code, rec.Body, len(rec.Result().Cookies()), svc.challenge, svc.code)
	}
}

func TestLogoutHandler_EndsTheSessionAndClearsTheCookie(t *testing.T) {
	sess := auth.Session{ID: sessionID, AccountID: accountID}
	svc := &stubService{}

	rec := serve(t, svc, discard(), signedIn(post(t, "/auth/logout", ""), sess))

	if rec.Code != http.StatusNoContent || svc.gotSession != sess {
		t.Fatalf("= %d %s, service got %+v; want 204 for the session", rec.Code, rec.Body, svc.gotSession)
	}
	c := rec.Result().Cookies()
	if len(c) != 1 || c[0].Name != "salus_session" || c[0].Value != "" || c[0].MaxAge >= 0 {
		t.Errorf("cookies %+v, want salus_session cleared", c)
	}
}

func TestLoginHandlers_MapErrorsToFixedCodes(t *testing.T) {
	sess := auth.Session{ID: sessionID, AccountID: accountID}
	tests := []struct {
		err     error
		request func(t *testing.T) *http.Request
		name    string
		body    string
		status  int
	}{
		{name: "login, not JSON", request: func(t *testing.T) *http.Request { return post(t, "/auth/login", `{`) }, status: 400, body: "invalid_request"},
		{name: "login, wrong credentials", err: auth.ErrWrongCredentials, request: func(t *testing.T) *http.Request { return post(t, "/auth/login", `{}`) }, status: 401, body: "wrong_credentials"},
		{name: "login fails", err: errors.New("db down"), request: func(t *testing.T) *http.Request { return post(t, "/auth/login", `{}`) }, status: 500, body: "internal error"},
		{name: "enrol, not JSON", request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/enrol", ``) }, status: 400, body: "invalid_request"},
		{name: "enrol, no session", err: auth.ErrUnauthenticated, request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/enrol", `{}`) }, status: 401, body: "unauthenticated"},
		{name: "enrol, enrolled", err: auth.ErrAlreadyEnrolled, request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/enrol", `{}`) }, status: 409, body: "already_enrolled"},
		{name: "confirm, not JSON", request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/confirm", `[]`) }, status: 400, body: "invalid_request"},
		{name: "confirm, not enrolled", err: auth.ErrNotEnrolled, request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/confirm", `{}`) }, status: 409, body: "not_enrolled"},
		{name: "verify, not JSON", request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/verify", `{"x":1}`) }, status: 400, body: "invalid_request"},
		{name: "verify, wrong code", err: auth.ErrWrongCode, request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/verify", `{}`) }, status: 401, body: "wrong_code"},
		{name: "verify, invalid challenge", err: auth.ErrInvalidChallenge, request: func(t *testing.T) *http.Request { return post(t, "/auth/mfa/verify", `{}`) }, status: 401, body: "invalid_challenge"},
		{name: "logout, nobody signed in", request: func(t *testing.T) *http.Request { return post(t, "/auth/logout", "") }, status: 401, body: "unauthenticated"},
		{name: "logout, session gone", err: auth.ErrUnauthenticated, request: func(t *testing.T) *http.Request { return signedIn(post(t, "/auth/logout", ""), sess) }, status: 401, body: "unauthenticated"},
		{name: "logout fails", err: errors.New("db down"), request: func(t *testing.T) *http.Request { return signedIn(post(t, "/auth/logout", ""), sess) }, status: 500, body: "internal error"},
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

// sessionServe runs r through the session middleware in front of a handler that reports the session it
// sees.
func sessionServe(t *testing.T, svc *stubService, r *http.Request) (*httptest.ResponseRecorder, *auth.Session) {
	t.Helper()
	var seen *auth.Session
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := auth.SessionFrom(r.Context()); ok {
			seen = &s
		}
		w.WriteHeader(http.StatusTeapot)
	})
	rec := httptest.NewRecorder()
	auth.NewHandler(svc, discard(), nil).Session(next).ServeHTTP(rec, r)
	return rec, seen
}

func withCookie(r *http.Request) *http.Request {
	r.AddCookie(&http.Cookie{Name: "salus_session", Value: token})
	return r
}

func TestSession_AuthenticatesABearerTokenOrTheCookie(t *testing.T) {
	sess := auth.Session{ID: sessionID, AccountID: accountID, ExpiresAt: expires}
	bearer := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil)
	bearer.Header.Set("Authorization", "Bearer "+token)
	cookieGet := withCookie(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil))
	cookiePost := fromWeb(withCookie(post(t, "/auth/logout", "")))
	bearerPost := post(t, "/auth/logout", "")
	bearerPost.Header.Set("Authorization", "Bearer "+token)

	for name, r := range map[string]*http.Request{"bearer": bearer, "cookie": cookieGet, "cookie with the CSRF header": cookiePost, "bearer write": bearerPost} {
		svc := &stubService{session: sess}
		rec, seen := sessionServe(t, svc, r)
		if rec.Code != http.StatusTeapot || seen == nil || *seen != sess || svc.token != token {
			t.Errorf("%s: = %d, session %v, token %q; want the session passed on", name, rec.Code, seen, svc.token)
		}
	}
}

func TestSession_PassesRequestsWithoutAValidSessionOnAnonymously(t *testing.T) {
	none := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", nil)
	stale := withCookie(fromWeb(post(t, "/auth/login", "")))
	for name, tt := range map[string]struct {
		r         *http.Request
		wantCalls int
	}{"no token": {none, 0}, "a revoked or expired token": {stale, 1}} {
		svc := &stubService{authErr: auth.ErrUnauthenticated}
		rec, seen := sessionServe(t, svc, tt.r)
		if rec.Code != http.StatusTeapot || seen != nil || svc.authenticated != tt.wantCalls {
			t.Errorf("%s: = %d, session %v, %d checks; want passed on with no session", name, rec.Code, seen, svc.authenticated)
		}
	}
}

func TestSession_RejectsACookieWriteWithoutTheCSRFHeader(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		svc := &stubService{}
		r := withCookie(httptest.NewRequestWithContext(t.Context(), method, "/auth/logout", nil))
		rec, _ := sessionServe(t, svc, r)
		if rec.Code != http.StatusForbidden || rec.Body.String() != `{"error":"csrf"}`+"\n" || svc.authenticated != 0 {
			t.Errorf("%s with the cookie and no X-Salus-CSRF: = %d %s, want 403 csrf before the session check", method, rec.Code, rec.Body)
		}
	}
}

func TestSession_AnswersAFailedCheckWith500(t *testing.T) {
	var logs bytes.Buffer
	svc := &stubService{authErr: errors.New("db down")}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	auth.NewHandler(svc, slog.New(slog.NewJSONHandler(&logs, nil)), nil).Session(http.NotFoundHandler()).ServeHTTP(rec, r)
	if rec.Code != http.StatusInternalServerError || strings.Contains(logs.String(), token) {
		t.Errorf("= %d, log %s; want 500 and no token in the log", rec.Code, logs.String())
	}
}

func TestLoginHandler_TakesTheClientIPFromTrustedProxiesOnly(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	tests := []struct {
		name, remote, forwarded, want string
	}{
		{name: "direct", remote: "192.0.2.10:5000", want: "192.0.2.10"},
		{name: "direct, header ignored", remote: "192.0.2.10:5000", forwarded: "198.51.100.1", want: "192.0.2.10"},
		{name: "through a trusted proxy", remote: "10.0.0.2:5000", forwarded: "198.51.100.1", want: "198.51.100.1"},
		{name: "through two trusted proxies", remote: "10.0.0.2:5000", forwarded: "203.0.113.9, 198.51.100.1, 10.0.0.3", want: "198.51.100.1"},
		{name: "trusted proxy, no header", remote: "10.0.0.2:5000", want: "10.0.0.2"},
		{name: "trusted proxy, malformed header", remote: "10.0.0.2:5000", forwarded: "not-an-ip", want: "10.0.0.2"},
		{name: "IPv6", remote: "[2001:db8::1]:5000", want: "2001:db8::1"},
		{name: "IPv4-mapped IPv6 behind an IPv6 proxy", remote: "[fd00::2]:5000", forwarded: "::ffff:198.51.100.1", want: "198.51.100.1"},
		{name: "unparsable remote", remote: "pipe", want: "pipe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &stubService{result: doneResult()}
			r := post(t, "/auth/login", `{"email":"ana@example.test","password":"x"}`)
			r.RemoteAddr = tt.remote
			if tt.forwarded != "" {
				r.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			mux := http.NewServeMux()
			auth.NewHandler(svc, discard(), trusted).Routes(mux)
			mux.ServeHTTP(httptest.NewRecorder(), r)
			if svc.ip != tt.want {
				t.Errorf("client IP %q, want %q", svc.ip, tt.want)
			}
		})
	}
}

func TestLoginHandlers_LogNoEmailPasswordTokenChallengeOrCode(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	leaky := fmt.Errorf("auth: login: %w", errors.New("db down"))
	sess := auth.Session{ID: sessionID, AccountID: accountID}
	for _, svc := range []*stubService{{result: doneResult()}, {err: auth.ErrWrongCode}, {err: leaky}} {
		serve(t, svc, log, post(t, "/auth/login", `{"email":"ana@example.test","password":"`+password+`"}`))
		serve(t, svc, log, post(t, "/auth/mfa/enrol", `{"challenge":"`+challenge+`"}`))
		serve(t, svc, log, post(t, "/auth/mfa/confirm", `{"challenge":"`+challenge+`","code":"123456"}`))
		serve(t, svc, log, post(t, "/auth/mfa/verify", `{"challenge":"`+challenge+`","code":"123456"}`))
		serve(t, svc, log, signedIn(post(t, "/auth/logout", ""), sess))
	}
	if logs.Len() == 0 {
		t.Fatal("nothing logged: the internal errors should be")
	}
	for _, secret := range []string{"example.test", password, token, challenge, "123456"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("the log holds %q:\n%s", secret, logs.String())
		}
	}
}

// Active is the check a long-lived socket repeats (docs/plans/chat.md): the request's session is still open, as
// Authenticate sees it now, from the bearer token or the cookie the request came with.
func TestActive_ReportsWhetherTheRequestsSessionIsStillOpen(t *testing.T) {
	for _, tt := range []struct {
		authErr error
		name    string
		header  string
		want    bool
		wantErr bool
	}{
		{name: "an open session", header: "Bearer tok", want: true},
		{name: "a revoked or expired one", header: "Bearer tok", authErr: auth.ErrUnauthenticated},
		{name: "no credentials", header: ""},
		{name: "a failure", header: "Bearer tok", authErr: errors.New("down"), wantErr: true},
	} {
		svc := &stubService{authErr: tt.authErr, session: auth.Session{ID: sessionID}}
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/conversations/x/events", nil)
		if tt.header != "" {
			r.Header.Set("Authorization", tt.header)
		}
		got, err := auth.NewHandler(svc, discard(), nil).Active(r)
		if got != tt.want || (err != nil) != tt.wantErr {
			t.Errorf("%s: Active = %v, %v", tt.name, got, err)
		}
		if tt.header != "" && svc.token != "tok" {
			t.Errorf("%s: authenticated %q, want the request's token", tt.name, svc.token)
		}
	}
}
