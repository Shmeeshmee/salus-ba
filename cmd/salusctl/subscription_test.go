package main

import (
	"bytes"
	"strings"
	"testing"
)

// The owner's membership operations: plans, the trial with the clinic's creation, marking an invoice paid (which the
// app may not) and moving a status forward only; each move audited as salusctl.
func TestSubscription_TheOwnerSetsPlansMarksPaidAndMovesStatusesForward(t *testing.T) {
	db, count := ownerDB(t)
	runs := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := run(t.Context(), args, &out); err != nil {
			t.Fatalf("run(%q): %v", args, err)
		}
		return out.String()
	}
	create := []string{"create-clinic", "-name", "Synthetic clinic", "-admin-email", "dee@example.test"}
	if out := runs(create...); !strings.Contains(out, "trial none") || count(`SELECT count(*) FROM clinic_subscription`) != 0 {
		t.Errorf("before a plan is set: %q, want the clinic without a trial", out)
	}
	runs("subscription", "plan", "-code", "basic", "-name", "Basic", "-price-minor", "10000", "-features", "video,chat")
	runs("subscription", "plan", "-code", "basic", "-name", "Basic", "-price-minor", "12000")
	if n := count(`SELECT count(*) FROM subscription_plan WHERE code = 'basic' AND price_per_seat_minor = 12000 AND currency = 'BAM' AND active`); n != 1 {
		t.Errorf("%d basic plans at the new price, want 1", n)
	}
	create[2], create[4] = "Synthetic clinic two", "eli@example.test"
	if out := runs(create...); !strings.Contains(out, "trial started") {
		t.Errorf("create-clinic = %q, want the trial started", out)
	}
	var org string
	if err := db.QueryRow(t.Context(), `SELECT organization_id::text FROM clinic_subscription WHERE status = 'trial' AND trial_ends_at > now() + interval '29 days'`).Scan(&org); err != nil {
		t.Fatalf("the new clinic's trial: %v", err)
	}
	var first string
	if err := db.QueryRow(t.Context(), `SELECT id::text FROM organization WHERE id <> $1`, org).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if out := runs("subscription", "start-trial", first); !strings.Contains(out, "trial on plan basic") {
		t.Errorf("start-trial = %q", out)
	}
	if _, err := db.Exec(t.Context(), `INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats, amount_minor, issued_at, due_at)
		VALUES ($1, 1, '2026-10-01', '2026-11-01', 2, 24000, '2026-10-01T00:00:00Z', '2026-10-15T00:00:00Z')`, org); err != nil {
		t.Fatal(err)
	}
	if out := runs("subscription", "status", org, "suspended"); !strings.Contains(out, "suspended") {
		t.Errorf("status = %q", out)
	}
	if out := runs("subscription", "mark-paid", "1"); !strings.Contains(out, "invoice 1") {
		t.Errorf("mark-paid = %q", out)
	}
	if n := count(`SELECT count(*) FROM subscription_invoice i JOIN clinic_subscription s USING (organization_id)
		WHERE i.status = 'paid' AND i.paid_at IS NOT NULL AND s.status = 'active'`); n != 1 {
		t.Errorf("%d paid invoices of an active clinic, want 1: the payment lifts the stop", n)
	}
	// Two trials, the move to suspended, the payment and the return to active.
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND entity_reference LIKE 'Organization/%' AND subtype_code IN ('create', 'update') AND organization_id IN ($1, $2)`, org, first); n < 5 {
		t.Errorf("%d audit events of the membership, want at least 5", n)
	}
	for _, args := range [][]string{
		{"subscription"}, {"subscription", "status", org, "trial"}, {"subscription", "status", org, "paused"}, {"subscription", "mark-paid", "1"},
		{"subscription", "mark-paid", "x"}, {"subscription", "plan", "-code", "Bad code", "-name", "x", "-price-minor", "1"},
		{"subscription", "plan", "-code", "pro", "-name", "Pro"}, {"subscription", "issue"},
	} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%q): error = nil", args)
		}
	}
}
