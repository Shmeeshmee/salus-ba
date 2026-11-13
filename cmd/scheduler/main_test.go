package main

import (
	"bytes"
	"context"
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
