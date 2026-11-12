// Package livesocket holds the conventions of a WebSocket that pushes live updates to the web and mobile apps
// (ADR 0012): the upgrade only from the API's own origin or from no browser, one writer at a time, every write
// within a deadline, a ping every PingEvery, a peer gone when it answers none within PongWait, a bounded read
// loop for the client's frames, and the application close codes for an end of access. It knows no frame: the
// caller writes wire envelopes. It is the one package that serves a WebSocket (.golangci.yml, depguard).
package livesocket

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The timings: a ping keeps the connection (and any proxy on the way) alive, a peer that answers no ping within
// PongWait is gone, and a frame that cannot be written within WriteWait drops the socket.
const (
	PingEvery = 25 * time.Second
	PongWait  = 60 * time.Second
	WriteWait = 10 * time.Second
)

// Close codes: the server's error (RFC 6455 §7.4.1), and the application's (4000–4999) for an end of access during
// the stream.
const (
	CloseInternalError = websocket.CloseInternalServerErr
	CloseForbidden     = 4403
	CloseNotFound      = 4404
)

// IsUpgrade reports whether the request is a WebSocket handshake.
func IsUpgrade(r *http.Request) bool { return websocket.IsWebSocketUpgrade(r) }

// Socket is an upgraded connection whose writes are made one at a time: a stream's frames and the answers to the
// client's frames come from two goroutines. Control frames need no lock.
type Socket struct {
	Conn  *websocket.Conn
	clock clock.Clock
	mu    sync.Mutex
}

// Upgrade upgrades the request once it comes from the API's own origin (SameOrigin); on a refusal the upgrader
// has answered the client.
func Upgrade(w http.ResponseWriter, r *http.Request, clk clock.Clock) (*Socket, error) {
	up := websocket.Upgrader{CheckOrigin: SameOrigin}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &Socket{Conn: conn, clock: clk}, nil
}

// Write writes each frame as a text message, each within WriteWait.
func (s *Socket) Write(frames ...[]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range frames {
		_ = s.Conn.SetWriteDeadline(s.clock.Now().Add(WriteWait))
		if err := s.Conn.WriteMessage(websocket.TextMessage, f); err != nil {
			return err
		}
	}
	return nil
}

// Ping writes a ping within WriteWait.
func (s *Socket) Ping() error {
	return s.Conn.WriteControl(websocket.PingMessage, nil, s.clock.Now().Add(WriteWait))
}

// Close sends the close frame with code and reason; the caller closes the connection.
func (s *Socket) Close(code int, reason string) {
	_ = s.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), s.clock.Now().Add(WriteWait))
}

// Receive reads the client's frames until the connection fails, a frame is over limit bytes, the client goes
// silent past PongWait, or ctx ends. Each text frame goes to answer, whose answer (nil for none) is written back.
func (s *Socket) Receive(ctx context.Context, limit int64, answer func(msg []byte) []byte) {
	conn := s.Conn
	conn.SetReadLimit(limit)
	_ = conn.SetReadDeadline(s.clock.Now().Add(PongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(s.clock.Now().Add(PongWait)) })
	for ctx.Err() == nil {
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if typ != websocket.TextMessage {
			continue
		}
		if a := answer(msg); a != nil {
			if err := s.Write(a); err != nil {
				return
			}
		}
	}
}

// SameOrigin reports whether the handshake comes from the API's own origin, or from no browser (no Origin): the
// web app calls the API on its own origin and its session cookie is SameSite=Strict, but a WebSocket cannot carry
// X-Salus-CSRF, so a page of a sibling origin of the same site is refused here. The mobile app sends no Origin
// and signs in with its bearer token.
func SameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}
