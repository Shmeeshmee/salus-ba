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
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func TestRun_StartsRiverAndStopsCleanly(t *testing.T) {
	dbURL := pgtest.NewDatabaseURL(t)
	t.Setenv("DATABASE_URL", pgtest.AppURL(t, dbURL))
	t.Setenv("RETENTION_DATABASE_URL", pgtest.RetentionURL(t, dbURL))
	t.Setenv("REDIS_URL", redistest.URL(t))
	setLiveKit(t)
	setFiles(t)
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

// The retention_apply job needs its own login; without it the worker does not start.
func TestRun_FailsWithoutTheRetentionDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.AppURL(t, pgtest.NewDatabaseURL(t)))
	t.Setenv("RETENTION_DATABASE_URL", "")
	setLiveKit(t)
	setFiles(t)

	err := run(t.Context(), &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "RETENTION_DATABASE_URL") {
		t.Fatalf("run() error = %v, want one naming RETENTION_DATABASE_URL", err)
	}
}

// The consultations' stream consumer reads Redis; without it the worker does not start.
func TestRun_FailsWithoutTheRedisURL(t *testing.T) {
	dbURL := pgtest.NewDatabaseURL(t)
	t.Setenv("DATABASE_URL", pgtest.AppURL(t, dbURL))
	t.Setenv("RETENTION_DATABASE_URL", pgtest.RetentionURL(t, dbURL))
	t.Setenv("REDIS_URL", "")
	setLiveKit(t)
	setFiles(t)

	err := run(t.Context(), &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "REDIS_URL") {
		t.Fatalf("run() error = %v, want one naming REDIS_URL", err)
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
// (connecting to Postgres) gets up to 15 s, so a busy test server does not fail the test.
func runUntilStarted(t *testing.T) (string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, logs) }()
	deadline := time.After(15 * time.Second)
	for !strings.Contains(logs.String(), `"msg":"started"`) {
		select {
		case err := <-done:
			return logs.String(), err
		case <-deadline:
			cancel()
			err := <-done
			t.Fatalf("run did not log started within 15 s (error after cancel: %v):\n%s", err, logs.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	err := <-done
	return logs.String(), err
}

// setLiveKit sets the LiveKit settings the binary requires, synthetic as in compose.
func setLiveKit(t *testing.T) {
	t.Helper()
	t.Setenv("LIVEKIT_URL", "http://livekit:7880")
	t.Setenv("LIVEKIT_PUBLIC_URL", "ws://localhost:7880")
	t.Setenv("LIVEKIT_API_KEY", "devkey")
	t.Setenv("LIVEKIT_API_SECRET", "local-only-synthetic-livekit-secret-0123456789")
}

// Without the LiveKit settings the binary does not start (docs/consultation/SPEC.md §2 "Configuration").
func TestRun_FailsWithoutTheLiveKitSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://unused@localhost/unused")
	t.Setenv("RETENTION_DATABASE_URL", "postgres://unused@localhost/unused")
	setLiveKit(t)
	t.Setenv("LIVEKIT_API_SECRET", "")

	err := run(t.Context(), &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "LIVEKIT_API_SECRET") {
		t.Fatalf("run() error = %v, want one naming LIVEKIT_API_SECRET", err)
	}
}

// setFiles sets the object store's and clamd's settings the binary jobs require, synthetic as in compose
// (nothing connects to either at start).
func setFiles(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"S3_ENDPOINT": "http://objectstore:3900", "S3_REGION": "garage", "S3_BUCKET": "salus-binaries",
		"S3_ACCESS_KEY_ID": "GK5a1a50000000000000000001", "S3_SECRET_ACCESS_KEY": "synthetic-secret",
		"BINARY_KEK": "bG9jYWwtb25seS1zeW50aGV0aWMtYmluYXJ5LWtlayE=", "BINARY_KEK_ID": "local-1",
		"CLAMD_ADDR": "clamd:3310",
	} {
		t.Setenv(k, v)
	}
}

// Without the object store's or clamd's settings the binary does not start: the scan and discard jobs and
// binary's purger need them (docs/attachments/SPEC.md §2 "Configuration").
func TestRun_FailsWithoutTheFilesSettings(t *testing.T) {
	for _, name := range []string{"S3_ENDPOINT", "BINARY_KEK", "CLAMD_ADDR"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://unused@localhost/unused")
			t.Setenv("RETENTION_DATABASE_URL", "postgres://unused@localhost/unused")
			setLiveKit(t)
			setFiles(t)
			t.Setenv(name, "")

			err := run(t.Context(), &bytes.Buffer{})

			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("run() error = %v, want one naming %s", err, name)
			}
		})
	}
}
