package transcript

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/httpx"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The WebSocket's timings: a ping keeps the connection (and any proxy on the way) alive, a peer that answers
// no ping within pongWait is gone, and a frame that cannot be written within writeWait drops the socket.
const (
	pingEvery = 25 * time.Second
	pongWait  = 60 * time.Second
	writeWait = 10 * time.Second
	// readLimit bounds a frame from the client, which has nothing to send but control frames.
	readLimit = 512
)

// Application close codes (RFC 6455 §7.4.2, 4000–4999) for an end of access during the stream.
const (
	closeForbidden = 4403
	closeNotFound  = 4404
)

// Transcripts is what the handler calls (Service).
type Transcripts interface {
	Export(ctx context.Context, c appointment.Caller, encounterID string) ([]Event, error)
	Subscribe(ctx context.Context, c appointment.Caller, encounterID, after string) (string, error)
	Next(ctx context.Context, c appointment.Caller, encounterID, after string) ([]Entry, error)
}

// HTTPHandler serves the transcript's JSON API (api/openapi.yaml): the encounter's live events over a
// WebSocket and its stored transcript. Errors are {"error": "<code>"}; internal error text goes to the log
// only, and no log line holds an envelope or anything said.
type HTTPHandler struct {
	svc Transcripts
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller   func(ctx context.Context) (appointment.Caller, bool, error)
	clock    clock.Clock // for the socket's deadlines and pings
	log      *slog.Logger
	upgrader websocket.Upgrader
}

// NewHTTPHandler returns the handler on svc; clk times the socket's deadlines and pings.
func NewHTTPHandler(svc Transcripts, caller func(ctx context.Context) (appointment.Caller, bool, error), clk clock.Clock, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, clock: clk, log: log, upgrader: websocket.Upgrader{CheckOrigin: sameOrigin}}
}

// Routes declares the transcript's export on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /encounters/{id}/transcript", h.export)
}

// StreamRoutes declares the event stream on mux: it lives as long as the socket, so it is served without the
// request timeout (router.Config.Streams).
func (h *HTTPHandler) StreamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /encounters/{id}/events", h.events)
}

// exportJSON is the Transcript schema.
type exportJSON struct {
	Segments []Event `json:"segments"`
}

func (h *HTTPHandler) export(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	segments, err := h.svc.Export(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, exportJSON{Segments: segments})
}

// events upgrades to the encounter's event stream once the handshake comes from the web app's own origin, the
// caller is signed in and the subscription is allowed (and audited); a refusal is a plain JSON error. Then it
// forwards every entry after the cursor, live, until the client leaves or access ends.
func (h *HTTPHandler) events(w http.ResponseWriter, r *http.Request) {
	switch {
	case !websocket.IsWebSocketUpgrade(r):
		writeCode(w, http.StatusBadRequest, "websocket_required")
		return
	case !sameOrigin(r):
		writeCode(w, http.StatusForbidden, "origin")
		return
	}
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	after, err := h.svc.Subscribe(r.Context(), c, id, r.URL.Query().Get("after"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // the upgrader answered the client
	}
	defer func() { _ = conn.Close() }()
	h.forward(r, conn, c, id, after)
}

// forward sends each entry after the cursor as a cursor frame and then its envelope's bytes unchanged, until
// the client leaves or Next refuses, which closes the socket with the reason.
func (h *HTTPHandler) forward(r *http.Request, conn *websocket.Conn, c appointment.Caller, id, after string) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go h.discard(conn, cancel)
	pinged := h.clock.Now()
	for {
		entries, err := h.svc.Next(ctx, c, id, after)
		if ctx.Err() != nil {
			return // the client left
		}
		if err != nil {
			h.close(r, conn, id, err)
			return
		}
		for _, e := range entries {
			if e.Envelope != nil {
				if err := h.send(conn, e); err != nil {
					return
				}
			}
			after = e.ID
		}
		if now := h.clock.Now(); now.Sub(pinged) >= pingEvery {
			if err := conn.WriteControl(websocket.PingMessage, nil, now.Add(writeWait)); err != nil {
				return
			}
			pinged = now
		}
	}
}

// send writes the entry's cursor frame and then its envelope.
func (h *HTTPHandler) send(conn *websocket.Conn, e Entry) error {
	frame, err := wire.Encode(CursorType, CursorVersion, Cursor{ID: e.ID})
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(h.clock.Now().Add(writeWait))
	if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, e.Envelope)
}

// discard reads the client's frames, which carry nothing but control frames (pongs, the close), until the
// connection fails or the client goes silent past pongWait; then it cancels the stream.
func (h *HTTPHandler) discard(conn *websocket.Conn, cancel context.CancelFunc) {
	defer cancel()
	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(h.clock.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(h.clock.Now().Add(pongWait)) })
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

// close ends the socket for err: the end of access with its reason, anything else logged as internal.
func (h *HTTPHandler) close(r *http.Request, conn *websocket.Conn, id string, err error) {
	code, reason := websocket.CloseInternalServerErr, "internal error"
	switch {
	case errors.Is(err, ErrConsentRequired):
		code, reason = closeForbidden, "consent_required"
	case errors.Is(err, ErrForbidden):
		code, reason = closeForbidden, "forbidden"
	case errors.Is(err, ErrNotFound):
		code, reason = closeNotFound, "not_found"
	default:
		h.log.ErrorContext(r.Context(), "event stream failed", "route", r.Pattern, "encounter_id", id, "error", err)
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), h.clock.Now().Add(writeWait))
}

// sameOrigin reports whether the handshake comes from the API's own origin, or from no browser (no Origin):
// the web app calls the API on its own origin, and its session cookie is SameSite=Strict, but a WebSocket
// cannot carry X-Salus-CSRF, so a page of a sibling origin of the same site is refused here.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// signedIn returns the caller, or answers 401 (or 500) and false.
func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return appointment.Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return appointment.Caller{}, false
	}
	return c, true
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCursor):
		writeCode(w, http.StatusBadRequest, "invalid_cursor")
	case errors.Is(err, ErrConsentRequired):
		writeCode(w, http.StatusForbidden, "consent_required")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
