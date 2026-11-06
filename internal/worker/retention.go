package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// RetentionApplyArgs applies every retention rule (internal/retention). The scheduler enqueues it daily; it
// has no arguments.
type RetentionApplyArgs struct{}

// Kind names the job in the river_job table.
func (RetentionApplyArgs) Kind() string { return "retention_apply" }

// retentionWorker runs the retention service and logs, per rule that removes rows, the table and its counts:
// nothing else.
type retentionWorker struct {
	river.WorkerDefaults[RetentionApplyArgs]
	rules *retention.Service
	log   *slog.Logger
}

func (w *retentionWorker) Work(ctx context.Context, _ *river.Job[RetentionApplyArgs]) error {
	results, err := w.rules.Apply(ctx)
	for _, r := range results {
		w.log.InfoContext(ctx, "retention applied", "table", r.Table, "deleted", r.Deleted, "overdue", r.Overdue)
	}
	return err
}

// newRetention builds the retention service: the purgers of the delete rules on retentionPool (salus_retention
// may delete those tables' rows, past row-level security), and the audit check on pool (salus_app may read
// audit_event; salus_retention may not). It fails when a rule has no applier.
func newRetention(pool, retentionPool *pgxpool.Pool) (*retention.Service, error) {
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), postgres.NewTransactor(pool), clock.System{})
	accounts := auth.NewPostgresRepository(retentionPool)
	return retention.NewService(map[string]retention.Purger{
		"session":                auth.NewSessionPurger(accounts),
		"invite":                 auth.NewInvitePurger(accounts),
		"slot":                   slot.NewPurger(slot.NewPostgresRepository(retentionPool)),
		"video_session":          videosession.NewPurger(videosession.NewPostgresRepository(retentionPool)),
		"questionnaire_response": questionnaireresponse.NewPurger(questionnaireresponse.NewPostgresRepository(retentionPool)),
	}, audit, clock.System{})
}
