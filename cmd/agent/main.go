// Command agent is the transcription agent of the consultation pipeline (docs/plans/consult-intelligence.md,
// SAL-4): a River client on the transcription queue. Each transcription_join job puts it in a consultation's
// LiveKit room, where it streams each person's microphone through speech-to-text (pkg/stt) and puts the partial
// and final transcript events on the encounter's Redis Stream (transcript.StreamKey), each final one translated
// into the other participant's language first (pkg/llm, SAL-5). Storing the final segments (SAL-6) is not here.
// It logs in as salus_app.
//
// The speech engine is chosen by STT_ENGINE (internal/config): the fake, the default, plays placeholder captions
// and sends audio nowhere; azure (pkg/stt/azure) is a processor that may not be enabled with real patient audio
// before its DPA, an EU region and zero data retention are in writing (docs/privacy/processors.md). The language
// model is chosen by LLM_ENGINE: the fake, the default, marks each line with its target language and sends text
// nowhere; anthropic (pkg/llm/anthropic) is a processor that may not be enabled with real patient data before its
// DPA, zero data retention and an EU region are in place (docs/privacy/processors.md).
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/anthropic"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/azure"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
)

// stopTimeout is how long a running join gets to finish after a stop signal.
const stopTimeout = 30 * time.Second

// handshakeTimeout bounds the speech service's WebSocket handshake.
const handshakeTimeout = 10 * time.Second

// llmTimeout bounds one call to the language model; a translation is bounded tighter by its budget
// (transcript.LivePorts.TranslateBudget).
const llmTimeout = 2 * time.Minute

// placeholder is what the fake engine says for every microphone: a partial and a final caption, labelled as
// placeholders, so the pipeline can be seen working without a real engine.
var placeholder = []stt.Result{
	{Text: "Placeholder caption", Start: 0, End: 800 * time.Millisecond},
	{Text: "Placeholder caption from the test speech engine", Start: 0, End: 1500 * time.Millisecond, Final: true},
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	lk, err := config.LoadLiveKit()
	if err != nil {
		return err
	}
	engine, err := config.LoadSTT()
	if err != nil {
		return err
	}
	recognizer, err := newRecognizer(engine)
	if err != nil {
		return err
	}
	model, err := config.LoadLLM()
	if err != nil {
		return err
	}
	translator, err := newTranslator(model)
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()
	video := worker.Video{
		Provider: videosession.NewLiveKitProvider(lk.URL, lk.APIKey, lk.APISecret), Listener: videosession.NewLiveKitListener(),
		URL: strings.Replace(lk.URL, "http", "ws", 1), // the agent reaches LiveKit as the api does, not as the clients do
	}
	client, live, err := worker.NewAgent(pool, rdb, video, recognizer, translator, log)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	log.Info("started", "stt_engine", string(engine.Engine), "llm_engine", string(model.Engine))

	<-ctx.Done()
	log.Info("shutting down")
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	err = client.Stop(stopCtx)
	live.Stop() // leaves every room
	if err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

// newRecognizer is the configured engine: the fake, or Azure in its EU region (azure.New refuses any other).
func newRecognizer(engine config.STT) (stt.Recognizer, error) {
	if engine.Engine != config.STTAzure {
		return &stttest.Recognizer{Results: placeholder, DiscardAudio: true}, nil
	}
	return azure.New(azure.Config{Region: engine.AzureRegion, Key: engine.AzureKey},
		azure.NewDialer(handshakeTimeout), clock.System{})
}

// newTranslator is the configured language model: the fake (keeping no calls), or Anthropic's Claude API.
func newTranslator(model config.LLM) (llm.Translator, error) {
	if model.Engine != config.LLMAnthropic {
		return &llmtest.Translator{DiscardCalls: true}, nil
	}
	return anthropic.New(anthropic.Config{Key: model.AnthropicKey, Model: model.Model}, &http.Client{Timeout: llmTimeout})
}
