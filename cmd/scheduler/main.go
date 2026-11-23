// Command scheduler enqueues River's periodic jobs (retention_apply daily at 03:00 UTC, and the trends builds while
// ANALYTICS_PLATFORM_ENABLED is on) and, as River's only
// leader, runs its maintenance (internal/worker.NewScheduler). It works no job and logs in as salus_app.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// stopTimeout is how long the client gets to stop after a stop signal.
const stopTimeout = 30 * time.Second

// startAttempts and startRetryWait bound the tries to start River. Its first connections, the notifier's listener
// among them, have 10 s each inside River and time out while Postgres is slow to hand them out (a deploy, a busy test
// host: cmd/scheduler's TestRun timed out this way under load); River resets a client whose start failed, so a later
// try can succeed.
const (
	startAttempts  = 3
	startRetryWait = 2 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "scheduler:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	client, err := worker.NewScheduler(pool, cfg.Analytics.PlatformEnabled, log)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := start(ctx, func(ctx context.Context) error { return client.Start(context.WithoutCancel(ctx)) }, startRetryWait, log); err != nil {
		return err
	}
	log.Info("started")

	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

// start runs startFn up to startAttempts times, wait apart, until it succeeds; it gives up with the last error, or at
// once when ctx ends. A failed try is logged with River's error, which names no personal data.
func start(ctx context.Context, startFn func(context.Context) error, wait time.Duration, log *slog.Logger) error {
	var err error
	for attempt := 1; attempt <= startAttempts; attempt++ {
		if err = startFn(ctx); err == nil {
			return nil
		}
		log.Warn("river did not start", "attempt", attempt, "error", err)
		if attempt == startAttempts {
			break
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
