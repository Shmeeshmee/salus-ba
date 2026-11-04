package retention_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/retention"
)

func TestRules_AreCompleteAndNameEachTableOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range retention.Rules() {
		if r.Table == "" || r.Basis == "" || r.Keep <= 0 || r.Then == "" {
			t.Errorf("rule %+v: want a table, a basis, a positive period and what happens then", r)
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
