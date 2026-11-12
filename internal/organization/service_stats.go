package organization

import (
	"context"
	"fmt"
)

// Stats serves the clinics' locations to the aggregate builds (docs/plans/analytics.md §3), on salus_stats.
type Stats struct {
	repo StatsRepository
}

// NewStats returns the Stats on repo.
func NewStats(repo StatsRepository) *Stats { return &Stats{repo: repo} }

// Locations returns every clinic with its time zone, by id.
func (s *Stats) Locations(ctx context.Context) ([]Location, error) {
	l, err := s.repo.Locations(ctx)
	if err != nil {
		return nil, fmt.Errorf("organization: locations: %w", err)
	}
	return l, nil
}
