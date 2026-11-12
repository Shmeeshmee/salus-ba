package worker

import (
	"log/slog"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/binary"
)

// River's reindexer runs REINDEX INDEX CONCURRENTLY, which only a table's owner may run; the worker logs in
// as salus_app, which owns nothing. Index maintenance on River's tables is an owner-side ops task.
func TestWorkConfig_ReindexesNoIndex(t *testing.T) {
	cfg := workConfig(slog.New(slog.DiscardHandler), river.QueueDefault, river.NewWorkers())

	if cfg.ReindexerIndexNames == nil || len(cfg.ReindexerIndexNames) != 0 {
		t.Errorf("ReindexerIndexNames = %#v, want an empty, non-nil list (nil means River's defaults)", cfg.ReindexerIndexNames)
	}
}

// River's leader runs the periodic jobs and the maintenance services; leadership is shared by every client on
// the database. The worker and the notifier never lead, so the scheduler, which alone has the periodic jobs,
// always does.
func TestWorkConfig_NeverLeads(t *testing.T) {
	cfg := workConfig(slog.New(slog.DiscardHandler), emailQueue, river.NewWorkers())

	if !cfg.LeaderElectionDisabled || len(cfg.PeriodicJobs) != 0 {
		t.Errorf("LeaderElectionDisabled = %v with %d periodic jobs, want true and none", cfg.LeaderElectionDisabled, len(cfg.PeriodicJobs))
	}
	if _, ok := cfg.Queues[emailQueue]; !ok || len(cfg.Queues) != 1 {
		t.Errorf("Queues = %v, want only %s", cfg.Queues, emailQueue)
	}
}

// cmd/worker works the default queue, the scan queue, which runs 2 files at a time: clamd holds each stream it
// scans (docs/attachments/SPEC.md §2 "Scan"), and the clinics' webhooks (SAL-29).
func TestWorkerQueues_ScanTwoFilesAtATime(t *testing.T) {
	q := workerQueues()

	if len(q) != 3 || q[river.QueueDefault].MaxWorkers != maxWorkers || q[binary.QueueScan].MaxWorkers != 2 || q[webhookQueue].MaxWorkers != maxWorkers {
		t.Errorf("queues %+v, want the default and %s with %d workers and %s with 2", q, webhookQueue, maxWorkers, binary.QueueScan)
	}
}
