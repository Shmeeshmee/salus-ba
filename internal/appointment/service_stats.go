package appointment

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/pkg/pseudonym"
)

// Stats serves the appointments to the aggregate builds (docs/plans/analytics.md §3, ADR 0013), on the
// salus_stats role. No access decision: only the builds in cmd/worker hold that role, and they audit their build.
type Stats struct {
	repo StatsRepository
}

// NewStats returns the Stats on repo.
func NewStats(repo StatsRepository) *Stats { return &Stats{repo: repo} }

// Starts returns a clinic's appointments starting in [from, to), oldest first, each patient as HMAC-SHA256 of
// their id under key (zero without a key, nil; ErrShortKey for a key shorter than 32 bytes).
func (s *Stats) Starts(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]Start, error) {
	if key != nil && len(key) < pseudonym.MinKey {
		return nil, ErrShortKey
	}
	rows, err := s.repo.Booked(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("appointment: starts: %w", err)
	}
	return starts(rows, key), nil
}

// Fulfilled is Starts for the fulfilled appointments only (the platform heatmap, docs/plans/analytics.md §5): the
// others are not read.
func (s *Stats) Fulfilled(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]Start, error) {
	if key != nil && len(key) < pseudonym.MinKey {
		return nil, ErrShortKey
	}
	rows, err := s.repo.Fulfilled(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("appointment: fulfilled: %w", err)
	}
	return starts(rows, key), nil
}

// starts maps the rows, each patient as HMAC-SHA256 of their id under key (zero without a key).
func starts(rows []Booked, key []byte) []Start {
	out := make([]Start, len(rows))
	for i, b := range rows {
		out[i] = Start{Start: b.Start, PractitionerRoleID: b.PractitionerRoleID, Status: b.Status}
		if key != nil {
			out[i].Patient = pseudonym.Of(key, b.PatientID)
		}
	}
	return out
}
