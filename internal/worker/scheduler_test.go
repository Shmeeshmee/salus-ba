package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestDailyAt_IsTheNextTimeOfDayInUTC(t *testing.T) {
	sarajevo, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	day := func(d, h, m, s int) time.Time { return time.Date(2026, 11, d, h, m, s, 0, time.UTC) }
	tests := []struct {
		at, want time.Time
	}{
		{at: day(5, 2, 59, 59), want: day(5, 3, 0, 0)},
		{at: day(5, 3, 0, 0), want: day(6, 3, 0, 0)},
		{at: day(5, 3, 0, 1), want: day(6, 3, 0, 0)},
		{at: day(5, 23, 0, 0).In(sarajevo), want: day(6, 3, 0, 0)}, // 00:00 the next day in Sarajevo
	}
	for _, tt := range tests {
		if got := (dailyAt{hour: retentionHour}).Next(tt.at); !got.Equal(tt.want) {
			t.Errorf("Next(%v) = %v, want %v", tt.at, got, tt.want)
		}
	}
}

// atFirstCall fires when River's periodic enqueuer first asks, which it does once its client becomes
// leader, then never again. Each client has its own, so a second leader would fire a second job.
type atFirstCall struct{ asked atomic.Bool }

func (s *atFirstCall) Next(after time.Time) time.Time {
	if s.asked.CompareAndSwap(false, true) {
		return after
	}
	return river.NeverSchedule().Next(after)
}

// Two scheduler replicas: River elects one leader, which alone enqueues the periodic job.
func TestScheduler_EnqueuesThePeriodicJobOnceWithTwoClientsRunning(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	schedules := []*atFirstCall{{}, {}}
	for _, schedule := range schedules {
		client, err := newScheduler(app, slog.New(slog.DiscardHandler), schedule)
		if err != nil {
			t.Fatalf("newScheduler() error = %v", err)
		}
		if err := client.Start(t.Context()); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = client.Stop(context.WithoutCancel(t.Context())) })
	}

	jobs := func() int {
		var n int
		if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM river_job WHERE kind = 'retention_apply'").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	wall := clock.System{}
	deadline := wall.Now().Add(15 * time.Second)
	for jobs() == 0 {
		if wall.Now().After(deadline) {
			t.Fatal("no retention_apply job within 15 s")
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Both clients tried to lead as they started, before the job existed: a second leader would have
	// enqueued at once. Watch one more second for it.
	for end := wall.Now().Add(time.Second); wall.Now().Before(end); {
		time.Sleep(50 * time.Millisecond)
	}
	leaders := 0
	for _, s := range schedules {
		if s.asked.Load() {
			leaders++
		}
	}
	if n := jobs(); n != 1 || leaders != 1 {
		t.Errorf("%d retention_apply jobs from %d leaders, want 1 from 1", n, leaders)
	}
}

// The scheduler leads, so River's rescuer runs in it: it retries a stuck job only of a kind it knows, and
// discards the others. It knows every kind the worker, the notifier and the agent work.
func TestScheduler_KnowsEveryKindTheWorkerTheNotifierAndTheAgentWork(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	client, err := NewScheduler(app, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	for _, args := range []river.JobArgs{
		NoopArgs{}, RetentionApplyArgs{}, schedule.SlotsGenerateArgs{}, consent.RevokedArgs{}, auth.InviteEmailArgs{},
		appointment.BookedEmailArgs{}, appointment.CancelledEmailArgs{}, appointment.ReminderEmailArgs{}, appointment.AccessEndArgs{},
		videosession.WindowCloseArgs{}, appointment.QuestionnaireRequestArgs{}, appointment.QuestionnaireDiscardArgs{},
		binary.ScanArgs{}, binary.DiscardArgs{}, videosession.AgentJoinArgs{}, composition.SignedEmailArgs{}, communication.PartitionsArgs{},
	} {
		var unknown *river.UnknownJobKindError
		if _, err := client.Insert(t.Context(), args, nil); errors.As(err, &unknown) || err != nil {
			t.Errorf("Insert(%s) error = %v, want the kind known", args.Kind(), err)
		}
	}
}

// slots_generate runs hourly, and once as soon as the scheduler leads, so a deploy does not wait an hour.
func TestScheduler_EnqueuesTheHourlySlotGenerationWhenItStartsLeading(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := NewScheduler(app, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Stop(context.WithoutCancel(t.Context())) })

	wall := clock.System{}
	deadline := wall.Now().Add(15 * time.Second)
	for {
		var n int
		var args string
		err := owner.QueryRow(t.Context(), "SELECT count(*), coalesce(min(args::text), '') FROM river_job WHERE kind = 'slots_generate'").Scan(&n, &args)
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			if n != 1 || args != "{}" {
				t.Errorf("%d slots_generate jobs with %s, want one without ids", n, args)
			}
			return
		}
		if wall.Now().After(deadline) {
			t.Fatal("no slots_generate job within 15 s")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// communication_partitions runs daily, and once as soon as the scheduler leads (ADR 0012).
func TestScheduler_EnqueuesTheChatPartitionsWhenItStartsLeading(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := NewScheduler(app, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Stop(context.WithoutCancel(t.Context())) })
	wall := clock.System{}
	for deadline := wall.Now().Add(15 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		var n int
		if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM river_job WHERE kind = 'communication_partitions'").Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 1 {
			return
		}
		if n > 1 || wall.Now().After(deadline) {
			t.Fatalf("%d communication_partitions jobs within 15 s, want 1", n)
		}
	}
}
