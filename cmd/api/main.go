// Command api serves the HTTP and FHIR API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/anthropic"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// shutdownTimeout is how long in-flight requests get to finish after a stop signal.
const shutdownTimeout = 15 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(middleware.NewLogHandler(slog.NewJSONHandler(logOut, nil)))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Required from the start, so a deployment without them fails here rather than at the first
	// consultation (docs/consultation/SPEC.md §2 "Configuration").
	lk, err := config.LoadLiveKit()
	if err != nil {
		return err
	}
	// The questionnaire attachments' object store and key-encryption key (docs/attachments/SPEC.md §2
	// "Configuration").
	files, err := config.LoadObjectStore()
	if err != nil {
		return err
	}
	// The language model that drafts documents: the fake by default, Anthropic's API off until its DPA.
	model, err := config.LoadLLM()
	if err != nil {
		return err
	}
	drafter, err := newDrafter(model)
	if err != nil {
		return err
	}
	// The chat's AI summary: the fake only, until the owner decides the basis for a real model (SAL-31).
	if _, err := config.LoadChatSummary(); err != nil {
		return err
	}
	if cfg.WebhookKey, err = config.LoadWebhookKey(); err != nil { // each environment sets its own; dev generates one
		return err
	}
	app, err := api.New(ctx, cfg, lk, files, drafter, &llmtest.Summarizer{DiscardCalls: true}, log)
	if err != nil {
		return err
	}
	defer app.Close()

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           app.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      api.RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	log.Info("listening", "addr", ln.Addr().String())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}

// newDrafter is the configured language model for drafting: the fake (keeping no calls, every field missing but
// those make e2e's script fills when it has one), or
// Anthropic's Claude API (not to be enabled before its DPA, docs/privacy/processors.md), whose client's timeout
// leaves the drafting's own (composition.DraftTimeout) to bound a call.
func newDrafter(model config.LLM) (llm.Drafter, error) {
	if model.Engine != config.LLMAnthropic {
		script, err := llmtest.LoadScriptDir(model.Script)
		return &llmtest.Drafter{DiscardCalls: true, Script: script}, err
	}
	return anthropic.New(anthropic.Config{Key: model.AnthropicKey, Model: model.Model}, &http.Client{Timeout: composition.DraftTimeout + 5*time.Second})
}
