package suggestion

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The type and version of the client's message on the event stream that sets a suggestion's status
// (SuggestionStatusMessage in api/openapi.yaml).
const (
	StatusMessageType    = "suggestion_status"
	StatusMessageVersion = 1
)

// StatusChange is the payload of a suggestion_status message: what the practitioner did with a suggestion of
// the stream's encounter.
type StatusChange struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

// Statuses is what the command handler calls (Service).
type Statuses interface {
	SetStatus(ctx context.Context, c appointment.Caller, encounterID, id string, status Status) (Event, error)
}

// CommandHandler handles the practitioner's messages on the encounter's event stream (transcript.Commands):
// dismissing, pinning and unpinning a suggestion. Success is answered with nothing (the suggestion with its
// new status comes on the stream), a failure with a command_error frame of its code. Internal error text goes
// to the log only; nothing of a suggestion is logged.
type CommandHandler struct {
	svc Statuses
	log *slog.Logger
}

// NewCommandHandler returns the handler on svc.
func NewCommandHandler(svc Statuses, log *slog.Logger) *CommandHandler {
	return &CommandHandler{svc: svc, log: log}
}

// Command decodes the message, calls the service, and answers.
func (h *CommandHandler) Command(ctx context.Context, c appointment.Caller, encounterID string, env wire.Envelope) []byte {
	if env.Type != StatusMessageType || env.Version != StatusMessageVersion {
		return transcript.CommandErrorFrame(transcript.CommandError{Command: env.Type, Error: "unknown_command"})
	}
	var ch StatusChange
	if err := json.Unmarshal(env.Payload, &ch); err != nil {
		return transcript.CommandErrorFrame(transcript.CommandError{Command: env.Type, Error: "invalid"})
	}
	_, err := h.svc.SetStatus(ctx, c, encounterID, ch.ID, ch.Status)
	if err == nil {
		return nil
	}
	code := "internal"
	switch {
	case errors.Is(err, ErrInvalid):
		code = "invalid"
	case errors.Is(err, ErrNotFound):
		code = "not_found"
	case errors.Is(err, ErrConsentRequired):
		code = "consent_required"
	case errors.Is(err, ErrForbidden):
		code = "forbidden"
	default:
		h.log.ErrorContext(ctx, "suggestion status failed", "encounter_id", encounterID, "error", err)
	}
	return transcript.CommandErrorFrame(transcript.CommandError{Command: env.Type, ID: ch.ID, Error: code})
}
