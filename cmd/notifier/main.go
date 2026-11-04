// Command notifier sends e-mail (ADR 0007): it runs a River client that works only the email queue, through
// SMTP (Mailpit locally; SES once its DPA is signed). It logs in as salus_app.
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

// stopTimeout is how long a running send gets to finish after a stop signal.
const stopTimeout = 30 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notifier:", err)
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
	client, err := worker.NewNotifier(pool, log, cfg.SMTPAddr, cfg.WebBaseURL)
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
