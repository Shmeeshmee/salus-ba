package api_test

import (
	"encoding/base32"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const testPassword = "a long enough secret"

// client sends requests to the app as one caller: the mobile app (bearer token) or the web app (cookie and
// X-Salus-CSRF).
type client struct {
	t       *testing.T
	h       http.Handler
	cookie  *http.Cookie
	bearer  string
	ifMatch string // sent as If-Match when set
	web     bool
}

func (c *client) do(method, path, body string) *httptest.ResponseRecorder {
	c.t.Helper()
	r := httptest.NewRequestWithContext(c.t.Context(), method, path, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.10:5000"
	if c.web {
		r.Header.Set("X-Salus-CSRF", "1")
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	if c.bearer != "" {
		r.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.ifMatch != "" {
		r.Header.Set("If-Match", c.ifMatch)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, r)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "salus_session" {
			c.cookie = ck
		}
	}
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("body %s: %v", rec.Body, err)
	}
}

func newApp(t *testing.T) (*api.App, config.Config) {
	t.Helper()
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	return app, cfg
}

func openDB(t *testing.T, cfg config.Config) func(query string, args ...any) int {
	t.Helper()
	db, err := postgres.Open(t.Context(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
}

func TestNew_SignsAMobilePatientInAndOut(t *testing.T) {
	app, cfg := newApp(t)
	c := &client{t: t, h: app.Handler}
	c.do(http.MethodPost, "/auth/register", `{"email":"ana@example.test","password":"`+testPassword+`"}`)

	if rec := c.do(http.MethodPost, "/auth/login", `{"email":"ana@example.test","password":"wrong password here"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("a wrong password = %d %s, want 401", rec.Code, rec.Body)
	}
	rec := c.do(http.MethodPost, "/auth/login", `{"email":"ana@example.test","password":"`+testPassword+`"}`)
	var login struct{ Next, Token string }
	decode(t, rec, &login)
	if rec.Code != http.StatusOK || login.Next != "done" || login.Token == "" {
		t.Fatalf("POST /auth/login = %d %s, want a session token", rec.Code, rec.Body)
	}
	c.bearer = login.Token
	if rec := c.do(http.MethodGet, "/auth/me", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ana@example.test") {
		t.Errorf("GET /auth/me with the token = %d %s, want the account", rec.Code, rec.Body)
	}
	if rec := c.do(http.MethodPost, "/auth/logout", ""); rec.Code != http.StatusNoContent {
		t.Errorf("POST /auth/logout = %d %s, want 204", rec.Code, rec.Body)
	}
	if rec := c.do(http.MethodGet, "/auth/me", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /auth/me after logout = %d %s, want 401 on the next request", rec.Code, rec.Body)
	}

	count := openDB(t, cfg)
	if n := count(`SELECT count(*) FROM session WHERE revoked_at IS NOT NULL AND NOT mfa_verified AND organization_id IS NULL`); n != 1 {
		t.Errorf("%d revoked patient sessions, want 1", n)
	}
	audited := count(`SELECT count(*) FROM audit_event WHERE type_code = '110114' AND subtype_code = '110122' AND outcome = 4
		AND entity_reference = 'Account/' || agent_account_id::text`)
	audited += 10 * count(`SELECT count(*) FROM audit_event WHERE type_code = '110114' AND subtype_code = '110122' AND outcome = 0`)
	audited += 100 * count(`SELECT count(*) FROM audit_event WHERE type_code = '110114' AND subtype_code = '110123' AND outcome = 0`)
	if audited != 111 {
		t.Errorf("audited (logouts, logins, failed logins) as %03d, want 111", audited)
	}

	rdb := redisClient(t, cfg)
	keys := rdb.Keys(t.Context(), "*").Val()
	for _, k := range keys {
		if !strings.HasPrefix(k, "ratelimit:login:") || strings.Contains(k, "example") || strings.Contains(k, "192.0.2") {
			t.Errorf("Redis key %q: want only the login rate-limit counters, named by HMAC (no session or revocation key)", k)
		}
	}
}

func redisClient(t *testing.T, cfg config.Config) *goredis.Client {
	t.Helper()
	opts, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		t.Fatal(err)
	}
	rdb := goredis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestNew_SignsAWebPatientInWithTheCookieAndChecksCSRF(t *testing.T) {
	app, _ := newApp(t)
	c := &client{t: t, h: app.Handler, web: true}
	c.do(http.MethodPost, "/auth/register", `{"email":"ana@example.test","password":"`+testPassword+`"}`)
	rec := c.do(http.MethodPost, "/auth/login", `{"email":"ana@example.test","password":"`+testPassword+`"}`)
	if rec.Code != http.StatusOK || c.cookie == nil || strings.Contains(rec.Body.String(), c.cookie.Value) {
		t.Fatalf("POST /auth/login = %d %s, want the session in the cookie only", rec.Code, rec.Body)
	}
	if rec := c.do(http.MethodGet, "/auth/me", ""); rec.Code != http.StatusOK {
		t.Errorf("GET /auth/me with the cookie = %d %s", rec.Code, rec.Body)
	}
	c.web = false
	if rec := c.do(http.MethodPost, "/auth/logout", ""); rec.Code != http.StatusForbidden {
		t.Errorf("POST /auth/logout with the cookie and no X-Salus-CSRF = %d %s, want 403", rec.Code, rec.Body)
	}
	c.web = true
	if rec := c.do(http.MethodPost, "/auth/logout", ""); rec.Code != http.StatusNoContent || c.cookie.MaxAge >= 0 {
		t.Errorf("POST /auth/logout = %d %s, cookie %+v; want 204 and the cookie cleared", rec.Code, rec.Body, c.cookie)
	}
}

func TestNew_EnrolsStaffAtTheirFirstLoginAndThenAsksForACode(t *testing.T) {
	app, cfg := newApp(t)
	hash, err := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), 1).Hash(t.Context(), testPassword)
	if err != nil {
		t.Fatal(err)
	}
	count := openDB(t, cfg)
	count(`WITH a AS (INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ('ben@example.test', $1, 'admin', 'c0000000-0000-4000-8000-00000000000c') RETURNING 1) SELECT count(*) FROM a`, hash)
	c := &client{t: t, h: app.Handler}
	login := `{"email":"ben@example.test","password":"` + testPassword + `"}`

	var step struct{ Next, Challenge, Token, Secret, URI string }
	rec := c.do(http.MethodPost, "/auth/login", login)
	decode(t, rec, &step)
	if step.Next != "enrol" || step.Challenge == "" || step.Token != "" {
		t.Fatalf("first staff login = %d %s, want next enrol and no session", rec.Code, rec.Body)
	}
	challenge := step.Challenge
	rec = c.do(http.MethodPost, "/auth/mfa/enrol", `{"challenge":"`+challenge+`"}`)
	decode(t, rec, &step)
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(step.Secret)
	if rec.Code != http.StatusOK || err != nil || !strings.HasPrefix(step.URI, "otpauth://totp/") {
		t.Fatalf("POST /auth/mfa/enrol = %d %s", rec.Code, rec.Body)
	}
	if n := count(`SELECT count(*) FROM account WHERE position($1::bytea IN totp_secret) > 0`, secret); n != 0 {
		t.Error("the TOTP secret is stored in plain text")
	}
	confirmed := clock.System{}.Now() // its step is used by confirm; the codes below are counted from it
	rec = c.do(http.MethodPost, "/auth/mfa/confirm", `{"challenge":"`+challenge+`","code":"`+auth.TOTPCode(secret, confirmed)+`"}`)
	decode(t, rec, &step)
	if rec.Code != http.StatusOK || step.Next != "done" || step.Token == "" {
		t.Fatalf("POST /auth/mfa/confirm = %d %s, want a session", rec.Code, rec.Body)
	}

	rec = c.do(http.MethodPost, "/auth/login", login)
	decode(t, rec, &step)
	if step.Next != "verify" || step.Challenge == "" {
		t.Fatalf("the next login = %d %s, want next verify", rec.Code, rec.Body)
	}
	code := auth.TOTPCode(secret, confirmed) // the step confirm used: rejected as reused
	rec = c.do(http.MethodPost, "/auth/mfa/verify", `{"challenge":"`+step.Challenge+`","code":"`+code+`"}`)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "wrong_code") {
		t.Errorf("verify with a code of the step already used = %d %s, want 401 wrong_code", rec.Code, rec.Body)
	}
	rec = c.do(http.MethodPost, "/auth/mfa/verify", `{"challenge":"`+step.Challenge+`","code":"`+auth.TOTPCode(secret, confirmed.Add(30*time.Second))+`"}`)
	decode(t, rec, &step)
	if rec.Code != http.StatusOK || step.Token == "" {
		t.Fatalf("verify with the next step's code = %d %s, want a session", rec.Code, rec.Body)
	}
	c.bearer = step.Token
	if rec := c.do(http.MethodGet, "/auth/me", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"mfaEnrolled":true`) {
		t.Errorf("GET /auth/me = %d %s, want the enrolled admin", rec.Code, rec.Body)
	}

	if n := count(`SELECT count(*) FROM session WHERE mfa_verified AND organization_id IS NOT NULL
		AND expires_at - created_at = interval '12 hours'`); n != 2 {
		t.Errorf("%d MFA-verified 12-hour staff sessions, want 2", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE action = 'U' AND subtype_code = 'update' AND outcome = 0
		AND organization_id IS NOT NULL`); n != 2 {
		t.Errorf("%d audited enrolment steps (enrol, confirm), want 2", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = '110122' AND outcome = 4`); n != 1 {
		t.Errorf("%d failed logins audited, want 1 (the reused code)", n)
	}
}
