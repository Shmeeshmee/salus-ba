package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Session and login limits (docs/accounts/SPEC.md "Session tokens", "Login with MFA", "Login rate limit").
const (
	staffSessionLifetime   = 12 * time.Hour
	patientSessionLifetime = 30 * 24 * time.Hour
	challengeTTL           = 5 * time.Minute
	maxWrongCodes          = 5 // per challenge
	loginWindow            = 15 * time.Minute
	maxFailuresPerEmail    = 5  // per window
	maxAttemptsPerIP       = 30 // per window
	tokenLen               = 32 // bytes, for session tokens and challenges
)

// Login checks the password and either opens a session (a patient without MFA) or returns a challenge for
// the TOTP step: verify for an enrolled account, enrol for staff who have not set up MFA. Over the rate
// limit it returns a RateLimitedError without checking the password. Every checked attempt is audited.
func (s *Service) Login(ctx context.Context, email, password, clientIP string) (LoginResult, error) {
	email = normaliseEmail(email)
	if err := s.checkLoginRate(ctx, email, clientIP); err != nil {
		return LoginResult{}, err
	}
	// The hashes run before the transaction, so no connection is held while they do (docs/accounts/
	// PROGRESS.md, package 3: holding one per login let a burst take the whole pool).
	pre, verified, rehash, err := s.verifyPassword(ctx, email, password)
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: login: %w", err)
	}
	var res LoginResult
	var a Account
	var ok bool
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		a, ok, err = s.passwordLogin(ctx, pre, verified, rehash)
		if err != nil || !ok {
			return err
		}
		if res.Next = nextStep(a); res.Next == NextDone {
			res.Session, res.Token, err = s.openSession(ctx, a, false)
		}
		return err
	})
	switch {
	case err != nil:
		return LoginResult{}, fmt.Errorf("auth: login: %w", err)
	case !ok:
		if _, _, err := s.counters.Incr(ctx, s.counterKey("login:email:", email), loginWindow); err != nil {
			return LoginResult{}, fmt.Errorf("auth: login: count the failure: %w", err)
		}
		return LoginResult{}, ErrWrongCredentials
	case res.Next != NextDone:
		if res.Challenge, err = s.newChallenge(ctx, a.ID); err != nil {
			return LoginResult{}, fmt.Errorf("auth: login: %w", err)
		}
	}
	return res, nil
}

// checkLoginRate counts the attempt from clientIP and returns a RateLimitedError when the IP's attempts or
// the e-mail's failures are over their limit in the current window.
func (s *Service) checkLoginRate(ctx context.Context, email, clientIP string) error {
	n, left, err := s.counters.Incr(ctx, s.counterKey("login:ip:", clientIP), loginWindow)
	if err != nil {
		return fmt.Errorf("auth: login: rate limit: %w", err)
	}
	if n > maxAttemptsPerIP {
		return &RateLimitedError{RetryAfter: left}
	}
	n, left, err = s.counters.Get(ctx, s.counterKey("login:email:", email))
	if err != nil {
		return fmt.Errorf("auth: login: rate limit: %w", err)
	}
	if n >= maxFailuresPerEmail {
		return &RateLimitedError{RetryAfter: left}
	}
	return nil
}

// counterKey names a counter by the HMAC of value, so neither the e-mail nor the IP is readable in Redis.
func (s *Service) counterKey(prefix, value string) string {
	mac := hmac.New(sha256.New, s.rateKey)
	mac.Write([]byte(value))
	return prefix + hex.EncodeToString(mac.Sum(nil))
}

// verifyPassword reads the account with email, outside any transaction, and verifies password against its
// hash. For an unknown e-mail it hashes a dummy password and returns an empty account, so timing does not
// tell the two apart. When the stored hash uses older parameters, rehash is a new one.
func (s *Service) verifyPassword(ctx context.Context, email, password string) (a Account, ok bool, rehash string, err error) {
	a, err = s.repo.ByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_, err = s.hasher.Hash(ctx, dummyPassword)
		return Account{}, false, "", err
	}
	if err != nil {
		return Account{}, false, "", err
	}
	ok, stale, err := s.hasher.Verify(ctx, password, a.PasswordHash)
	if err != nil || !ok || !stale {
		return a, ok, "", err
	}
	rehash, err = s.hasher.Hash(ctx, password)
	return a, ok, rehash, err
}

// passwordLogin decides a password check inside the transaction and audits it as a login. The decision
// rests on the account as read here, not on the pre-read: it fails when the hash that was verified has been
// replaced in between.
func (s *Service) passwordLogin(ctx context.Context, pre Account, verified bool, rehash string) (Account, bool, error) {
	if pre.ID == "" { // unknown e-mail
		return Account{}, false, s.recordLogin(ctx, Account{}, auditevent.CodeLogin, auditevent.OutcomeMinorFailure)
	}
	a, err := s.repo.ByID(ctx, pre.ID)
	if errors.Is(err, ErrNotFound) {
		return Account{}, false, s.recordLogin(ctx, Account{}, auditevent.CodeLogin, auditevent.OutcomeMinorFailure)
	}
	if err != nil {
		return Account{}, false, err
	}
	if !verified || a.PasswordHash != pre.PasswordHash {
		return a, false, s.recordLogin(ctx, a, auditevent.CodeLogin, auditevent.OutcomeMinorFailure)
	}
	if rehash != "" {
		if err := s.repo.SetPasswordHash(ctx, a.ID, rehash); err != nil {
			return Account{}, false, err
		}
	}
	return a, true, s.recordLogin(ctx, a, auditevent.CodeLogin, auditevent.OutcomeSuccess)
}

// nextStep is what a login with the right password asks for next.
func nextStep(a Account) Next {
	switch {
	case a.MFAEnrolled():
		return NextVerify
	case a.Role != RolePatient: // MFA is required for staff
		return NextEnrol
	}
	return NextDone
}

// openSession stores a new session for a inside the caller's transaction and returns it with its token.
func (s *Service) openSession(ctx context.Context, a Account, mfaVerified bool) (Session, string, error) {
	token, hash := newToken()
	now := s.clock.Now()
	lifetime := staffSessionLifetime
	if a.Role == RolePatient {
		lifetime = patientSessionLifetime
	}
	sess := Session{AccountID: a.ID, OrganizationID: a.OrganizationID, MFAVerified: mfaVerified, ExpiresAt: now.Add(lifetime)}
	id, err := s.repo.CreateSession(ctx, sess, hash, now)
	if err != nil {
		return Session{}, "", err
	}
	sess.ID = id
	return sess, token, nil
}

// newChallenge stores a challenge for the account and returns it.
func (s *Service) newChallenge(ctx context.Context, accountID string) (string, error) {
	challenge, hash := newToken()
	if err := s.challenges.Put(ctx, hash, accountID, challengeTTL); err != nil {
		return "", err
	}
	return challenge, nil
}

// newToken returns tokenLen random bytes as base64url, and their SHA-256, the only form that is stored.
func newToken() (string, []byte) {
	raw := make([]byte, tokenLen)
	_, _ = rand.Read(raw) // crypto/rand.Read never returns an error
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), sum[:]
}

// tokenHash returns the SHA-256 of a token or challenge, and false when it is not tokenLen bytes of
// base64url.
func tokenHash(token string) ([]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenLen {
		return nil, false
	}
	sum := sha256.Sum256(raw)
	return sum[:], true
}

// Authenticate returns the open session of token: one indexed read, no Redis, not audited on its own
// (docs/accounts/SPEC.md "What every access to an account means"). An unknown, expired or revoked token
// gives ErrUnauthenticated.
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	hash, ok := tokenHash(token)
	if !ok {
		return Session{}, ErrUnauthenticated
	}
	sess, err := s.repo.SessionByTokenHash(ctx, hash, s.clock.Now())
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth: authenticate: %w", err)
	}
	return sess, nil
}

// Logout ends the session: it sets revoked_at, which the next request's Authenticate sees. Audited; a
// session that is no longer open gives ErrUnauthenticated.
func (s *Service) Logout(ctx context.Context, sess Session) error {
	a := Account{ID: sess.AccountID, OrganizationID: sess.OrganizationID}
	var gone bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		err := s.repo.RevokeSession(ctx, sess.ID, s.clock.Now())
		gone = errors.Is(err, ErrNotFound)
		if err != nil && !gone {
			return err
		}
		outcome := auditevent.OutcomeSuccess
		if gone {
			outcome = auditevent.OutcomeMinorFailure
		}
		return s.recordLogin(ctx, a, auditevent.CodeLogout, outcome)
	})
	switch {
	case err != nil:
		return fmt.Errorf("auth: logout: %w", err)
	case gone:
		return ErrUnauthenticated
	}
	return nil
}
