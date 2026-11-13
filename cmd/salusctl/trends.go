package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Shmeeshmee/salus/internal/trends"
)

// trendsCommand runs `salusctl trends build <period>`: a missed build of a closed month (2026-10), quarter
// (2026-Q3) or ISO week (2026-W44) of the platform figures, on the owner's connection, audited as salusctl
// (docs/plans/analytics.md §5). Refused while ANALYTICS_PLATFORM_ENABLED is off (§2a), and for a period built
// already (it is frozen). It prints counts only (A13).
func (s services) trendsCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) != 2 || args[0] != "build" {
		return errors.New(usage)
	}
	if s.trends == nil {
		return errors.New("trends: ANALYTICS_PLATFORM_ENABLED is off; the platform figures are not built")
	}
	sum, err := s.trends.Build(ctx, args[1])
	if err != nil {
		return err
	}
	status := trends.StatusPublished
	if sum.Refused {
		status = trends.StatusRefused
	}
	_, err = fmt.Fprintf(stdout, "%s %s: %d cells, %d suppressed, %d rises, code level dropped %v\n",
		args[1], status, sum.Cells, sum.Suppressed, sum.Rises, sum.CodeDropped)
	return err
}
