package worker_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The trends_build job builds a closed month on salus_stats and freezes it; a second one finds it built and is done.
func TestTrendsBuild_BuildsAMonthOnce(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	stats, err := postgres.Open(t.Context(), pgtest.StatsURL(t, db.owner.Config().ConnString()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stats.Close)
	client, err := worker.New(db.app, db.retention, stats, liveKit(t), testFiles(t), webhookKey, retention.DefaultPolicy(), worker.Trends{Enabled: true, MinCount: 10}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	done, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobFailed)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(t.Context()) }()
	for range 2 {
		if _, err := client.Insert(t.Context(), worker.TrendsBuildArgs{Period: "2026-09"}, nil); err != nil {
			t.Fatal(err)
		}
		select {
		case ev := <-done:
			if ev.Kind != river.EventKindJobCompleted {
				t.Fatalf("the build %s: %v", ev.Kind, ev.Job.Errors)
			}
		case <-time.After(20 * time.Second):
			t.Fatal("the build did not finish within 20 s")
		}
	}
	var status string
	if err := db.owner.QueryRow(t.Context(), `SELECT status FROM trends_period WHERE period = '2026-09'`).Scan(&status); err != nil || status != "published" {
		t.Errorf("period status %q, %v; want published", status, err)
	}
	var audited int
	if err := db.owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name = 'trends_build'`).Scan(&audited); err != nil || audited != 1 {
		t.Errorf("%d build audit events, %v; want one", audited, err)
	}
}
