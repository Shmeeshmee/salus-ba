package practitionerrole

import (
	"context"
	"fmt"
)

// Stats serves the roles' specialties to the aggregate builds (docs/plans/analytics.md §3), on salus_stats.
type Stats struct {
	repo StatsRepository
}

// NewStats returns the Stats on repo.
func NewStats(repo StatsRepository) *Stats { return &Stats{repo: repo} }

// Specialties returns a clinic's roles with their specialty's SNOMED CT code, empty for none, by role id.
func (s *Stats) Specialties(ctx context.Context, orgID string) (map[string]string, error) {
	m, err := s.repo.Specialties(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("practitionerrole: specialties: %w", err)
	}
	return m, nil
}
