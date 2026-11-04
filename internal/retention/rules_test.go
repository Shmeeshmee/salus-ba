package retention_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/retention"
)

func TestRules_AreCompleteAndNameEachTableOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range retention.Rules() {
		if r.Table == "" || r.Basis == "" || r.Then == "" {
			t.Errorf("rule %+v: want a table, a basis and what happens then", r)
		}
		// A row kept while its subject exists has no period; every other rule has a positive one.
		if (r.Then == retention.ThenKeep) != (r.Keep == 0) || r.Keep < 0 {
			t.Errorf("rule %+v: want no period exactly when the row is kept, a positive one otherwise", r)
		}
		if seen[r.Table] {
			t.Errorf("table %s has two rules", r.Table)
		}
		seen[r.Table] = true
	}
}

func TestRules_KeepAuditEventsTenYearsThenRemoveThemByRange(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table != "audit_event" {
			continue
		}
		if tenYears := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC).Sub(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); r.Keep < tenYears {
			t.Errorf("audit_event kept %v, want at least ten calendar years (%v)", r.Keep, tenYears)
		}
		if r.Then != retention.ThenRemoveRange {
			t.Errorf("audit_event then %q, want %q", r.Then, retention.ThenRemoveRange)
		}
		return
	}
	t.Fatal("no rule for audit_event")
}

func TestRules_KeepAccountsWhileTheyExist(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "account" {
			if r.Then != retention.ThenKeep {
				t.Errorf("account then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for account")
}
