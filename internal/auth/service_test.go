package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

const (
	accountID = "0b6c8a52-5d0e-4a8e-9a43-3f2f5f0f0a01"
	clinicID  = "c0000000-0000-4000-8000-00000000000c"
	password  = "a long enough secret"
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
	err       error // returned by every call when set
	rehashErr error // returned by SetPasswordHash when set
	accounts  map[string]auth.Account
	rehashed  map[string]string
	outside   int // calls made outside a transaction
}

func newFakeRepository(accounts ...auth.Account) *fakeRepository {
	r := &fakeRepository{accounts: map[string]auth.Account{}, rehashed: map[string]string{}}
	for _, a := range accounts {
		r.accounts[a.ID] = a
	}
	return r
}

func (r *fakeRepository) seen(ctx context.Context) {
	if ctx.Value(txKey{}) == nil {
		r.outside++
	}
}

func (r *fakeRepository) Create(ctx context.Context, a auth.Account) (string, error) {
	r.seen(ctx)
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
	r.seen(ctx)
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
	r.seen(ctx)
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
	r.seen(ctx)
	r.rehashed[id] = hash
	return r.rehashErr
}

// fakeHasher "hashes" by prefixing, and counts the hashes it computes: the timing seam for unknown e-mails.
type fakeHasher struct {
	err    error
	stale  bool
	hashes int
}

func (h *fakeHasher) Hash(_ context.Context, password string) (string, error) {
	h.hashes++
	return "hashed:" + strings.ToUpper(password), h.err
}

func (h *fakeHasher) Verify(_ context.Context, password, encoded string) (ok, stale bool, err error) {
	h.hashes++
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
	svc    *auth.Service
	repo   *fakeRepository
	hasher *fakeHasher
	audit  *fakeAuditor
}

func newFixture(accounts ...auth.Account) fixture {
	f := fixture{repo: newFakeRepository(accounts...), hasher: &fakeHasher{}, audit: &fakeAuditor{}}
	f.svc = auth.NewService(f.repo, f.hasher, f.audit, &fakeTransactor{})
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
	if f.repo.outside != 0 {
		t.Errorf("%d repository calls outside the transaction", f.repo.outside)
	}
	return f.audit.events[0]
}

var (
	hoperat  = auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT}
	rest     = auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest}
	create   = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate}
	read     = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead}
	userAuth = auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeUserAuthentication}
	login    = auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeLogin}
)

func TestRegister_CreatesAPatientWithTheNormalisedEmailAndOnlyTheHash(t *testing.T) {
	f := newFixture()

	got, err := f.svc.Register(t.Context(), "  Ana.Example@Example.TEST ", password)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	stored := f.repo.accounts[got.ID]
	if stored.Email != "ana.example@example.test" || got.Email != stored.Email {
		t.Errorf("stored e-mail %q, returned %q, want ana.example@example.test", stored.Email, got.Email)
	}
	if stored.Role != auth.RolePatient || stored.OrganizationID != "" {
		t.Errorf("stored role %q, clinic %q, want a patient with no clinic", stored.Role, stored.OrganizationID)
	}
	if stored.PasswordHash != "hashed:"+strings.ToUpper(password) || strings.Contains(stored.PasswordHash, password) {
		t.Errorf("stored password hash %q, want the hasher's output only", stored.PasswordHash)
	}
	if got.PasswordHash != "" {
		t.Error("Register returned the password hash")
	}
}

func TestRegister_AuditsTheCreatedAccount(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Register(t.Context(), "ana@example.test", password); err != nil {
		t.Fatal(err)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: create, Purpose: hoperat, Action: auditevent.ActionCreate, Outcome: auditevent.OutcomeSuccess,
		AgentAccountID: accountID, EntityReference: "Account/" + accountID,
	}
	if got := f.onlyEvent(t); got != want {
		t.Errorf("audit event %+v,\nwant %+v", got, want)
	}
}

func TestRegister_RejectsATakenEmailAndAuditsTheFailure(t *testing.T) {
	f := newFixture(auth.Account{ID: "existing", Email: "ana@example.test", Role: auth.RolePatient})

	_, err := f.svc.Register(t.Context(), "ANA@example.test", password)

	if !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("Register() error = %v, want ErrEmailTaken", err)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: create, Purpose: hoperat, Action: auditevent.ActionCreate, Outcome: auditevent.OutcomeMinorFailure,
	}
	if got := f.onlyEvent(t); got != want {
		t.Errorf("audit event %+v,\nwant %+v", got, want)
	}
}

func TestRegister_RejectsAnInvalidEmailOrPasswordBeforeTouchingAnything(t *testing.T) {
	tests := []struct {
		want            error
		name            string
		email, password string
	}{
		{name: "no e-mail", email: " ", password: password, want: auth.ErrInvalidEmail},
		{name: "no at sign", email: "ana.example.test", password: password, want: auth.ErrInvalidEmail},
		{name: "a display name", email: "Ana <ana@example.test>", password: password, want: auth.ErrInvalidEmail},
		{name: "e-mail too long", email: strings.Repeat("a", 250) + "@example.test", password: password, want: auth.ErrInvalidEmail},
		{name: "11 characters", email: "ana@example.test", password: "elevenchars", want: auth.ErrInvalidPassword},
		{name: "129 characters", email: "ana@example.test", password: strings.Repeat("p", 129), want: auth.ErrInvalidPassword},
		{name: "11 characters, multi-byte", email: "ana@example.test", password: strings.Repeat("ü", 11), want: auth.ErrInvalidPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			if _, err := f.svc.Register(t.Context(), tt.email, tt.password); !errors.Is(err, tt.want) {
				t.Errorf("Register() error = %v, want %v", err, tt.want)
			}
			if f.hasher.hashes != 0 || len(f.repo.accounts) != 0 {
				t.Error("an invalid registration was hashed or stored")
			}
		})
	}
}

func TestRegister_AcceptsTheLengthLimits(t *testing.T) {
	for _, pw := range []string{strings.Repeat("ü", 12), strings.Repeat("p", 128)} {
		f := newFixture()
		if _, err := f.svc.Register(t.Context(), "ana@example.test", pw); err != nil {
			t.Errorf("Register() with %d characters: error = %v", len([]rune(pw)), err)
		}
	}
}

func TestRegister_PassesFailuresOnWithoutTheirDetails(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"hasher":     func(f *fixture) { f.hasher.err = boom },
		"repository": func(f *fixture) { f.repo.err = boom },
		"audit":      func(f *fixture) { f.audit.err = boom },
	} {
		f := newFixture()
		change(&f)
		_, err := f.svc.Register(t.Context(), "ana@example.test", password)
		if !errors.Is(err, boom) || strings.Contains(err.Error(), "example.test") {
			t.Errorf("%s fails: Register() error = %v, want it wrapped without the e-mail", name, err)
		}
	}
}

func staff() auth.Account {
	return auth.Account{
		ID: accountID, Email: "ben@example.test", Role: auth.RoleAdmin, OrganizationID: clinicID,
		PasswordHash: "hashed:" + strings.ToUpper(password),
	}
}

func TestCheckPassword_ReturnsTheAccountAndAuditsTheLogin(t *testing.T) {
	f := newFixture(staff())

	got, err := f.svc.CheckPassword(t.Context(), " Ben@Example.test", password)

	if err != nil || got.ID != accountID {
		t.Fatalf("CheckPassword() = %+v, %v; want the account", got, err)
	}
	if got.PasswordHash != "" {
		t.Error("CheckPassword returned the password hash")
	}
	want := auditevent.AuditEvent{
		Type: userAuth, Subtype: login, Purpose: hoperat, Action: auditevent.ActionExecute, Outcome: auditevent.OutcomeSuccess,
		AgentAccountID: accountID, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
	if len(f.repo.rehashed) != 0 {
		t.Error("a current hash was re-hashed")
	}
}

func TestCheckPassword_RejectsAWrongPasswordAndAuditsTheFailure(t *testing.T) {
	f := newFixture(staff())

	_, err := f.svc.CheckPassword(t.Context(), "ben@example.test", "not the password")

	if !errors.Is(err, auth.ErrWrongCredentials) {
		t.Fatalf("CheckPassword() error = %v, want ErrWrongCredentials", err)
	}
	want := auditevent.AuditEvent{
		Type: userAuth, Subtype: login, Purpose: hoperat, Action: auditevent.ActionExecute, Outcome: auditevent.OutcomeMinorFailure,
		AgentAccountID: accountID, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
}

func TestCheckPassword_HashesADummyForAnUnknownEmail(t *testing.T) {
	f := newFixture(staff())

	_, err := f.svc.CheckPassword(t.Context(), "nobody@example.test", password)

	if !errors.Is(err, auth.ErrWrongCredentials) {
		t.Fatalf("CheckPassword() error = %v, want ErrWrongCredentials (the same as a wrong password)", err)
	}
	if f.hasher.hashes != 1 {
		t.Errorf("%d hashes for an unknown e-mail, want 1, as for a known one", f.hasher.hashes)
	}
	want := auditevent.AuditEvent{
		Type: userAuth, Subtype: login, Purpose: hoperat, Action: auditevent.ActionExecute, Outcome: auditevent.OutcomeMinorFailure,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v (no agent, no entity)", e, want)
	}
}

func TestCheckPassword_ReHashesAHashWithOlderParameters(t *testing.T) {
	f := newFixture(staff())
	f.hasher.stale = true

	if _, err := f.svc.CheckPassword(t.Context(), "ben@example.test", password); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.rehashed[accountID]; got != "hashed:"+strings.ToUpper(password) {
		t.Errorf("re-hashed to %q, want a fresh hash", got)
	}
	f.onlyEvent(t)
}

func TestCheckPassword_PassesFailuresOn(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"hasher":     func(f *fixture) { f.hasher.err = boom },
		"repository": func(f *fixture) { f.repo.err = boom },
		"audit":      func(f *fixture) { f.audit.err = boom },
		"re-hash":    func(f *fixture) { f.hasher.stale = true; f.repo.rehashErr = boom },
	} {
		f := newFixture(staff())
		change(&f)
		_, err := f.svc.CheckPassword(t.Context(), "ben@example.test", password)
		if err == nil || errors.Is(err, auth.ErrWrongCredentials) {
			t.Errorf("%s fails: CheckPassword() error = %v, want the failure", name, err)
		}
	}
}

func TestCheckPassword_PassesAnUnknownEmailsHashFailureOn(t *testing.T) {
	f := newFixture()
	f.hasher.err = context.Canceled
	if _, err := f.svc.CheckPassword(t.Context(), "nobody@example.test", password); !errors.Is(err, context.Canceled) {
		t.Errorf("CheckPassword() error = %v, want context.Canceled", err)
	}
}

func TestMe_ReturnsTheAccountAndAuditsTheRead(t *testing.T) {
	f := newFixture(staff())

	got, err := f.svc.Me(t.Context(), accountID)

	if err != nil || got.Email != "ben@example.test" || got.Role != auth.RoleAdmin || got.OrganizationID != clinicID {
		t.Fatalf("Me() = %+v, %v", got, err)
	}
	if got.PasswordHash != "" {
		t.Error("Me returned the password hash")
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: read, Purpose: hoperat, Action: auditevent.ActionRead, Outcome: auditevent.OutcomeSuccess,
		AgentAccountID: accountID, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}
}

func TestMe_AuditsAnUnknownAccount(t *testing.T) {
	f := newFixture()

	_, err := f.svc.Me(t.Context(), accountID)

	if !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("Me() error = %v, want ErrNotFound", err)
	}
	if e := f.onlyEvent(t); e.Outcome != auditevent.OutcomeMinorFailure || e.AgentAccountID != accountID {
		t.Errorf("audit event %+v, want a minor failure by the agent", e)
	}
}

func TestMe_PassesFailuresOn(t *testing.T) {
	boom := errors.New("boom")
	for name, change := range map[string]func(*fixture){
		"repository": func(f *fixture) { f.repo.err = boom },
		"audit":      func(f *fixture) { f.audit.err = boom },
	} {
		f := newFixture(staff())
		change(&f)
		if _, err := f.svc.Me(t.Context(), accountID); !errors.Is(err, boom) {
			t.Errorf("%s fails: Me() error = %v, want it wrapped", name, err)
		}
	}
}
