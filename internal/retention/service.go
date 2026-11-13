package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Purger deletes the rows of one table whose period ended before before, and counts them. The owning
// domain implements it (auth.SessionPurger), so the SQL stays with the table's owner; internal/worker builds
// it on the retention role's pool.
type Purger interface {
	Purge(ctx context.Context, before time.Time) (int64, error)
}

// Audit counts the audit events recorded before a cut-off and records the job's own events
// (auditevent.Service).
type Audit interface {
	CountRecordedBefore(ctx context.Context, before time.Time) (int64, error)
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// agent names the job as the agent of its audit events.
const agent = "retention_apply"

// auditTable is the one table removed by range; its applier is the overdue check through Audit.
const auditTable = "audit_event"

// Service applies the rules (docs/booking/SPEC.md §2 "Applying the rules"): keep and restrict apply nothing
// (closure, erasure and deactivation are later features), delete calls the table's Purger, and remove-range
// checks that no audit event is past its period, since nothing can remove one yet.
type Service struct {
	purgers map[string]Purger
	audit   Audit
	clock   clock.Clock
	policy  Policy
}

// NewService returns the Service on the policy's rules with a Purger per delete rule's table and the audit check;
// ErrClinicalDeletion for a policy that does not keep clinical data, ErrNoApplier naming the table when a rule that
// removes rows has no applier or a purger has no delete rule.
func NewService(policy Policy, purgers map[string]Purger, audit Audit, clk clock.Clock) (*Service, error) {
	if policy.ClinicalData != ThenKeep {
		return nil, ErrClinicalDeletion
	}
	deletes := map[string]bool{}
	for _, r := range Rules(policy) {
		switch {
		case r.Then == ThenDelete && purgers[r.Table] == nil,
			r.Then == ThenRemoveRange && (r.Table != auditTable || audit == nil):
			return nil, fmt.Errorf("%w: %s", ErrNoApplier, r.Table)
		}
		deletes[r.Table] = r.Then == ThenDelete
	}
	for table := range purgers {
		if !deletes[table] {
			return nil, fmt.Errorf("%w: %s", ErrNoApplier, table)
		}
	}
	return &Service{purgers: purgers, audit: audit, clock: clk, policy: policy}, nil
}

// Apply applies every rule once, in the order of Rules, and returns a Result per delete and remove-range
// rule, and records one AuditEvent per delete rule's table with the rows deleted, never one per row. An
// applier's or the audit's error stops it; overdue audit events do not stop the purges but fail the run with
// an OverdueError, so the job alerts. The purge and its event are not one transaction (the purge runs as
// salus_retention, the event as salus_app): a run that fails between them is retried and audits again.
func (s *Service) Apply(ctx context.Context) ([]Result, error) {
	now := s.clock.Now()
	var results []Result
	var overdue error
	for _, r := range Rules(s.policy) {
		before := now.Add(-r.Keep)
		switch r.Then {
		case ThenDelete:
			n, err := s.purgers[r.Table].Purge(ctx, before)
			if err != nil {
				return results, fmt.Errorf("retention: purge %s: %w", r.Table, err)
			}
			if err := s.audit.Record(ctx, deleted(r.Table, n)); err != nil {
				return results, fmt.Errorf("retention: audit %s: %w", r.Table, err)
			}
			results = append(results, Result{Table: r.Table, Deleted: n})
		case ThenRemoveRange:
			n, err := s.audit.CountRecordedBefore(ctx, before)
			if err != nil {
				return results, fmt.Errorf("retention: check %s: %w", r.Table, err)
			}
			results = append(results, Result{Table: r.Table, Overdue: n})
			if n > 0 {
				overdue = &OverdueError{Table: r.Table, Rows: n}
			}
		case ThenKeep, ThenRestrict:
		}
	}
	return results, overdue
}

// deleted is the audit event of a run's deletion of n rows from table, by the job.
func deleted(table string, n int64) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:        auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:     auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeDelete},
		Purpose:     auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		AgentName:   agent,
		EntityName:  table,
		EntityCount: n,
		Action:      auditevent.ActionDelete,
		Outcome:     auditevent.OutcomeSuccess,
	}
}
