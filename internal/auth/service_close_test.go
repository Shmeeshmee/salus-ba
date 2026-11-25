package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

const eraseAfter = 21 * 24 * time.Hour

func TestClose_EndsSignInAndSessionsAndSetsTheErasureDate(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	f.repo.sessions["h"] = &fakeSession{Session: auth.Session{ID: "s1", AccountID: accountID}}

	if err := f.svc.Close(t.Context(), accountID, password, eraseAfter); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	a := f.repo.accounts[accountID]
	if !a.ClosedAt.Equal(now) || !a.EraseAfter.Equal(now.Add(eraseAfter)) {
		t.Errorf("closed %v, erase after %v; want %v and %v", a.ClosedAt, a.EraseAfter, now, now.Add(eraseAfter))
	}
	if f.repo.sessions["h"].revokedAt.IsZero() {
		t.Error("the open session was not revoked")
	}
	if e := f.onlyEvent(t); e.Subtype != update || e.EntityReference != "Account/"+accountID {
		t.Errorf("audit event %+v, want an update of the account", e)
	}
	if _, err := f.svc.Login(t.Context(), "ana@example.test", password, "192.0.2.1", "", ""); !errors.Is(err, auth.ErrWrongCredentials) {
		t.Errorf("Login() after closure error = %v, want ErrWrongCredentials", err)
	}
}

func TestClose_Refuses(t *testing.T) {
	staff := patientWithoutMFA()
	staff.Role, staff.OrganizationID = auth.RoleAdmin, clinicID
	closed := patientWithoutMFA()
	closed.ClosedAt, closed.EraseAfter = now.Add(-time.Hour), now.Add(eraseAfter)
	tests := []struct {
		want     error
		name     string
		password string
		account  auth.Account
	}{
		{name: "a wrong password", account: patientWithoutMFA(), password: "not the password at all", want: auth.ErrWrongCredentials},
		{name: "a staff account", account: staff, password: password, want: auth.ErrNotFound},
		{name: "a closed account", account: closed, password: password, want: auth.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tt.account)
			if err := f.svc.Close(t.Context(), accountID, tt.password, eraseAfter); !errors.Is(err, tt.want) {
				t.Fatalf("Close() error = %v, want %v", err, tt.want)
			}
			if a := f.repo.accounts[accountID]; !a.ClosedAt.Equal(tt.account.ClosedAt) {
				t.Errorf("closed_at changed to %v", a.ClosedAt)
			}
		})
	}
}

func TestReopen_ClearsTheClosureBeforeTheErasureDate(t *testing.T) {
	closed := patientWithoutMFA()
	closed.ClosedAt, closed.EraseAfter = now.Add(-time.Hour), now.Add(time.Hour)
	f := newFixture(closed)

	if err := f.svc.Reopen(t.Context(), accountID); err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}
	if a := f.repo.accounts[accountID]; !a.ClosedAt.IsZero() || !a.EraseAfter.IsZero() {
		t.Errorf("closed %v, erase after %v; want both cleared", a.ClosedAt, a.EraseAfter)
	}
	if e := f.onlyEvent(t); e.Subtype != update || e.AgentName == "" {
		t.Errorf("audit event %+v, want the operator's update", e)
	}
}

func TestReopen_RefusesAnOpenOrDueAccount(t *testing.T) {
	due := patientWithoutMFA()
	due.ClosedAt, due.EraseAfter = now.Add(-eraseAfter), now
	for name, a := range map[string]auth.Account{"open": patientWithoutMFA(), "due": due} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(a)
			if err := f.svc.Reopen(t.Context(), accountID); !errors.Is(err, auth.ErrNotFound) {
				t.Errorf("Reopen() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestErase_BlanksTheLoginOnceAndOnlyWhenDue(t *testing.T) {
	due := patientWithoutMFA()
	due.ClosedAt, due.EraseAfter, due.Language, due.DetectedLanguage = now.Add(-eraseAfter), now, "bs", "de"
	due.MFAEnrolledAt = now.Add(-48 * time.Hour)
	f := newFixture(due)
	f.repo.secrets[accountID] = []byte("sealed")

	ids, err := f.svc.DueForErasure(t.Context())
	if err != nil || len(ids) != 1 || ids[0] != accountID {
		t.Fatalf("DueForErasure() = %v, %v; want [%s]", ids, err, accountID)
	}
	if err := f.svc.Erase(t.Context(), accountID); err != nil {
		t.Fatalf("Erase() error = %v", err)
	}
	a := f.repo.accounts[accountID]
	if a.Email != accountID+"@erased.invalid" || a.PasswordHash != "" || a.Language != "" || a.DetectedLanguage != "" ||
		!a.MFAEnrolledAt.IsZero() || f.repo.secrets[accountID] != nil || !a.ErasedAt.Equal(now) {
		t.Errorf("account after erasure %+v, want the login, password, MFA and languages blanked", a)
	}
	if ids, _ := f.svc.DueForErasure(t.Context()); len(ids) != 0 {
		t.Errorf("DueForErasure() after erasure = %v, want none", ids)
	}
	if err := f.svc.Erase(t.Context(), accountID); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("a second Erase() error = %v, want ErrNotFound", err)
	}
}

func TestErase_RefusesAnAccountNotYetDue(t *testing.T) {
	closed := patientWithoutMFA()
	closed.ClosedAt, closed.EraseAfter = now.Add(-time.Hour), now.Add(time.Hour)
	f := newFixture(closed)
	if err := f.svc.Erase(t.Context(), accountID); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("Erase() error = %v, want ErrNotFound", err)
	}
	if a := f.repo.accounts[accountID]; a.Email != closed.Email {
		t.Errorf("e-mail changed to %q", a.Email)
	}
}
