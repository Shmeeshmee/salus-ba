package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// startCeiling bounds the wait for the binary to start: it is polled every 10 ms and the test goes on as soon as it is
// up, so the ceiling only matters when something is wrong; it is high because a host running several stacks' suites
// at once can take tens of seconds to hand out a first Postgres connection (15 s was too short).
const startCeiling = 60 * time.Second

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func TestRun_StartsRiverAndStopsCleanly(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.AppURL(t, pgtest.NewDatabaseURL(t)))
	logs, err := runUntilStarted(t)
	if err != nil {
		t.Fatalf("run() error = %v, want nil after the context ends", err)
	}
	for _, msg := range []string{`"msg":"started"`, `"msg":"stopped"`} {
		if !strings.Contains(logs, msg) {
			t.Errorf("logs lack %s:\n%s", msg, logs)
		}
	}
}

// River's first connections, the notifier's listener among them (10 s inside River), can time out while Postgres is
// slow to hand them out: start tries again, and gives up after its attempts with the last error.
func TestStart_TriesAgainAfterAFailedStart(t *testing.T) {
	timeout := errors.New("notifier.listenerConnect: timeout")
	for _, tt := range []struct {
		name      string
		failures  int
		wantCalls int
		wantErr   bool
	}{
		{name: "at once", failures: 0, wantCalls: 1},
		{name: "after two timeouts", failures: 2, wantCalls: 3},
		{name: "never", failures: 5, wantCalls: startAttempts, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			var logs bytes.Buffer
			err := start(t.Context(), func(context.Context) error {
				calls++
				if calls <= tt.failures {
					return timeout
				}
				return nil
			}, time.Millisecond, slog.New(slog.NewJSONHandler(&logs, nil)))
			if calls != tt.wantCalls || (err != nil) != tt.wantErr || (tt.wantErr && !errors.Is(err, timeout)) {
				t.Errorf("start() = %v after %d calls, want error %v after %d", err, calls, tt.wantErr, tt.wantCalls)
			}
			if got := strings.Count(logs.String(), `"msg":"river did not start"`); got != min(tt.failures, startAttempts) {
				t.Errorf("%d warnings, want one per failed attempt:\n%s", got, logs.String())
			}
		})
	}
}

// A start the stop signal ends while it waits to try again returns at once.
func TestStart_StopsWaitingWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	err := start(ctx, func(context.Context) error {
		calls++
		cancel()
		return errors.New("notifier.listenerConnect: timeout")
	}, time.Hour, slog.New(slog.DiscardHandler))
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Errorf("start() = %v after %d calls, want context.Canceled after 1", err, calls)
	}
}

func TestRun_FailsWithoutConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if err := run(t.Context(), &bytes.Buffer{}); err == nil {
		t.Fatal("run() error = nil, want a config error")
	}
}

// syncBuffer collects run's log output; run writes it while the test reads it.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runUntilStarted runs run until it logs "started", then cancels it and returns its error and logs. Startup
// (connecting to Postgres) is waited for as long as it takes, up to startCeiling, so a busy host does not fail the test.
func runUntilStarted(t *testing.T) (string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, logs) }()
	deadline := time.After(startCeiling)
	for !strings.Contains(logs.String(), `"msg":"started"`) {
		select {
		case err := <-done:
			return logs.String(), err
		case <-deadline:
			cancel()
			err := <-done
			t.Fatalf("run did not log started within %v (error after cancel: %v):\n%s", startCeiling, err, logs.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	err := <-done
	return logs.String(), err
}
