package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// startCeiling bounds the wait for the binary to start: it is polled every 10 ms and the test goes on as soon as it is
// up, so the ceiling only matters when something is wrong; it is high because a host running several stacks' suites
// at once can take tens of seconds to hand out a first Postgres connection (15 s was too short).
const startCeiling = 60 * time.Second

func TestMain(m *testing.M) { pgtest.Main(m, nil) }

// syncBuffer is a log destination the server and the test can use at the same time.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// lines returns the JSON log lines written so far.
func (b *syncBuffer) lines(t *testing.T) []map[string]any {
	t.Helper()
	b.mu.Lock()
	text := b.buf.String()
	b.mu.Unlock()
	var out []map[string]any
	for line := range strings.Lines(text) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// TestRun_ServesUntilCancelledThenShutsDown starts run on a free port, finds the port in the "listening" log
// line, calls /healthz, then cancels and expects a clean return with the request in the access log.
func TestRun_ServesUntilCancelledThenShutsDown(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	t.Setenv("REDIS_URL", redistest.URL(t))
	t.Setenv("HTTP_ADDR", "127.0.0.1:0")
	t.Setenv("TOTP_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtdG90cC1rZXktMzI=")       // synthetic, as in compose
	t.Setenv("RATE_LIMIT_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtcmF0ZWxpbWl0LWs=") // synthetic, as in compose
	t.Setenv("WEBHOOK_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtd2ViaG9vay1rZXk=")    // synthetic, as in compose
	t.Setenv("FHIR_BASE_URL", "http://localhost:8080/fhir/R4")
	setLiveKit(t)
	setObjectStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	logs := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, logs) }()

	addr := waitForListening(t, logs, done)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz = %d, want 200", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() error = %v, want nil after a clean shutdown", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after cancel")
	}
	// Shutdown waits for handlers, so the access-log line is written by now.
	if !hasAccessLine(logs.lines(t), "GET /healthz", http.StatusOK) {
		t.Errorf("no access-log line for GET /healthz 200 in:\n%v", logs.lines(t))
	}
}

func waitForListening(t *testing.T, logs *syncBuffer, done <-chan error) string {
	t.Helper()
	timeout := time.After(startCeiling)
	for {
		for _, line := range logs.lines(t) {
			if line["msg"] == "listening" {
				addr, _ := line["addr"].(string)
				return addr
			}
		}
		select {
		case err := <-done:
			t.Fatalf("run() returned before listening: %v", err)
		case <-timeout:
			t.Fatalf("run() did not log \"listening\" within %v", startCeiling)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func hasAccessLine(lines []map[string]any, route string, status int) bool {
	for _, l := range lines {
		if l["msg"] == "request" && l["route"] == route && l["status"] == float64(status) && l["request_id"] != nil {
			return true
		}
	}
	return false
}

func TestRun_FailsWithoutConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if err := run(t.Context(), io.Discard); err == nil {
		t.Fatal("run() error = nil, want a config error")
	}
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

// setObjectStore sets the object store settings and the key-encryption key the binary requires, synthetic as
// in compose (nothing connects to the store at start).
func setObjectStore(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"S3_ENDPOINT": "http://objectstore:3900", "S3_REGION": "garage", "S3_BUCKET": "salus-binaries",
		"S3_ACCESS_KEY_ID": "GK5a1a50000000000000000001", "S3_SECRET_ACCESS_KEY": "synthetic-secret",
		"BINARY_KEK": "bG9jYWwtb25seS1zeW50aGV0aWMtYmluYXJ5LWtlayE=", "BINARY_KEK_ID": "local-1",
	} {
		t.Setenv(k, v)
	}
}

// Without the object store settings the binary does not start (docs/attachments/SPEC.md §2 "Configuration").
func TestRun_FailsWithoutTheObjectStoreSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://unused@localhost/unused")
	t.Setenv("TOTP_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtdG90cC1rZXktMzI=")
	t.Setenv("RATE_LIMIT_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtcmF0ZWxpbWl0LWs=")
	t.Setenv("WEBHOOK_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtd2ViaG9vay1rZXk=")
	t.Setenv("FHIR_BASE_URL", "http://localhost:8080/fhir/R4")
	setLiveKit(t)
	setObjectStore(t)
	t.Setenv("BINARY_KEK", "")

	err := run(t.Context(), &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "BINARY_KEK") {
		t.Fatalf("run() error = %v, want one naming BINARY_KEK", err)
	}
}

// Outside dev the binary does not start without its webhook key: each environment sets its own.
func TestRun_FailsOutsideDevWithoutTheWebhookKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://unused@localhost/unused")
	t.Setenv("TOTP_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtdG90cC1rZXktMzI=")
	t.Setenv("RATE_LIMIT_KEY", "bG9jYWwtb25seS1zeW50aGV0aWMtcmF0ZWxpbWl0LWs=")
	t.Setenv("FHIR_BASE_URL", "http://localhost:8080/fhir/R4")
	setLiveKit(t)
	setObjectStore(t)
	t.Setenv("WEBHOOK_KEY", "")
	t.Setenv("WEBHOOK_KEY_FILE", "")
	t.Setenv("SALUS_ENV", "production")

	err := run(t.Context(), &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "WEBHOOK_KEY") {
		t.Fatalf("run() error = %v, want one naming WEBHOOK_KEY", err)
	}
}

// The default drafter keeps no calls: an api running for months holds no more memory than its first draft. The
// Anthropic engine needs its key.
func TestNewDrafter_TheFakeKeepsNoCallsAndAnthropicNeedsItsKey(t *testing.T) {
	d, err := newDrafter(config.LLM{Engine: config.LLMFake})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := d.(*llmtest.Drafter)
	if !ok {
		t.Fatalf("the default drafter is %T, want the fake", d)
	}
	if _, err := fake.Draft(t.Context(), llm.DraftRequest{Lang: "bs", Fields: []llm.FieldSpec{{Key: "a", Label: "L"}}}); err != nil || len(fake.Calls()) != 0 {
		t.Errorf("the fake drafted (%v) and kept %d calls, want none", err, len(fake.Calls()))
	}
	if _, err := newDrafter(config.LLM{Engine: config.LLMAnthropic, Model: "m"}); err == nil {
		t.Error("anthropic without a key: newDrafter() = nil, want an error")
	}
}
