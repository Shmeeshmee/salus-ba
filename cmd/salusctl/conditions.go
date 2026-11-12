package main

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// conditionsCommand runs `salusctl conditions unconfirmed`: the ids of the Conditions backfilled from documents
// signed before the confirmation existed, which the analytics do not count (docs/plans/analytics.md §2).
func (s services) conditionsCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) != 1 || args[0] != "unconfirmed" {
		return errors.New(usage)
	}
	ids, err := s.conditions.Unconfirmed(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := fmt.Fprintln(stdout, id); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(stdout, "%d unconfirmed Condition(s)\n", len(ids))
	return err
}
