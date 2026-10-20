// Command scheduler will enqueue periodic jobs (reminders, retention).
// For now it starts, logs and stops cleanly; nothing else.
package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_ = run(ctx, os.Stderr)
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	log.Info("started")
	<-ctx.Done()
	log.Info("stopped")
	return nil
}
