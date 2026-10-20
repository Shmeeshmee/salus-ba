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

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

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
	t.Setenv("HTTP_ADDR", "127.0.0.1:0")
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
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range logs.lines(t) {
			if line["msg"] == "listening" {
				addr, _ := line["addr"].(string)
				return addr
			}
		}
		select {
		case err := <-done:
			t.Fatalf("run() returned before listening: %v", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatal("run() did not log \"listening\" within 10s")
	return ""
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
