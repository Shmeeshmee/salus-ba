package worker

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// chatPartitionsHour is the UTC hour of the daily communication_partitions job: before the retention job, well
// before a month ends.
const chatPartitionsHour = 2

// chatPartitionsWorker keeps the chat's monthly message partitions ahead (ADR 0012).
type chatPartitionsWorker struct {
	river.WorkerDefaults[communication.PartitionsArgs]
	chat *communication.Service
}

func (w *chatPartitionsWorker) Work(ctx context.Context, _ *river.Job[communication.PartitionsArgs]) error {
	return w.chat.EnsurePartitions(ctx)
}

// newChatPartitions is the chat's service on pool (salus_app) for the partitions job alone: it needs only the
// repository and the clock.
func newChatPartitions(pool *pgxpool.Pool, clk clock.Clock) *communication.Service {
	return communication.NewService(communication.Ports{Repo: communication.NewPostgresRepository(pool), Clock: clk})
}
