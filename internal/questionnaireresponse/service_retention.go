package questionnaireresponse

import (
	"context"
	"time"
)

// Purger deletes the responses whose retention period is over (internal/retention: an unsubmitted one, in
// progress or withdrawn, once its due time has passed; a completed one is kept with the clinical record), in
// every clinic. It is the retention job's applier for questionnaire_response, built on the retention role's
// pool in internal/worker.
type Purger struct {
	repo Repository
}

// NewPurger returns the purger of questionnaire_response on repo.
func NewPurger(repo Repository) *Purger {
	return &Purger{repo: repo}
}

// Purge deletes the unsubmitted responses due before before and returns how many.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteUnsubmittedBefore(ctx, before)
}
