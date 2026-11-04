package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const clientIP = "192.0.2.10" // TEST-NET-1

// hashOf is what the service stores for a token or challenge: the SHA-256 of its 32 bytes.
func hashOf(t *testing.T, token string) []byte {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token %q is not 32 bytes of base64url: %v", token, err)
	}
	sum := sha256.Sum256(raw)
	return sum[:]
}

func loginEvent(a auth.Account, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type: userAuth, Subtype: login, Purpose: hoperat, Action: auditevent.ActionExecute, Outcome: outcome,
		AgentAccountID: a.ID, OrganizationID: a.OrganizationID,
	}
	if a.ID != "" {
		e.EntityReference = "Account/" + a.ID
	}
	return e
}

func withHash(a auth.Account, hash string) auth.Account {
	a.PasswordHash = hash
	return a
}

func enrolled(a auth.Account) auth.Account {
	a.MFAEnrolledAt = now.Add(-time.Hour)
	return a
}

func TestLogin_OpensASessionForAPatientWithoutMFA(t *testing.T) {
	f := newFixture(patientWithoutMFA())

	got, err := f.svc.Login(t.Context(), " Ana@Example.test", password, clientIP)

	if err != nil || got.Next != auth.NextDone || got.Challenge != "" {
		t.Fatalf("Login() = %+v, %v; want an open session", got, err)
	}
	s := f.repo.session(t)
	if f.repo.sessions[string(hashOf(t, got.Token))] != s {
		t.Error("the session is not stored under the SHA-256 of its token")
	}
	want := auth.Session{ID: s.ID, AccountID: accountID, ExpiresAt: now.Add(30 * 24 * time.Hour)}
	if s.Session != want || got.Session != want || !s.createdAt.Equal(now) {
		t.Errorf("session %+v created %v, returned %+v;\nwant %+v created now", s.Session, s.createdAt, got.Session, want)
	}
	if e := f.onlyEvent(t); e != loginEvent(patientWithoutMFA(), auditevent.OutcomeSuccess) {
		t.Errorf("audit event %+v", e)
	}
	if f.repo.outside[0] != "ByEmail" || len(f.repo.outside) != 1 {
		t.Errorf("repository calls outside the transaction: %v, want the password pre-read only", f.repo.outside)
	}
}

func TestLogin_GivesEachSessionItsOwnToken(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	first, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	second, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	if first.Token == second.Token || len(f.repo.sessions) != 2 {
		t.Errorf("two logins gave tokens %q and %q, %d sessions", first.Token, second.Token, len(f.repo.sessions))
	}
}

func TestLogin_AsksForMFAAndOpensNoSessionUntilThen(t *testing.T) {
	tests := []struct {
		account auth.Account
		name    string
		want    auth.Next
	}{
		{name: "staff not enrolled", account: staff(), want: auth.NextEnrol},
		{name: "staff enrolled", account: enrolled(staff()), want: auth.NextVerify},
		{name: "a patient enrolled", account: enrolled(patientWithoutMFA()), want: auth.NextVerify},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tt.account)

			got, err := f.svc.Login(t.Context(), tt.account.Email, password, clientIP)

			if err != nil || got.Next != tt.want || got.Token != "" || len(f.repo.sessions) != 0 {
				t.Fatalf("Login() = %+v, %v with %d sessions; want next %q and no session", got, err, len(f.repo.sessions), tt.want)
			}
			ch := f.challenges.challenges[string(hashOf(t, got.Challenge))]
			if ch == nil || ch.accountID != accountID || ch.ttl != 5*time.Minute {
				t.Errorf("challenge stored as %+v, want the account for 5 minutes under its hash", ch)
			}
			if e := f.onlyEvent(t); e != loginEvent(tt.account, auditevent.OutcomeSuccess) {
				t.Errorf("audit event %+v", e)
			}
		})
	}
}

func TestLogin_RejectsAWrongPasswordCountsItAndAuditsTheFailure(t *testing.T) {
	f := newFixture(staff())

	_, err := f.svc.Login(t.Context(), "ben@example.test", "not the password", clientIP)

	if !errors.Is(err, auth.ErrWrongCredentials) {
		t.Fatalf("Login() error = %v, want ErrWrongCredentials", err)
	}
	if e := f.onlyEvent(t); e != loginEvent(staff(), auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v", e)
	}
	if len(f.challenges.challenges) != 0 || len(f.repo.sessions) != 0 {
		t.Error("a wrong password gave a challenge or a session")
	}
	if n := f.emailFailures(); n != 1 {
		t.Errorf("%d failures counted for the e-mail, want 1", n)
	}
}

// emailFailures returns the failure count of the one e-mail counter.
func (f fixture) emailFailures() int64 {
	var n int64
	for k, v := range f.counters.counts {
		if strings.HasPrefix(k, "login:email:") {
			n += v
		}
	}
	return n
}

func TestLogin_HashesADummyForAnUnknownEmail(t *testing.T) {
	f := newFixture(staff())

	_, err := f.svc.Login(t.Context(), "nobody@example.test", password, clientIP)

	if !errors.Is(err, auth.ErrWrongCredentials) {
		t.Fatalf("Login() error = %v, want ErrWrongCredentials (the same as a wrong password)", err)
	}
	if f.hasher.hashes != 1 {
		t.Errorf("%d hashes for an unknown e-mail, want 1, as for a known one", f.hasher.hashes)
	}
	if e := f.onlyEvent(t); e != loginEvent(auth.Account{}, auditevent.OutcomeMinorFailure) {
		t.Errorf("audit event %+v, want one with no agent and no entity", e)
	}
	if n := f.emailFailures(); n != 1 {
		t.Errorf("%d failures counted for the e-mail, want 1", n)
	}
}

func TestLogin_ReHashesAHashWithOlderParameters(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	f.hasher.stale = true

	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.rehashed[accountID]; got != "hashed:"+strings.ToUpper(password) {
		t.Errorf("re-hashed to %q, want a fresh hash", got)
	}
	f.onlyEvent(t)
}

func TestLogin_HashesOutsideTheTransaction(t *testing.T) {
	for name, change := range map[string]func(*fixture){
		"known e-mail":   func(*fixture) {},
		"unknown e-mail": func(f *fixture) { f.repo.accounts = map[string]auth.Account{} },
		"stale hash":     func(f *fixture) { f.hasher.stale = true },
		"wrong password": func(f *fixture) { f.repo.accounts[accountID] = withHash(staff(), "hashed:OTHER") },
	} {
		f := newFixture(staff())
		change(&f)
		_, _ = f.svc.Login(t.Context(), "ben@example.test", password, clientIP)
		if f.hasher.hashes == 0 || f.hasher.inTx != 0 {
			t.Errorf("%s: %d of %d hashes inside the transaction, want none (it would hold a connection)", name, f.hasher.inTx, f.hasher.hashes)
		}
	}
}

func TestLogin_DecidesOnTheAccountAsReadInTheTransaction(t *testing.T) {
	t.Run("the account enrolled while hashing", func(t *testing.T) {
		f := newFixture(patientWithoutMFA())
		f.repo.inTx = func() { f.repo.accounts[accountID] = enrolled(patientWithoutMFA()) }
		got, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
		if err != nil || got.Next != auth.NextVerify {
			t.Errorf("Login() = %+v, %v; want the next step of the account as read in the transaction", got, err)
		}
	})
	t.Run("the hash was replaced while hashing", func(t *testing.T) {
		f := newFixture(staff())
		f.repo.inTx = func() { f.repo.accounts[accountID] = withHash(staff(), "hashed:NEWER") }
		_, err := f.svc.Login(t.Context(), "ben@example.test", password, clientIP)
		if !errors.Is(err, auth.ErrWrongCredentials) {
			t.Fatalf("Login() error = %v, want ErrWrongCredentials: the verified hash is no longer the stored one", err)
		}
		if e := f.onlyEvent(t); e != loginEvent(staff(), auditevent.OutcomeMinorFailure) {
			t.Errorf("audit event %+v, want a failed login by the account", e)
		}
	})
	t.Run("the account was removed while hashing", func(t *testing.T) {
		f := newFixture(staff())
		f.repo.inTx = func() { delete(f.repo.accounts, accountID) }
		_, err := f.svc.Login(t.Context(), "ben@example.test", password, clientIP)
		if !errors.Is(err, auth.ErrWrongCredentials) {
			t.Fatalf("Login() error = %v, want ErrWrongCredentials", err)
		}
		if e := f.onlyEvent(t); e != loginEvent(auth.Account{}, auditevent.OutcomeMinorFailure) {
			t.Errorf("audit event %+v, want a failed login with no agent", e)
		}
	})
}

func TestLogin_LimitsFailuresPerEmailWithoutCheckingThePassword(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	for i := range 5 {
		ip := "192.0.2." + string(rune('1'+i)) // other addresses: the e-mail's limit holds across them
		if _, err := f.svc.Login(t.Context(), "ana@example.test", "not the password", ip); !errors.Is(err, auth.ErrWrongCredentials) {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	hashes, events := f.hasher.hashes, len(f.audit.events)

	_, err := f.svc.Login(t.Context(), "ANA@example.test", password, "198.51.100.7")

	var limited *auth.RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 15*time.Minute {
		t.Fatalf("Login() after 5 failures = %v, want RateLimitedError with the window's 15 minutes left", err)
	}
	if f.hasher.hashes != hashes || len(f.audit.events) != events || len(f.repo.sessions) != 0 {
		t.Error("a limited login checked the password, was audited or opened a session")
	}
}

func TestLogin_LimitsAttemptsPerIP(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	for i := range 30 { // successful ones count too
		if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP); err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	var limited *auth.RateLimitedError
	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP); !errors.As(err, &limited) {
		t.Fatalf("attempt 31 from one IP: %v, want RateLimitedError", err)
	}
	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, "198.51.100.7"); err != nil {
		t.Errorf("an attempt from another IP: %v, want it to pass", err)
	}
}

func TestLogin_KeysTheCountersByHMACNotByEmailOrIP(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	_, _ = f.svc.Login(t.Context(), "ana@example.test", "not the password", clientIP)

	if len(f.counters.counts) != 2 {
		t.Fatalf("counters %v, want one for the IP and one for the e-mail", f.counters.counts)
	}
	for k := range f.counters.counts {
		if strings.Contains(k, "ana") || strings.Contains(k, "example") || strings.Contains(k, "192.0.2") {
			t.Errorf("counter key %q holds the e-mail or the IP", k)
		}
		if !strings.HasPrefix(k, "login:ip:") && !strings.HasPrefix(k, "login:email:") || len(k) < len("login:ip:")+64 {
			t.Errorf("counter key %q, want login:ip: or login:email: and a hex HMAC-SHA256", k)
		}
	}
}

func TestLogin_PassesFailuresOn(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"hasher":    func(f *fixture) { f.hasher.err = boom },
		"read":      func(f *fixture) { f.repo.err = boom },
		"audit":     func(f *fixture) { f.audit.err = boom },
		"re-hash":   func(f *fixture) { f.hasher.stale = true; f.repo.rehashErr = boom },
		"session":   func(f *fixture) { f.repo.sessionErr = boom },
		"challenge": func(f *fixture) { f.repo.accounts[accountID] = enrolled(patientWithoutMFA()); f.challenges.err = boom },
		"counters":  func(f *fixture) { f.counters.err = boom },
	} {
		f := newFixture(patientWithoutMFA())
		change(&f)
		_, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
		if !errors.Is(err, boom) || strings.Contains(err.Error(), "example.test") {
			t.Errorf("%s fails: Login() error = %v, want it wrapped without the e-mail", name, err)
		}
	}
}

func TestLogin_PassesAFailureToCountAWrongPasswordOn(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	boom := errors.New("boom")
	failing := &failingCounters{fakeCounters: f.counters, failOn: "login:email:", err: boom}
	svc := auth.NewService(auth.Ports{
		Repo: f.repo, Hasher: f.hasher, Audit: f.audit, Tx: &fakeTransactor{}, Clock: clock.Fixed{T: now},
		Counters: failing, RateLimitKey: []byte("k"),
	})
	if _, err := svc.Login(t.Context(), "ana@example.test", "not the password", clientIP); !errors.Is(err, boom) {
		t.Errorf("Login() error = %v, want the counter's failure", err)
	}
}

// failingCounters fails Incr for keys with the prefix failOn.
type failingCounters struct {
	*fakeCounters
	err    error
	failOn string
}

func (c *failingCounters) Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if strings.HasPrefix(key, c.failOn) {
		return 0, 0, c.err
	}
	return c.fakeCounters.Incr(ctx, key, window)
}

func TestAuthenticate_ReturnsAnOpenSessionWithoutAuditing(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	res, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	if err != nil {
		t.Fatal(err)
	}
	events := len(f.audit.events)

	got, err := f.svc.Authenticate(t.Context(), res.Token)

	if err != nil || got != res.Session {
		t.Errorf("Authenticate() = %+v, %v; want %+v", got, err, res.Session)
	}
	if len(f.audit.events) != events {
		t.Error("the per-request session check was audited (SPEC.md: only the login and the access are)")
	}
}

func TestAuthenticate_RejectsUnknownMalformedExpiredAndRevokedTokens(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	res, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	expired := newFixture(patientWithoutMFA())
	expiredRes, _ := expired.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	expired.repo.session(t).ExpiresAt = now
	revoked := newFixture(patientWithoutMFA())
	revokedRes, _ := revoked.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	if err := revoked.svc.Logout(t.Context(), revokedRes.Session); err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		svc   *auth.Service
		token string
	}{
		"unknown":       {f.svc, base64.RawURLEncoding.EncodeToString(make([]byte, 32))},
		"not base64url": {f.svc, res.Token[:42] + "*"},
		"too short":     {f.svc, res.Token[:20]},
		"empty":         {f.svc, ""},
		"expired":       {expired.svc, expiredRes.Token},
		"revoked":       {revoked.svc, revokedRes.Token},
	}
	for name, tt := range tests {
		if _, err := tt.svc.Authenticate(t.Context(), tt.token); !errors.Is(err, auth.ErrUnauthenticated) {
			t.Errorf("%s token: Authenticate() error = %v, want ErrUnauthenticated", name, err)
		}
	}
}

func TestAuthenticate_PassesFailuresOn(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	res, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
	boom := errors.New("boom")
	f.repo.sessionErr = boom
	if _, err := f.svc.Authenticate(t.Context(), res.Token); !errors.Is(err, boom) {
		t.Errorf("Authenticate() error = %v, want the repository's failure", err)
	}
}

func TestLogout_RevokesTheSessionAndAuditsIt(t *testing.T) {
	f := newFixture(enrolled(staff()))
	f.repo.sessions["h"] = &fakeSession{Session: auth.Session{
		ID: "5e550000-0000-4000-8000-000000000001", AccountID: accountID, OrganizationID: clinicID,
		MFAVerified: true, ExpiresAt: now.Add(time.Hour),
	}}
	s := f.repo.sessions["h"].Session

	if err := f.svc.Logout(t.Context(), s); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !f.repo.sessions["h"].revokedAt.Equal(now) {
		t.Errorf("revoked at %v, want now", f.repo.sessions["h"].revokedAt)
	}
	want := auditevent.AuditEvent{
		Type: userAuth, Subtype: logout, Purpose: hoperat, Action: auditevent.ActionExecute, Outcome: auditevent.OutcomeSuccess,
		AgentAccountID: accountID, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
	if len(f.challenges.challenges) != 0 || len(f.counters.counts) != 0 {
		t.Error("logout wrote to Redis; revocation is session.revoked_at only")
	}

	f.audit.events = nil
	if err := f.svc.Logout(t.Context(), s); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Errorf("a second Logout() error = %v, want ErrUnauthenticated", err)
	}
	if e := f.onlyEvent(t); e.Outcome != auditevent.OutcomeMinorFailure || e.Subtype != logout {
		t.Errorf("audit event %+v, want a failed logout", e)
	}
}

func TestLogout_PassesFailuresOn(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"revoke": func(f *fixture) { f.repo.sessionErr = boom },
		"audit":  func(f *fixture) { f.audit.err = boom },
	} {
		f := newFixture(patientWithoutMFA())
		res, _ := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
		change(&f)
		if err := f.svc.Logout(t.Context(), res.Session); !errors.Is(err, boom) {
			t.Errorf("%s fails: Logout() error = %v, want it wrapped", name, err)
		}
	}
}
