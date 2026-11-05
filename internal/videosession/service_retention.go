package videosession

import (
	"context"
	"time"
)

// Purger deletes the sessions whose retention period is over (internal/retention: 30 days after the end, or
// after the window's close for a session that never ended), in every clinic. It is the retention job's
// applier for video_session, built on the retention role's pool in internal/worker.
type Purger struct {
	repo Repository
}

// NewPurger returns the purger of video_session on repo.
func NewPurger(repo Repository) *Purger {
	return &Purger{repo: repo}
}

// Purge deletes the sessions that ended, or whose window closed unended, before before and returns how many.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteEndedBefore(ctx, before)
}
