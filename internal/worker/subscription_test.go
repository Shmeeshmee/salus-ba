package worker_test

import (
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/subscription"
)

// The monthly job, as the app role: one invoice per clinic and month however often it runs, for the clinic's active
// roles at its plan's price; the trial ends with it, audited; nothing but counts is logged.
func TestWorker_IssuesEachClinicsMonthlyInvoiceOnce(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	const clinic, other = "0a000000-0000-4000-8000-0000000000d1", "0b000000-0000-4000-8000-0000000000d2"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic D1'), ($2, 'Synthetic clinic D2')", clinic, other)
	exec("INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)")
	exec(`INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES
		($1, 'basic', 'trial', '2026-11-20T00:00:00Z'), ($2, 'basic', 'trial', '2027-06-01T00:00:00Z')`, clinic, other)
	for i, active := range []bool{true, true, false} {
		var practitioner string
		if err := db.owner.QueryRow(t.Context(), "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Synthetic', $2) RETURNING id", clinic, "Doctor "+string(rune('A'+i))).Scan(&practitioner); err != nil {
			t.Fatal(err)
		}
		exec("INSERT INTO practitioner_role (organization_id, practitioner_id, active) VALUES ($1, $2, $3)", clinic, practitioner, active)
	}
	month := time.Date(2026, 12, 1, 6, 0, 0, 0, time.UTC)
	for _, args := range []subscription.MonthlyArgs{{Month: month}, {Month: month}, {Month: month, OrganizationID: clinic}} {
		if kind, logs := runJobOn(t, db, liveKit(t), testFiles(t), args, nil); kind != river.EventKindJobCompleted {
			t.Fatalf("job %+v ended %s: %s", args, kind, logs)
		}
	}
	var n, seats int
	var amount int64
	var start time.Time
	if err := db.owner.QueryRow(t.Context(), "SELECT count(*), min(seats), min(amount_minor), min(period_start) FROM subscription_invoice WHERE organization_id = $1", clinic).Scan(&n, &seats, &amount, &start); err != nil {
		t.Fatal(err)
	}
	if n != 1 || seats != 2 || amount != 20000 || start.Format(time.DateOnly) != "2026-12-01" {
		t.Errorf("%d invoices with %d seats, %d, from %s; want one for December with 2 seats at 200.00", n, seats, amount, start.Format(time.DateOnly))
	}
	var status string
	if err := db.owner.QueryRow(t.Context(), "SELECT status FROM clinic_subscription WHERE organization_id = $1", clinic).Scan(&status); err != nil || status != "active" {
		t.Errorf("status %q, %v, want active after the first invoice", status, err)
	}
	if err := db.owner.QueryRow(t.Context(), "SELECT count(*) FROM subscription_invoice WHERE organization_id = $1", other).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d invoices for the clinic still in its trial, %v, want none", n, err)
	}
	if err := db.owner.QueryRow(t.Context(), "SELECT count(*) FROM audit_event WHERE agent_name = 'subscription_monthly' AND organization_id = $1", clinic).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d audit events of the job, %v, want one: the trial's end", n, err)
	}
}
