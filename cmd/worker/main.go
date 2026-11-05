// Command worker runs the River background jobs of the default queue. It logs in as salus_app
// (DATABASE_URL), and as salus_retention (RETENTION_DATABASE_URL) for the retention_apply job alone.
package main

import (
	"context"
	"errors"
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

// stopTimeout is how long running jobs get to finish after a stop signal.
const stopTimeout = 30 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Required from the start, like the api (docs/consultation/SPEC.md §2 "Configuration"); the
	// consultation's jobs take them.
	if _, err := config.LoadLiveKit(); err != nil {
		return err
	}
	ret := config.LoadRetention()
	if ret.DatabaseURL == "" {
		return errors.New("config: RETENTION_DATABASE_URL is required")
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	retentionPool, err := postgres.Open(ctx, ret.DatabaseURL)
	if err != nil {
		return err
	}
	defer retentionPool.Close()
	client, err := worker.New(pool, retentionPool, log)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	log.Info("started")

	<-ctx.Done()
	log.Info("shutting down")
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}
