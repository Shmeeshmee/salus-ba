package livesocket_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/livesocket"
)

func TestSameOrigin_AcceptsTheAPIsOwnOriginOrNoBrowser(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		origin, host string
		want         bool
	}{
		{"", "api.example.test", true},
		{"https://api.example.test", "api.example.test", true},
		{"https://API.example.test", "api.example.test", true},
		{"https://web.example.test", "api.example.test", false},
		{"https://api.example.test:8443", "api.example.test", false},
		{"::not a url", "api.example.test", false},
	} {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.Host = tt.host
		if tt.origin != "" {
			r.Header.Set("Origin", tt.origin)
		}
		if got := livesocket.SameOrigin(r); got != tt.want {
			t.Errorf("Origin %q on %q = %v, want %v", tt.origin, tt.host, got, tt.want)
		}
	}
}

// serve runs handle on every upgraded socket and returns a client connected to it.
func serve(t *testing.T, handle func(ctx context.Context, s *livesocket.Socket)) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, err := livesocket.Upgrade(w, r, clock.System{})
		if err != nil {
			return
		}
		defer func() { _ = s.Conn.Close() }()
		handle(r.Context(), s)
	}))
	t.Cleanup(srv.Close)
	conn, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetReadDeadline(clock.System{}.Now().Add(5 * time.Second))
	return conn
}

func TestSocket_WritesFramesAndAnswersEachTextFrame(t *testing.T) {
	t.Parallel()
	conn := serve(t, func(ctx context.Context, s *livesocket.Socket) {
		if err := s.Write([]byte("first"), []byte("second")); err != nil {
			return
		}
		s.Receive(ctx, 64, func(msg []byte) []byte {
			if string(msg) == "quiet" {
				return nil
			}
			return append([]byte("re:"), msg...)
		})
	})
	for _, want := range []string{"first", "second"} {
		if _, got, err := conn.ReadMessage(); err != nil || string(got) != want {
			t.Fatalf("read %q, %v; want %q", got, err, want)
		}
	}
	for _, msg := range []string{"quiet", "hello"} {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			t.Fatal(err)
		}
	}
	if _, got, err := conn.ReadMessage(); err != nil || string(got) != "re:hello" {
		t.Fatalf("read %q, %v; want the answer to hello only", got, err)
	}
}

func TestSocket_ReceiveEndsOnAFrameOverTheLimit(t *testing.T) {
	t.Parallel()
	ended := make(chan struct{})
	conn := serve(t, func(ctx context.Context, s *livesocket.Socket) {
		s.Receive(ctx, 8, func([]byte) []byte { return nil })
		close(ended)
	})
	if err := conn.WriteMessage(websocket.TextMessage, []byte("far more than eight bytes")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("Receive did not end")
	}
}

func TestSocket_CloseSendsTheCodeAndReason(t *testing.T) {
	t.Parallel()
	conn := serve(t, func(_ context.Context, s *livesocket.Socket) {
		s.Close(livesocket.CloseForbidden, "consent_required")
	})
	_, _, err := conn.ReadMessage()
	var ce *websocket.CloseError
	if !errors.As(err, &ce) || ce.Code != livesocket.CloseForbidden || ce.Text != "consent_required" {
		t.Fatalf("read %v, want close 4403 consent_required", err)
	}
}

func TestSocket_PingReachesTheClient(t *testing.T) {
	t.Parallel()
	pinged := make(chan struct{}, 1)
	conn := serve(t, func(_ context.Context, s *livesocket.Socket) {
		_ = s.Ping()
		_ = s.Write([]byte("after"))
	})
	conn.SetPingHandler(func(string) error { pinged <- struct{}{}; return nil })
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pinged:
	default:
		t.Error("no ping before the frame")
	}
}

func TestUpgrade_RefusesAnotherOrigin(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := livesocket.Upgrade(w, r, clock.System{}); err == nil {
			t.Error("upgraded a sibling origin")
		}
	}))
	defer srv.Close()
	h := http.Header{"Origin": {"https://elsewhere.example.test"}}
	_, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), h)
	if res != nil {
		_ = res.Body.Close()
	}
	if err == nil {
		t.Error("dial from another origin succeeded")
	}
}
