package auth_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

type txKey struct{}

// fakeTransactor marks the context it hands fn, as postgres.Transactor carries its transaction, and keeps
// what fn returned.
type fakeTransactor struct{ calls int }

func (f *fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds accounts by id.
type fakeRepository struct {
	err        error // returned by every call when set
	rehashErr  error // returned by SetPasswordHash when set
	sessionErr error // returned by the session calls when set
	accounts   map[string]auth.Account
	rehashed   map[string]string
	secrets    map[string][]byte // sealed TOTP secrets by account
	lastStep   map[string]int64
	sessions   map[string]*fakeSession // by token hash
	inTx       func()                  // called on the first repository call inside a transaction, when set
	outside    []string                // the calls made outside a transaction
}

type fakeSession struct {
	revokedAt time.Time
	createdAt time.Time
	auth.Session
}

func newFakeRepository(accounts ...auth.Account) *fakeRepository {
	r := &fakeRepository{
		accounts: map[string]auth.Account{}, rehashed: map[string]string{}, secrets: map[string][]byte{},
		lastStep: map[string]int64{}, sessions: map[string]*fakeSession{},
	}
	for _, a := range accounts {
		r.accounts[a.ID] = a
	}
	return r
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		r.outside = append(r.outside, call)
		return
	}
	if r.inTx != nil {
		r.inTx()
		r.inTx = nil
	}
}

func (r *fakeRepository) Create(ctx context.Context, a auth.Account) (string, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return "", r.err
	}
	for _, existing := range r.accounts {
		if existing.Email == a.Email {
			return "", auth.ErrEmailTaken
		}
	}
	a.ID = accountID
	r.accounts[a.ID] = a
	return a.ID, nil
}

func (r *fakeRepository) ByEmail(ctx context.Context, email string) (auth.Account, error) {
	r.seen(ctx, "ByEmail")
	if r.err != nil {
		return auth.Account{}, r.err
	}
	for _, a := range r.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return auth.Account{}, auth.ErrNotFound
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (auth.Account, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return auth.Account{}, r.err
	}
	a, ok := r.accounts[id]
	if !ok {
		return auth.Account{}, auth.ErrNotFound
	}
	return a, nil
}

func (r *fakeRepository) SetPasswordHash(ctx context.Context, id, hash string) error {
	r.seen(ctx, "SetPasswordHash")
	r.rehashed[id] = hash
	return r.rehashErr
}

func (r *fakeRepository) TOTPSecret(ctx context.Context, id string) ([]byte, error) {
	r.seen(ctx, "TOTPSecret")
	return r.secrets[id], r.err
}

func (r *fakeRepository) SetTOTPSecret(ctx context.Context, id string, sealed []byte) error {
	r.seen(ctx, "SetTOTPSecret")
	r.secrets[id] = sealed
	return r.err
}

func (r *fakeRepository) ConfirmMFA(ctx context.Context, id string, step int64, at time.Time) error {
	r.seen(ctx, "ConfirmMFA")
	a := r.accounts[id]
	if a.MFAEnrolled() {
		return auth.ErrAlreadyEnrolled
	}
	a.MFAEnrolledAt = at
	r.accounts[id], r.lastStep[id] = a, step
	return r.err
}

func (r *fakeRepository) UseTOTPStep(ctx context.Context, id string, step int64) (bool, error) {
	r.seen(ctx, "UseTOTPStep")
	if last, ok := r.lastStep[id]; ok && step <= last {
		return false, r.err
	}
	r.lastStep[id] = step
	return true, r.err
}

func (r *fakeRepository) CreateSession(ctx context.Context, s auth.Session, tokenHash []byte, at time.Time) (string, error) {
	r.seen(ctx, "CreateSession")
	if r.sessionErr != nil {
		return "", r.sessionErr
	}
	s.ID = fmt.Sprintf("5e550000-0000-4000-8000-%012d", len(r.sessions)+1)
	r.sessions[string(tokenHash)] = &fakeSession{Session: s, createdAt: at}
	return s.ID, nil
}

func (r *fakeRepository) SessionByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (auth.Session, error) {
	r.seen(ctx, "SessionByTokenHash")
	if r.sessionErr != nil {
		return auth.Session{}, r.sessionErr
	}
	s, ok := r.sessions[string(tokenHash)]
	if !ok || !s.revokedAt.IsZero() || !s.ExpiresAt.After(now) {
		return auth.Session{}, auth.ErrNotFound
	}
	return s.Session, nil
}

func (r *fakeRepository) RevokeSession(ctx context.Context, id string, at time.Time) error {
	r.seen(ctx, "RevokeSession")
	if r.sessionErr != nil {
		return r.sessionErr
	}
	for _, s := range r.sessions {
		if s.ID == id && s.revokedAt.IsZero() {
			s.revokedAt = at
			return nil
		}
	}
	return auth.ErrNotFound
}

// session returns the one stored session, failing unless there is exactly one.
func (r *fakeRepository) session(t *testing.T) *fakeSession {
	t.Helper()
	if len(r.sessions) != 1 {
		t.Fatalf("%d sessions stored, want 1", len(r.sessions))
	}
	for _, s := range r.sessions {
		return s
	}
	return nil
}

// fakeChallenges keeps challenges by hash, as the Redis store does, deleting one at its limit-th failure.
type fakeChallenges struct {
	err        error // returned by Put, Fail and Delete when set
	getErr     error // returned by Get when set
	challenges map[string]*fakeChallenge
}

type fakeChallenge struct {
	accountID string
	ttl       time.Duration
	fails     int
}

func (c *fakeChallenges) Put(_ context.Context, hash []byte, accountID string, ttl time.Duration) error {
	c.challenges[string(hash)] = &fakeChallenge{accountID: accountID, ttl: ttl}
	return c.err
}

func (c *fakeChallenges) Get(_ context.Context, hash []byte) (string, error) {
	ch, ok := c.challenges[string(hash)]
	if !ok {
		return "", auth.ErrInvalidChallenge
	}
	return ch.accountID, c.getErr
}

func (c *fakeChallenges) Fail(_ context.Context, hash []byte, limit int) error {
	if ch, ok := c.challenges[string(hash)]; ok {
		if ch.fails++; ch.fails >= limit {
			delete(c.challenges, string(hash))
		}
	}
	return c.err
}

func (c *fakeChallenges) Delete(_ context.Context, hash []byte) error {
	delete(c.challenges, string(hash))
	return c.err
}

// fakeCounters are fixed-window counters whose windows never move on: each reports the window it was
// started with as the time left.
type fakeCounters struct {
	err    error
	counts map[string]int64
	window map[string]time.Duration
}

func (c *fakeCounters) Incr(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	c.counts[key]++
	if _, ok := c.window[key]; !ok {
		c.window[key] = window
	}
	return c.counts[key], c.window[key], c.err
}

func (c *fakeCounters) Get(_ context.Context, key string) (int64, time.Duration, error) {
	return c.counts[key], c.window[key], c.err
}

// fakeBox "seals" by prefixing the account id, so a secret opens only for its account.
type fakeBox struct{ err error }

func (b fakeBox) Seal(plaintext []byte, accountID string) ([]byte, error) {
	return append([]byte("sealed:"+accountID+":"), plaintext...), b.err
}

func (b fakeBox) Open(sealed []byte, accountID string) ([]byte, error) {
	rest, ok := bytes.CutPrefix(sealed, []byte("sealed:"+accountID+":"))
	if !ok {
		return nil, auth.ErrSealedSecret
	}
	return rest, b.err
}

// fakeHasher "hashes" by prefixing, and counts the hashes it computes: the timing seam for unknown e-mails.
type fakeHasher struct {
	err    error
	stale  bool
	hashes int
	inTx   int // hashes computed inside a transaction
}

func (h *fakeHasher) count(ctx context.Context) {
	h.hashes++
	if ctx.Value(txKey{}) != nil {
		h.inTx++
	}
}

func (h *fakeHasher) Hash(ctx context.Context, password string) (string, error) {
	h.count(ctx)
	return "hashed:" + strings.ToUpper(password), h.err
}

func (h *fakeHasher) Verify(ctx context.Context, password, encoded string) (ok, stale bool, err error) {
	h.count(ctx)
	ok = encoded == "hashed:"+strings.ToUpper(password)
	return ok, ok && h.stale, h.err
}

// fakeAuditor keeps the events it records and whether each was recorded inside a transaction.
type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
	inTx   []bool
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	a.inTx = append(a.inTx, ctx.Value(txKey{}) != nil)
	return a.err
}

type fixture struct {
	svc        *auth.Service
	repo       *fakeRepository
	hasher     *fakeHasher
	audit      *fakeAuditor
	challenges *fakeChallenges
	counters   *fakeCounters
	box        *fakeBox
}

func newFixture(accounts ...auth.Account) fixture {
	f := fixture{
		repo: newFakeRepository(accounts...), hasher: &fakeHasher{}, audit: &fakeAuditor{},
		challenges: &fakeChallenges{challenges: map[string]*fakeChallenge{}},
		counters:   &fakeCounters{counts: map[string]int64{}, window: map[string]time.Duration{}},
		box:        &fakeBox{},
	}
	f.svc = auth.NewService(auth.Ports{
		Repo: f.repo, Hasher: f.hasher, Audit: f.audit, Tx: &fakeTransactor{}, Clock: clock.Fixed{T: now},
		Challenges: f.challenges, Counters: f.counters, Box: f.box, RateLimitKey: []byte("synthetic rate-limit key for tests"),
	})
	return f
}

// onlyEvent returns the one recorded event, failing unless exactly one was recorded inside the transaction.
func (f fixture) onlyEvent(t *testing.T) auditevent.AuditEvent {
	t.Helper()
	if len(f.audit.events) != 1 {
		t.Fatalf("%d audit events, want 1", len(f.audit.events))
	}
	if !f.audit.inTx[0] {
		t.Error("the audit event was recorded outside the transaction")
	}
	return f.audit.events[0]
}
