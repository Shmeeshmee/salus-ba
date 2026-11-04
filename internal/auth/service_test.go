package auth_test

import (
	"errors"
	"slices"
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

var (
	hoperat  = auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT}
	rest     = auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest}
	create   = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate}
	read     = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead}
	update   = auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate}
	userAuth = auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeUserAuthentication}
	login    = auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeLogin}
	logout   = auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeLogout}
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

func patientWithoutMFA() auth.Account {
	return auth.Account{ID: accountID, Email: "ana@example.test", Role: auth.RolePatient, PasswordHash: "hashed:" + strings.ToUpper(password)}
}

func TestService_ReadsAndWritesAccountsInTheAuditedTransaction(t *testing.T) {
	tests := []struct {
		call        func(f fixture) error
		name        string
		wantOutside []string
	}{
		{name: "Register", call: func(f fixture) error {
			_, err := f.svc.Register(t.Context(), "cleo@example.test", password)
			return err
		}},
		{name: "Me", call: func(f fixture) error { _, err := f.svc.Me(t.Context(), accountID); return err }},
		// The account is read once before the transaction, only to verify the password without holding a
		// connection; the decision rests on the read inside it.
		{name: "Login", call: func(f fixture) error {
			_, err := f.svc.Login(t.Context(), "ana@example.test", password, clientIP)
			return err
		}, wantOutside: []string{"ByEmail"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(patientWithoutMFA())
			if err := tt.call(f); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(f.repo.outside, tt.wantOutside) {
				t.Errorf("repository calls outside the transaction: %v, want %v", f.repo.outside, tt.wantOutside)
			}
		})
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
