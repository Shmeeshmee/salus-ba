package trends

import (
	"context"
	"time"
)

// Repository stores the built periods, their cells and their rise flags. Nothing in it is ever changed: a period is
// written once with its cells (A3), and only the retention job deletes cells and flags.
type Repository interface {
	// Built reports whether the period has been built.
	Built(ctx context.Context, period string) (bool, error)
	// SavePeriod stores a built period, or returns ErrFrozen when it was built already.
	SavePeriod(ctx context.Context, p Period) error
	// SaveCells stores the period's cells.
	SaveCells(ctx context.Context, period string, rows []Row) error
	// SaveFlags stores the week's rise flags.
	SaveFlags(ctx context.Context, period string, flags []Flag) error
	// Periods returns the built periods of a kind, oldest first; a month or quarter whose cells are deleted is left out.
	Periods(ctx context.Context, kind string) ([]Period, error)
	// Cells returns a table's cells of the periods from to to, inclusive.
	Cells(ctx context.Context, table, from, to string) ([]Row, error)
	// Flags returns the rise flags of the weeks.
	Flags(ctx context.Context, weeks []string) ([]Flag, error)
	// DeleteCellsBefore and DeleteFlagsBefore delete the cells and the flags built before before, and count them
	// (the retention job, on salus_retention).
	DeleteCellsBefore(ctx context.Context, before time.Time) (int64, error)
	DeleteFlagsBefore(ctx context.Context, before time.Time) (int64, error)
}
