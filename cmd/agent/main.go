// Command agent will be the LiveKit agent of the consultation pipeline (docs/plans/consult-intelligence.md):
// it joins a consultation's room, streams each participant's audio to speech-to-text (pkg/stt), translates the
// segments (pkg/llm) and puts them on the encounter's Redis Stream (transcript.StreamKey). The contracts are
// in place (SAL-2); the room comes with SAL-3. For now it starts, logs and stops cleanly; nothing else.
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
