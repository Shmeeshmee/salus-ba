package communication

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Shmeeshmee/salus/pkg/wire"
)

// command is the answer to the client's frame msg: a send is answered with send_ack; a read and a typing frame with
// nothing (SAL-30: the others get a receipt or a typing frame); anything else with a command_error. Sends and reads
// share the socket's limit; typing frames past one a TypingEvery are dropped here, before Redis.
func (s *stream) command(ctx context.Context, msg []byte) []byte {
	env, err := wire.Decode(msg)
	switch {
	case err != nil:
		return commandError(CommandError{Error: "invalid_message"})
	case env.Version != FrameVersion:
		return commandError(CommandError{Command: env.Type, Error: "unknown_command"})
	}
	switch env.Type {
	case SendType, ReadType:
		if !s.sends.allow(s.h.clock.Now()) {
			return commandError(CommandError{Command: env.Type, Error: "rate_limited"})
		}
		if env.Type == SendType {
			return s.h.send(ctx, s.c, s.id, env)
		}
		return s.read(ctx, env)
	case TypingType:
		return s.typing(ctx)
	}
	return commandError(CommandError{Command: env.Type, Error: "unknown_command"})
}

// read moves the caller's read position as PUT …/read does; the service tells the others.
func (s *stream) read(ctx context.Context, env wire.Envelope) []byte {
	var cmd ReadCommand
	if json.Unmarshal(env.Payload, &cmd) != nil {
		return commandError(CommandError{Command: env.Type, Error: "invalid_message"})
	}
	if err := s.h.svc.MarkRead(ctx, s.c, s.id, cmd.Seq); err != nil {
		return commandError(CommandError{Command: env.Type, Error: s.h.code(ctx, s.id, err)})
	}
	return nil
}

// typing tells the others the caller types, at most once a TypingEvery; a failure is logged, never answered.
func (s *stream) typing(ctx context.Context) []byte {
	now := s.h.clock.Now()
	if !s.typed.IsZero() && now.Sub(s.typed) < TypingEvery {
		return nil
	}
	s.typed = now
	if err := s.h.svc.Typing(ctx, s.id, s.participant); err != nil {
		s.h.log.WarnContext(ctx, "chat typing failed", "conversation_id", s.id, "error", err.Error())
	}
	return nil
}

// receipts sends how far each current participant has read, as kept, once the socket has replayed; a failure is logged and
// the socket goes on without them.
func (s *stream) receipts(ctx context.Context) error {
	rs, err := s.h.svc.Receipts(ctx, s.c, s.id)
	if err != nil {
		s.h.log.WarnContext(ctx, "chat receipts failed", "conversation_id", s.id, "error", err.Error())
		return nil
	}
	for _, r := range rs {
		frame, err := wire.Encode(ReceiptType, FrameVersion, r)
		if err != nil {
			return err
		}
		if err := s.write(frame); err != nil {
			return err
		}
	}
	return nil
}

// code is the fixed code of a command's error; an internal one is logged.
func (h *HTTPHandler) code(ctx context.Context, id string, err error) string {
	for _, e := range allErrorCodes {
		if errors.Is(err, e.err) {
			return e.code
		}
	}
	h.log.ErrorContext(ctx, "chat command failed", "conversation_id", id, "error", err)
	return "internal_error"
}
