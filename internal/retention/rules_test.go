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
		// A row kept while its subject exists, or restricted once it is no longer active, has no period; every
		// other rule has a positive one.
		noPeriod := r.Then == retention.ThenKeep || r.Then == retention.ThenRestrict
		if noPeriod != (r.Keep == 0) || r.Keep < 0 {
			t.Errorf("rule %+v: want no period exactly when the row is kept or restricted, a positive one otherwise", r)
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

func TestRules_DeleteSessionsThirtyDaysAfterTheyEnd(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "session" {
			if r.Then != retention.ThenDelete || r.Keep != 30*24*time.Hour {
				t.Errorf("session kept %v then %q, want 30 days then %q", r.Keep, r.Then, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for session")
}

func TestRules_RestrictPractitionersAndTheirRolesAfterTheyLeave(t *testing.T) {
	want := map[string]bool{"practitioner": true, "practitioner_role": true}
	for _, r := range retention.Rules() {
		if !want[r.Table] {
			continue
		}
		if r.Then != retention.ThenRestrict {
			t.Errorf("%s then %q, want %q", r.Table, r.Then, retention.ThenRestrict)
		}
		delete(want, r.Table)
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}

func TestRules_DeleteInvitesThirtyDaysAfterTheyEnd(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "invite" {
			if r.Then != retention.ThenDelete || r.Keep != 30*24*time.Hour {
				t.Errorf("invite kept %v then %q, want 30 days then %q", r.Keep, r.Then, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for invite")
}

func TestRules_KeepPatientsWhileTheAccountExists(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "patient" {
			if r.Then != retention.ThenKeep {
				t.Errorf("patient then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for patient")
}

func TestRules_KeepConsentsWithTheClinicalRecord(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "consent" {
			if r.Then != retention.ThenKeep {
				t.Errorf("consent then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for consent")
}

func TestRules_KeepAppointmentsWithTheClinicalRecord(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "appointment" {
			if r.Then != retention.ThenKeep {
				t.Errorf("appointment then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for appointment")
}

func TestRules_RestrictSchedulesAndDeleteFreeSlotsThirtyDaysAfterTheyEnd(t *testing.T) {
	want := map[string]retention.Rule{
		"schedule": {Table: "schedule", Then: retention.ThenRestrict},
		"slot":     {Table: "slot", Then: retention.ThenDelete, Keep: 30 * 24 * time.Hour},
	}
	for _, r := range retention.Rules() {
		if w, ok := want[r.Table]; ok {
			if r.Then != w.Then || r.Keep != w.Keep {
				t.Errorf("%s then %q after %v, want %q after %v", r.Table, r.Then, r.Keep, w.Then, w.Keep)
			}
			delete(want, r.Table)
		}
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}
