package transcript

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/httpx"
	"github.com/Shmeeshmee/salus/pkg/livesocket"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// readLimit bounds a frame from the client: a command (a suggestion's status) is about a hundred bytes. A frame
// over it ends the socket. The socket's other conventions are pkg/livesocket's.
const readLimit = 512

// Transcripts is what the handler calls (Service).
type Transcripts interface {
	Export(ctx context.Context, c appointment.Caller, encounterID string) ([]Event, error)
	Subscribe(ctx context.Context, c appointment.Caller, encounterID, after string) (string, error)
	Next(ctx context.Context, c appointment.Caller, encounterID, after string) ([]Entry, error)
}

// Commands handles a message the client sends on the event stream (ConsultCommand in api/openapi.yaml), in the
// wire envelope, for the signed-in caller and the stream's encounter, and returns the frame to answer with, or
// nil for none. It checks the caller's access itself, and audits.
type Commands interface {
	Command(ctx context.Context, c appointment.Caller, encounterID string, env wire.Envelope) []byte
}

// The type and version of the frame that answers a client's message that failed (CommandErrorMessage).
const (
	CommandErrorType    = "command_error"
	CommandErrorVersion = 1
)

// CommandError is the payload of a command_error frame: the type of the client's message ("" when it was no
// envelope), the id it named when it named one, and the error's code.
type CommandError struct {
	Command string `json:"command"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error"`
}

// CommandErrorFrame is the command_error frame of e.
func CommandErrorFrame(e CommandError) []byte {
	b, _ := wire.Encode(CommandErrorType, CommandErrorVersion, e) // a struct of strings always encodes
	return b
}

// HTTPHandler serves the transcript's JSON API (api/openapi.yaml): the encounter's live events over a
// WebSocket and its stored transcript. Errors are {"error": "<code>"}; internal error text goes to the log
// only, and no log line holds an envelope or anything said.
type HTTPHandler struct {
	svc      Transcripts
	commands Commands // nil: every message is an unknown command
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	clock  clock.Clock // for the socket's deadlines and pings
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc, handing the client's messages to commands; clk times the socket's
// deadlines and pings.
func NewHTTPHandler(svc Transcripts, commands Commands, caller func(ctx context.Context) (appointment.Caller, bool, error), clk clock.Clock, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, commands: commands, caller: caller, clock: clk, log: log}
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
	case !livesocket.IsUpgrade(r):
		writeCode(w, http.StatusBadRequest, "websocket_required")
		return
	case !livesocket.SameOrigin(r):
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
	sock, err := livesocket.Upgrade(w, r, h.clock)
	if err != nil {
		return // the upgrader answered the client
	}
	defer func() { _ = sock.Conn.Close() }()
	h.forward(r, sock, c, id, after)
}

// forward sends each entry after the cursor as a cursor frame and then its envelope's bytes unchanged, until
// the client leaves or Next refuses, which closes the socket with the reason; meanwhile the client's messages
// go to the commands.
func (h *HTTPHandler) forward(r *http.Request, sock *livesocket.Socket, c appointment.Caller, id, after string) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		defer cancel()
		sock.Receive(ctx, readLimit, func(msg []byte) []byte { return h.command(ctx, c, id, msg) })
	}()
	pinged := h.clock.Now()
	for {
		entries, err := h.svc.Next(ctx, c, id, after)
		if ctx.Err() != nil {
			return // the client left
		}
		if err != nil {
			h.close(r, sock, id, err)
			return
		}
		for _, e := range entries {
			if e.Envelope != nil {
				if err := h.send(sock, e); err != nil {
					return
				}
			}
			after = e.ID
		}
		if now := h.clock.Now(); now.Sub(pinged) >= livesocket.PingEvery {
			if err := sock.Ping(); err != nil {
				return
			}
			pinged = now
		}
	}
}

// send writes the entry's cursor frame and then its envelope, each within the write deadline.
func (h *HTTPHandler) send(sock *livesocket.Socket, e Entry) error {
	frame, err := wire.Encode(CursorType, CursorVersion, Cursor{ID: e.ID})
	if err != nil {
		return err
	}
	return sock.Write(frame, e.Envelope)
}

// command is the answer to the client's message msg, or nil.
func (h *HTTPHandler) command(ctx context.Context, c appointment.Caller, id string, msg []byte) []byte {
	env, err := wire.Decode(msg)
	switch {
	case err != nil:
		return CommandErrorFrame(CommandError{Error: "invalid_message"})
	case h.commands == nil:
		return CommandErrorFrame(CommandError{Command: env.Type, Error: "unknown_command"})
	}
	return h.commands.Command(ctx, c, id, env)
}

// close ends the socket for err: the end of access with its reason, anything else logged as internal.
func (h *HTTPHandler) close(r *http.Request, sock *livesocket.Socket, id string, err error) {
	code, reason := livesocket.CloseInternalError, "internal error"
	switch {
	case errors.Is(err, ErrConsentRequired):
		code, reason = livesocket.CloseForbidden, "consent_required"
	case errors.Is(err, ErrForbidden):
		code, reason = livesocket.CloseForbidden, "forbidden"
	case errors.Is(err, ErrNotFound):
		code, reason = livesocket.CloseNotFound, "not_found"
	default:
		h.log.ErrorContext(r.Context(), "event stream failed", "route", r.Pattern, "encounter_id", id, "error", err)
	}
	sock.Close(code, reason)
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
