package retention

import (
	"errors"
	"fmt"
)

// ErrNoApplier is returned by NewService when a rule that removes rows (delete, remove-range) has no applier,
// or an applier is given for a table whose rule is not delete: the job refuses to start rather than skip a
// rule or apply one to the wrong table.
var ErrNoApplier = errors.New("retention: a rule that removes rows has no applier, or an applier has no such rule")

// OverdueError reports rows past their period that nothing can remove yet (audit_event before its range
// removal exists). It carries the table and the count, never the rows.
type OverdueError struct {
	Table string
	Rows  int64
}

func (e *OverdueError) Error() string {
	return fmt.Sprintf("retention: %d %s rows are past their period and cannot be removed yet", e.Rows, e.Table)
}
