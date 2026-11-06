package retention_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
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

// fakeAudit counts overdue audit events, overdue for any cut-off, and keeps the events recorded.
type fakeAudit struct {
	err       error
	recordErr error
	before    []time.Time
	events    []auditevent.AuditEvent
	overdue   int64
}

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	if a.recordErr != nil {
		return a.recordErr
	}
	a.events = append(a.events, e)
	return nil
}

func (a *fakeAudit) CountRecordedBefore(_ context.Context, before time.Time) (int64, error) {
	a.before = append(a.before, before)
	return a.overdue, a.err
}

type appliers struct {
	sessions, invites, slots, videoSessions, responses *fakePurger
	audit                                              *fakeAudit
}

func newAppliers() appliers {
	return appliers{sessions: &fakePurger{rows: 4}, invites: &fakePurger{rows: 1}, slots: &fakePurger{rows: 9}, videoSessions: &fakePurger{rows: 2},
		responses: &fakePurger{rows: 3}, audit: &fakeAudit{}}
}

func (a appliers) purgers() map[string]retention.Purger {
	return map[string]retention.Purger{"session": a.sessions, "invite": a.invites, "slot": a.slots, "video_session": a.videoSessions,
		"questionnaire_response": a.responses}
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
		!slices.Equal(a.slots.before, want) || !slices.Equal(a.videoSessions.before, want) {
		t.Errorf("purged sessions before %v, invites before %v and slots before %v, want once each before %v",
			a.sessions.before, a.invites.before, a.slots.before, want)
	}
	if want := []time.Time{now}; !slices.Equal(a.responses.before, want) {
		t.Errorf("purged questionnaire responses before %v, want once before %v", a.responses.before, want)
	}
	if want := []time.Time{now.Add(-3653 * day)}; !slices.Equal(a.audit.before, want) {
		t.Errorf("counted audit events before %v, want once before %v", a.audit.before, want)
	}
	// keep and restrict apply nothing: account, practitioner, practitioner_role, patient, consent and schedule
	// have no result.
	want := []retention.Result{{Table: "audit_event"}, {Table: "session", Deleted: 4}, {Table: "invite", Deleted: 1}, {Table: "slot", Deleted: 9}, {Table: "video_session", Deleted: 2},
		{Table: "questionnaire_response", Deleted: 3}}
	if !slices.Equal(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
}

// One AuditEvent per run per table rows are deleted from: the job as agent, the table and its row count,
// zero included; nothing per row (docs/booking/SPEC.md §2 "Applying the rules").
func TestApply_AuditsEachDeleteRuleOnceWithItsTableAndRowCount(t *testing.T) {
	a := newAppliers()
	a.slots.rows = 0

	if _, err := a.service(t).Apply(t.Context()); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	event := func(table string, rows int64) auditevent.AuditEvent {
		return auditevent.AuditEvent{
			Type:        auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:     auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeDelete},
			Purpose:     auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
			AgentName:   "retention_apply",
			EntityName:  table,
			EntityCount: rows,
			Action:      auditevent.ActionDelete,
			Outcome:     auditevent.OutcomeSuccess,
		}
	}
	want := []auditevent.AuditEvent{event("session", 4), event("invite", 1), event("slot", 0), event("video_session", 2), event("questionnaire_response", 3)}
	if !slices.Equal(a.audit.events, want) {
		t.Errorf("recorded %+v,\nwant %+v", a.audit.events, want)
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
		"record": func(a appliers) { a.audit.recordErr = errors.New("synthetic failure") },
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

// purgersBut is p for every table whose rule is delete but the table without, and for each extra table.
func purgersBut(p retention.Purger, without string, extra ...string) map[string]retention.Purger {
	out := map[string]retention.Purger{}
	for _, table := range append([]string{"session", "invite", "slot", "video_session", "questionnaire_response"}, extra...) {
		if table != without {
			out[table] = p
		}
	}
	return out
}

// The job refuses to start when a delete or remove-range rule has no applier, or an applier has no delete
// rule to apply.
func TestNewService_RefusesRulesWithoutAnApplierAndAppliersWithoutARule(t *testing.T) {
	purger := &fakePurger{}
	tests := []struct {
		purgers map[string]retention.Purger
		audit   retention.Audit
		name    string
		table   string
	}{
		{name: "no session purger", purgers: purgersBut(purger, "session"), audit: &fakeAudit{}, table: "session"},
		{name: "no invite purger", purgers: purgersBut(purger, "invite"), audit: &fakeAudit{}, table: "invite"},
		{name: "no slot purger", purgers: purgersBut(purger, "slot"), audit: &fakeAudit{}, table: "slot"},
		{name: "no video_session purger", purgers: purgersBut(purger, "video_session"), audit: &fakeAudit{}, table: "video_session"},
		{name: "no questionnaire_response purger", purgers: purgersBut(purger, "questionnaire_response"), audit: &fakeAudit{}, table: "questionnaire_response"},
		{name: "no audit check", purgers: purgersBut(purger, ""), table: "audit_event"},
		{name: "a purger for a kept table", purgers: purgersBut(purger, "", "account"), audit: &fakeAudit{}, table: "account"},
		{name: "a purger for a restricted table", purgers: purgersBut(purger, "", "schedule"), audit: &fakeAudit{}, table: "schedule"},
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
