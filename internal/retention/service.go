package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Purger deletes the rows of one table whose period ended before before, and counts them. The owning
// domain implements it (auth.SessionPurger), so the SQL stays with the table's owner; internal/worker builds
// it on the retention role's pool.
type Purger interface {
	Purge(ctx context.Context, before time.Time) (int64, error)
}

// AuditAges counts the audit events recorded before a cut-off (auditevent.Service).
type AuditAges interface {
	CountRecordedBefore(ctx context.Context, before time.Time) (int64, error)
}

// auditTable is the one table removed by range; its applier is the overdue check through AuditAges.
const auditTable = "audit_event"

// Service applies the rules (docs/booking/SPEC.md §2 "Applying the rules"): keep and restrict apply nothing
// (closure, erasure and deactivation are later features), delete calls the table's Purger, and remove-range
// checks that no audit event is past its period, since nothing can remove one yet.
type Service struct {
	purgers map[string]Purger
	audit   AuditAges
	clock   clock.Clock
}

// NewService returns the Service with a Purger per delete rule's table and the audit check, or ErrNoApplier
// naming the table when a rule that removes rows has no applier or a purger has no delete rule.
func NewService(purgers map[string]Purger, audit AuditAges, clk clock.Clock) (*Service, error) {
	deletes := map[string]bool{}
	for _, r := range Rules() {
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
	return &Service{purgers: purgers, audit: audit, clock: clk}, nil
}

// Apply applies every rule once, in the order of Rules, and returns a Result per delete and remove-range
// rule. An applier's error stops it; overdue audit events do not stop the purges but fail the run with an
// OverdueError, so the job alerts.
func (s *Service) Apply(ctx context.Context) ([]Result, error) {
	now := s.clock.Now()
	var results []Result
	var overdue error
	for _, r := range Rules() {
		before := now.Add(-r.Keep)
		switch r.Then {
		case ThenDelete:
			n, err := s.purgers[r.Table].Purge(ctx, before)
			if err != nil {
				return results, fmt.Errorf("retention: purge %s: %w", r.Table, err)
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
