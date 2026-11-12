package communication

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
	"github.com/Shmeeshmee/salus/pkg/livesocket"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The type and version of the frame that answers a client's frame that failed.
const (
	CommandErrorType    = "command_error"
	CommandErrorVersion = 1
)

// CommandError is the payload of a command_error frame: the type of the client's frame ("" when it was no
// envelope), the client message id it named, and the error's code.
type CommandError struct {
	Command string `json:"command"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error"`
}

// events upgrades to the conversation's socket once the handshake comes from the API's own origin (or from no
// browser: the mobile app), the caller is signed in and may follow it (audited); a refusal is a plain JSON
// error. Then it replays every message after the cursor from Postgres and forwards the live ones, in order, until
// the client leaves or access ends (ADR 0012).
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
	sub, err := h.svc.Subscribe(r.Context(), c, id, r.URL.Query().Get("after"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// Subscribed before the replay, so nothing sent in between is missed.
	frames, stop, err := h.live.Subscribe(r.Context(), ChannelKey(id))
	if err != nil {
		httpx.InternalError(w, r, h.log, err)
		return
	}
	defer stop()
	sock, err := livesocket.Upgrade(w, r, h.clock)
	if err != nil {
		return // the upgrader answered the client
	}
	defer func() { _ = sock.Conn.Close() }()
	s := &stream{h: h, sock: sock, c: c, id: id, last: sub.From, history: sub.HistoryFromSeq, participant: sub.ParticipantID, r: r}
	s.run(r, frames)
}

// stream is one socket's place in the conversation: the last message it sent, and the caller's starting point.
type stream struct {
	h           *HTTPHandler
	sock        *livesocket.Socket
	r           *http.Request // the handshake: its credentials are checked again with every check
	last        Cursor
	typed       time.Time // the last typing frame taken from the client
	sends       sendLimit
	id          string
	participant string // the caller's participant row (SAL-30)
	c           Caller
	history     int64
}

// errSessionEnded is the end of the socket's session (a logout, a revoked or expired session).
var errSessionEnded = errors.New("communication: session ended")

// check is the socket's periodic check: the session is still open, then fill (which checks access again).
func (s *stream) check(ctx context.Context) error {
	ok, err := s.h.active(s.r)
	switch {
	case err != nil:
		return err
	case !ok:
		return errSessionEnded
	}
	return s.fill(ctx)
}

// run replays, then forwards the live frames, fills gaps and checks access every check, and pings, until the
// client leaves or access ends; the client's frames go to command meanwhile.
func (s *stream) run(r *http.Request, frames <-chan []byte) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		defer cancel()
		s.sock.Receive(ctx, maxBody, func(msg []byte) []byte { return s.command(ctx, msg) })
	}()
	check, ping := time.NewTicker(s.h.check), time.NewTicker(livesocket.PingEvery)
	defer check.Stop()
	defer ping.Stop()
	err := s.fill(ctx)
	if err == nil {
		err = s.receipts(ctx)
	}
	for err == nil {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			err = s.live(ctx, f)
		case <-check.C:
			err = s.check(ctx)
		case <-ping.C:
			if s.sock.Ping() != nil {
				err = errSocket
			}
		}
	}
	if ctx.Err() == nil {
		s.h.close(r, s.sock, s.id, err)
	}
}

// fill sends every message after the last one from Postgres, a page at a time, checking access again with each
// page.
func (s *stream) fill(ctx context.Context) error {
	for {
		msgs, err := s.h.svc.After(ctx, s.c, s.id, s.last)
		if err != nil {
			return err
		}
		for _, m := range msgs {
			frame, err := wire.Encode(MessageType, FrameVersion, m)
			if err != nil {
				return err
			}
			if err := s.sock.Write(frame); err != nil {
				return errSocket
			}
			s.last = m.Cursor()
		}
		if len(msgs) < ReplayPage {
			return nil
		}
	}
}

// frameHead is what the socket reads of a live frame: its type and its message's place.
type frameHead struct {
	Type    string `json:"type"`
	Payload struct {
		SentAt time.Time `json:"sent_at"`
		Seq    int64     `json:"seq"`
	} `json:"payload"`
}

// live forwards a live frame: the next message as it is; one already sent is dropped; after a gap (pub/sub drops
// messages) the missing ones come from Postgres. An edit goes out unless it is of a message before the caller's
// starting point; one of a message not sent yet fills first instead.
func (s *stream) live(ctx context.Context, f []byte) error {
	var head frameHead
	if json.Unmarshal(f, &head) != nil {
		return nil // not ours: nothing else publishes on the channel
	}
	seq := head.Payload.Seq
	switch {
	case head.Type == ReceiptType || head.Type == TypingType: // ephemeral: as it comes (SAL-30)
		return s.write(f)
	case head.Type == MessageEditedType && seq > s.last.Seq: // its message has not gone out yet: it goes first
		return s.fill(ctx) // the fill carries the edited body; the edit frame would repeat it
	case head.Type == MessageEditedType && seq > s.history:
		return s.write(f)
	case head.Type != MessageType || seq <= s.last.Seq:
		return nil
	case seq == s.last.Seq+1:
		s.last = Cursor{SentAt: head.Payload.SentAt, Seq: seq}
		return s.write(f)
	}
	return s.fill(ctx)
}

// errSocket is a failed write: the client is gone, nothing to tell it.
var errSocket = errors.New("communication: socket write failed")

func (s *stream) write(f []byte) error {
	if err := s.sock.Write(f); err != nil {
		return errSocket
	}
	return nil
}

// send is the answer to the client's send frame: send_ack, or a command_error.
func (h *HTTPHandler) send(ctx context.Context, c Caller, id string, env wire.Envelope) []byte {
	var cmd SendCommand
	if json.Unmarshal(env.Payload, &cmd) != nil {
		return commandError(CommandError{Command: env.Type, Error: "invalid_message"})
	}
	m, _, err := h.svc.Send(ctx, c, id, cmd.ClientMessageID, cmd.Body, cmd.AttachmentIDs...)
	if err != nil {
		return commandError(CommandError{Command: env.Type, ID: cmd.ClientMessageID, Error: h.code(ctx, id, err)})
	}
	b, _ := wire.Encode(SendAckType, FrameVersion, SendAck{ClientMessageID: cmd.ClientMessageID, Message: m}) // always encodes
	return b
}

func commandError(e CommandError) []byte {
	b, _ := wire.Encode(CommandErrorType, CommandErrorVersion, e) // a struct of strings always encodes
	return b
}

// close ends the socket for err: the end of access with its reason, a gone client with nothing, anything else
// logged as internal.
func (h *HTTPHandler) close(r *http.Request, sock *livesocket.Socket, id string, err error) {
	code, reason := livesocket.CloseInternalError, "internal error"
	switch {
	case errors.Is(err, errSocket):
		return
	case errors.Is(err, errSessionEnded):
		code, reason = livesocket.CloseUnauthenticated, "unauthenticated"
	case errors.Is(err, ErrConsentRequired):
		code, reason = livesocket.CloseForbidden, "consent_required"
	case errors.Is(err, ErrForbidden):
		code, reason = livesocket.CloseForbidden, "forbidden"
	case errors.Is(err, ErrNotFound):
		code, reason = livesocket.CloseNotFound, "not_found"
	default:
		h.log.ErrorContext(r.Context(), "chat stream failed", "route", r.Pattern, "conversation_id", id, "error", err)
	}
	sock.Close(code, reason)
}

// The client's frames a socket takes: sendBurst at once, then one every sendEvery (a person types; a loop does not).
const (
	sendBurst = 5
	sendEvery = 200 * time.Millisecond
)

// sendLimit is a token bucket over the client's frames, per socket: a send on the socket costs one round trip, so a
// client in a loop would otherwise write as fast as the database commits.
type sendLimit struct {
	last   time.Time
	tokens float64
}

func (l *sendLimit) allow(now time.Time) bool {
	if l.last.IsZero() {
		l.tokens = sendBurst
	} else {
		l.tokens = min(sendBurst, l.tokens+float64(now.Sub(l.last))/float64(sendEvery))
	}
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
