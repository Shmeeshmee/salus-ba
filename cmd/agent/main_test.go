package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
)

// startCeiling bounds the wait for the binary to start: it is polled every 10 ms and the test goes on as soon as it is
// up, so the ceiling only matters when something is wrong; it is high because a host running several stacks' suites
// at once can take tens of seconds to hand out a first Postgres connection (15 s was too short).
const startCeiling = 60 * time.Second

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// setStack points the agent at a fresh database, the compose Redis and LiveKit, with the fake engine.
func setStack(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", pgtest.AppURL(t, pgtest.NewDatabaseURL(t)))
	t.Setenv("REDIS_URL", redistest.URL(t))
	t.Setenv("LIVEKIT_URL", os.Getenv("TEST_LIVEKIT_URL"))
	t.Setenv("LIVEKIT_PUBLIC_URL", "ws://localhost:7880")
	t.Setenv("LIVEKIT_API_KEY", os.Getenv("TEST_LIVEKIT_API_KEY"))
	t.Setenv("LIVEKIT_API_SECRET", os.Getenv("TEST_LIVEKIT_API_SECRET"))
	t.Setenv("STT_ENGINE", "")
	t.Setenv("LLM_ENGINE", "")
}

// The agent starts its River client on the transcription queue, with the fake engine by default, and stops
// cleanly when its context ends.
func TestRun_StartsWithTheFakeEngineAndStopsCleanly(t *testing.T) {
	setStack(t)
	logs, err := runUntilStarted(t)
	if err != nil {
		t.Fatalf("run() error = %v, want nil after the context ends", err)
	}
	for _, msg := range []string{`"msg":"started"`, `"stt_engine":"fake"`, `"llm_engine":"fake"`, `"msg":"stopped"`} {
		if !strings.Contains(logs, msg) {
			t.Errorf("logs lack %s:\n%s", msg, logs)
		}
	}
}

// Azure only in an EU region: without one, or with another, the agent refuses to start, and never says the key.
func TestRun_RefusesAzureOutsideTheEU(t *testing.T) {
	for _, region := range []string{"", "eastus"} {
		setStack(t)
		t.Setenv("STT_ENGINE", "azure")
		t.Setenv("AZURE_SPEECH_REGION", region)
		t.Setenv("AZURE_SPEECH_KEY", "speech-key-marker")
		err := run(t.Context(), &bytes.Buffer{})
		if err == nil || strings.Contains(err.Error(), "marker") {
			t.Errorf("region %q: run() = %v, want a refusal without the key", region, err)
		}
	}
}

// With Azure in an EU region the agent starts; nothing connects to Azure until a call has audio.
func TestRun_StartsWithAzureInAnEURegion(t *testing.T) {
	setStack(t)
	t.Setenv("STT_ENGINE", "azure")
	t.Setenv("AZURE_SPEECH_REGION", "westeurope")
	t.Setenv("AZURE_SPEECH_KEY", "speech-key-marker")
	logs, err := runUntilStarted(t)
	if err != nil || !strings.Contains(logs, `"stt_engine":"azure"`) || strings.Contains(logs, "marker") {
		t.Errorf("run() = %v, logs:\n%s\nwant started with azure, without the key", err, logs)
	}
}

// With Anthropic's API and its key the agent starts; nothing is sent until a final segment needs translating.
// Without the key it refuses to start. Neither says the key.
func TestRun_StartsWithAnthropicOnlyWithItsKey(t *testing.T) {
	setStack(t)
	t.Setenv("LLM_ENGINE", "anthropic")
	t.Setenv("ANTHROPIC_API_KEY", "llm-key-marker")
	logs, err := runUntilStarted(t)
	if err != nil || !strings.Contains(logs, `"llm_engine":"anthropic"`) || strings.Contains(logs, "marker") {
		t.Errorf("run() = %v, logs:\n%s\nwant started with anthropic, without the key", err, logs)
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	if err := run(t.Context(), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("without the key: run() = %v, want the missing setting named", err)
	}
}

func TestRun_FailsWithoutConfig(t *testing.T) {
	for _, name := range []string{"DATABASE_URL", "REDIS_URL", "LIVEKIT_API_SECRET"} {
		setStack(t)
		t.Setenv(name, "")
		if err := run(t.Context(), &bytes.Buffer{}); err == nil {
			t.Errorf("without %s: run() error = nil, want a config error", name)
		}
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

// runUntilStarted runs run until it logs "started", then cancels it and returns its error and logs.
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

// The default translator keeps no calls: a consultation of hours holds no more memory than its first line.
func TestNewTranslator_TheFakeKeepsNoCalls(t *testing.T) {
	tr, err := newTranslator(config.LLM{Engine: config.LLMFake})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := tr.(*llmtest.Translator)
	if !ok {
		t.Fatalf("the default translator is %T, want the fake", tr)
	}
	if _, err := fake.Translate(t.Context(), llm.TranslateRequest{Text: "placeholder", From: "tr", To: "bs"}); err != nil || len(fake.Calls()) != 0 {
		t.Errorf("the fake translated (%v) and kept %d calls, want none", err, len(fake.Calls()))
	}
}

// The default engine keeps no audio: a consultation of hours holds no more memory than its placeholder script.
func TestNewRecognizer_TheFakeKeepsNoAudio(t *testing.T) {
	r, err := newRecognizer(config.STT{Engine: config.STTFake})
	if err != nil {
		t.Fatal(err)
	}
	if fake, ok := r.(*stttest.Recognizer); !ok || !fake.DiscardAudio {
		t.Errorf("the default recognizer is %T %+v, want the fake discarding its audio", r, r)
	}
}
