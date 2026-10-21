package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/middleware"
)

func jsonLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(middleware.NewLogHandler(slog.NewJSONHandler(buf, nil)))
}

// logLines decodes one JSON object per line.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for line := range strings.Lines(buf.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestRequestID_GeneratesOneWhenMissing(t *testing.T) {
	var seen string
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = middleware.RequestIDFrom(r.Context())
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if seen == "" {
		t.Fatal("no request id in the context")
	}
	if got := rec.Header().Get("X-Request-ID"); got != seen {
		t.Errorf("X-Request-ID = %q, want %q", got, seen)
	}
}

func TestRequestID_KeepsAValidIncomingID(t *testing.T) {
	var seen string
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = middleware.RequestIDFrom(r.Context())
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "edge-7f3a.42_b")

	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "edge-7f3a.42_b" {
		t.Errorf("request id = %q, want the incoming one", seen)
	}
}

func TestRequestID_ReplacesAnUnsafeIncomingID(t *testing.T) {
	for _, bad := range []string{"has space", "line\nbreak", strings.Repeat("a", 65), "jane@example.org"} {
		var seen string
		h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = middleware.RequestIDFrom(r.Context())
		}))
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", bad)

		h.ServeHTTP(httptest.NewRecorder(), req)

		if seen == bad || seen == "" {
			t.Errorf("incoming id %q: request id = %q, want a generated one", bad, seen)
		}
	}
}

func TestLogHandler_AddsTheRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := jsonLogger(&buf)
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "hello")
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-1")

	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := logLines(t, &buf)[0]["request_id"]; got != "req-1" {
		t.Errorf("request_id = %v, want req-1", got)
	}
}

func TestAccessLog_LogsRoutePatternNotPathOrQuery(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /things/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := middleware.RequestID(middleware.AccessLog(jsonLogger(&buf), mux)(mux))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/things/abc123?name=Jane+Placeholder", nil)
	req.Header.Set("X-Request-ID", "req-2")

	h.ServeHTTP(httptest.NewRecorder(), req)

	line := logLines(t, &buf)[0]
	if line["route"] != "GET /things/{id}" {
		t.Errorf("route = %v, want the pattern", line["route"])
	}
	if line["status"] != float64(http.StatusTeapot) {
		t.Errorf("status = %v, want 418", line["status"])
	}
	if line["request_id"] != "req-2" || line["method"] != "GET" {
		t.Errorf("request_id, method = %v, %v", line["request_id"], line["method"])
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Error("no duration_ms")
	}
	if strings.Contains(buf.String(), "Jane") || strings.Contains(buf.String(), "abc123") {
		t.Errorf("access log carries the path or query: %s", buf.String())
	}
}

func TestAccessLog_DefaultsStatusTo200AndMarksUnmatchedRoutes(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	h := middleware.AccessLog(jsonLogger(&buf), mux)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/nowhere", nil))

	line := logLines(t, &buf)[0]
	if line["status"] != float64(http.StatusOK) || line["route"] != "unmatched" {
		t.Errorf("status, route = %v, %v; want 200, unmatched", line["status"], line["route"])
	}
}

func TestRecover_TurnsAPanicIntoAFixed500(t *testing.T) {
	var buf bytes.Buffer
	h := middleware.Recover(jsonLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":\"internal error\"}\n" {
		t.Errorf("body = %q, want the fixed message", body)
	}
	line := logLines(t, &buf)[0]
	if line["msg"] != "panic" || line["panic"] != "boom" {
		t.Errorf("log = %v, want a panic line", line)
	}
	if stack, _ := line["stack"].(string); !strings.Contains(stack, "middleware_test") {
		t.Errorf("stack = %q, want the panicking frame", stack)
	}
}

func TestRecover_LetsAbortHandlerThrough(t *testing.T) {
	h := middleware.Recover(slog.New(slog.DiscardHandler))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler to propagate", p)
		}
	}()

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}

func TestRequestIDFrom_EmptyWithoutMiddleware(t *testing.T) {
	if got := middleware.RequestIDFrom(context.Background()); got != "" {
		t.Errorf("RequestIDFrom(background) = %q, want empty", got)
	}
}

func TestLogHandler_KeepsTheRequestIDThroughWithAndWithGroup(t *testing.T) {
	var buf bytes.Buffer
	log := jsonLogger(&buf)
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.With("component", "test").InfoContext(r.Context(), "with")
		log.WithGroup("g").InfoContext(r.Context(), "group", "k", "v")
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-3")

	h.ServeHTTP(httptest.NewRecorder(), req)

	lines := logLines(t, &buf)
	if lines[0]["request_id"] != "req-3" || lines[0]["component"] != "test" {
		t.Errorf("With: line = %v, want request_id and component", lines[0])
	}
	if g, _ := lines[1]["g"].(map[string]any); g["request_id"] != "req-3" || g["k"] != "v" {
		t.Errorf("WithGroup: line = %v, want request_id and k in the group", lines[1])
	}
}

func TestAccessLog_LetsResponseControllerReachTheWriter(t *testing.T) {
	mux := http.NewServeMux()
	h := middleware.AccessLog(slog.New(slog.DiscardHandler), mux)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush through the access log: %v", err)
		}
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if !rec.Flushed {
		t.Error("the underlying writer was not flushed")
	}
}

func TestAccessLog_LogsAnEmptyResponseAs200(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	h := middleware.AccessLog(jsonLogger(&buf), mux)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if status := logLines(t, &buf)[0]["status"]; status != float64(http.StatusOK) {
		t.Errorf("status = %v, want 200", status)
	}
}
