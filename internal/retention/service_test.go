package retention_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

var now = time.Date(2026, 11, 5, 3, 0, 0, 0, time.UTC)

const day = 24 * time.Hour

// fakePurger records the cut-off of each call and returns rows.
type fakePurger struct {
	err    error
	before []time.Time
	rows   int64
}

func (p *fakePurger) Purge(_ context.Context, before time.Time) (int64, error) {
	p.before = append(p.before, before)
	return p.rows, p.err
}

// fakeAudit counts overdue audit events: overdue for any cut-off.
type fakeAudit struct {
	err     error
	before  []time.Time
	overdue int64
}

func (a *fakeAudit) CountRecordedBefore(_ context.Context, before time.Time) (int64, error) {
	a.before = append(a.before, before)
	return a.overdue, a.err
}

type appliers struct {
	sessions, invites, slots *fakePurger
	audit                    *fakeAudit
}

func newAppliers() appliers {
	return appliers{sessions: &fakePurger{rows: 4}, invites: &fakePurger{rows: 1}, slots: &fakePurger{rows: 9}, audit: &fakeAudit{}}
}

func (a appliers) purgers() map[string]retention.Purger {
	return map[string]retention.Purger{"session": a.sessions, "invite": a.invites, "slot": a.slots}
}

func (a appliers) service(t *testing.T) *retention.Service {
	t.Helper()
	svc, err := retention.NewService(a.purgers(), a.audit, clock.Fixed{T: now})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func TestApply_CallsEachApplierOnceWithItsRulesCutOffAndCountsTheRows(t *testing.T) {
	a := newAppliers()

	results, err := a.service(t).Apply(t.Context())

	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if want := []time.Time{now.Add(-30 * day)}; !slices.Equal(a.sessions.before, want) || !slices.Equal(a.invites.before, want) ||
		!slices.Equal(a.slots.before, want) {
		t.Errorf("purged sessions before %v, invites before %v and slots before %v, want once each before %v",
			a.sessions.before, a.invites.before, a.slots.before, want)
	}
	if want := []time.Time{now.Add(-3653 * day)}; !slices.Equal(a.audit.before, want) {
		t.Errorf("counted audit events before %v, want once before %v", a.audit.before, want)
	}
	// keep and restrict apply nothing: account, practitioner, practitioner_role, patient, consent and schedule
	// have no result.
	want := []retention.Result{{Table: "audit_event"}, {Table: "session", Deleted: 4}, {Table: "invite", Deleted: 1}, {Table: "slot", Deleted: 9}}
	if !slices.Equal(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
}

func TestApply_FailsWithTheCountOnlyWhenAnAuditEventIsOverdueAndStillPurges(t *testing.T) {
	a := newAppliers()
	a.audit.overdue = 7

	results, err := a.service(t).Apply(t.Context())

	var overdue *retention.OverdueError
	if !errors.As(err, &overdue) || overdue.Table != "audit_event" || overdue.Rows != 7 {
		t.Fatalf("Apply() error = %v, want an OverdueError for 7 audit_event rows", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "7") || !strings.Contains(msg, "audit_event") {
		t.Errorf("error %q does not give the table and the count", msg)
	}
	if len(a.sessions.before) != 1 || len(a.invites.before) != 1 {
		t.Error("an overdue audit event stopped the purges")
	}
	if !slices.Contains(results, retention.Result{Table: "audit_event", Overdue: 7}) {
		t.Errorf("results %+v lack the overdue count", results)
	}
}

func TestApply_StopsAtAnApplierThatFails(t *testing.T) {
	for name, breakIt := range map[string]func(appliers){
		"purger": func(a appliers) { a.sessions.err = errors.New("synthetic failure") },
		"audit":  func(a appliers) { a.audit.err = errors.New("synthetic failure") },
	} {
		t.Run(name, func(t *testing.T) {
			a := newAppliers()
			breakIt(a)

			if _, err := a.service(t).Apply(t.Context()); err == nil || errors.As(err, new(*retention.OverdueError)) {
				t.Errorf("Apply() error = %v, want the applier's error", err)
			}
		})
	}
}

// The job refuses to start when a delete or remove-range rule has no applier, or an applier has no delete
// rule to apply.
func TestNewService_RefusesRulesWithoutAnApplierAndAppliersWithoutARule(t *testing.T) {
	purger := &fakePurger{}
	tests := []struct {
		purgers map[string]retention.Purger
		audit   retention.AuditAges
		name    string
		table   string
	}{
		{name: "no session purger", purgers: map[string]retention.Purger{"invite": purger, "slot": purger}, audit: &fakeAudit{}, table: "session"},
		{name: "no invite purger", purgers: map[string]retention.Purger{"session": purger, "slot": purger}, audit: &fakeAudit{}, table: "invite"},
		{name: "no slot purger", purgers: map[string]retention.Purger{"session": purger, "invite": purger}, audit: &fakeAudit{}, table: "slot"},
		{name: "no audit check", purgers: map[string]retention.Purger{"session": purger, "invite": purger, "slot": purger}, table: "audit_event"},
		{name: "a purger for a kept table", purgers: map[string]retention.Purger{"session": purger, "invite": purger, "slot": purger, "account": purger}, audit: &fakeAudit{}, table: "account"},
		{name: "a purger for a restricted table", purgers: map[string]retention.Purger{"session": purger, "invite": purger, "slot": purger, "schedule": purger}, audit: &fakeAudit{}, table: "schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := retention.NewService(tt.purgers, tt.audit, clock.Fixed{T: now})

			if !errors.Is(err, retention.ErrNoApplier) || !strings.Contains(err.Error(), tt.table) {
				t.Errorf("NewService() error = %v, want ErrNoApplier naming %s", err, tt.table)
			}
		})
	}
}

// Every delete and remove-range rule has an applier the service knows how to run: a purger for delete, and
// the audit check for audit_event, the only table removed by range.
func TestRules_EveryRuleThatRemovesRowsCanBeApplied(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Then == retention.ThenRemoveRange && r.Table != "audit_event" {
			t.Errorf("rule for %s removes by range; only audit_event has a remove-range applier", r.Table)
		}
	}
}
