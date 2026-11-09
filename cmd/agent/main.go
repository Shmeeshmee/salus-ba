// Command agent is the transcription agent of the consultation pipeline (docs/plans/consult-intelligence.md,
// SAL-4): a River client on the transcription queue. Each transcription_join job puts it in a consultation's
// LiveKit room, where it streams each person's microphone through speech-to-text (pkg/stt) and puts the partial
// and final transcript events on the encounter's Redis Stream (transcript.StreamKey). Translation (SAL-5) and
// storing the final segments (SAL-6) are not here. It logs in as salus_app.
//
// The engine is chosen by STT_ENGINE (internal/config): the fake, the default, plays placeholder captions and
// sends audio nowhere; azure (pkg/stt/azure) is a processor that may not be enabled with real patient audio
// before its DPA, an EU region and zero data retention are in writing (docs/privacy/processors.md).
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
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
	client, live, err := worker.NewAgent(pool, rdb, video, recognizer, log)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	log.Info("started", "stt_engine", string(engine.Engine))

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
		&websocket.Dialer{HandshakeTimeout: handshakeTimeout}, clock.System{})
}
