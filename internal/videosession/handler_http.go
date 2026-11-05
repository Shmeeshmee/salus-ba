package videosession

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxWebhookBody bounds a LiveKit webhook's body (SPEC.md §2 "Webhook endpoint").
const maxWebhookBody = 64 << 10

// Sessions is what the handler calls (Service).
type Sessions interface {
	HandleWebhook(ctx context.Context, body []byte, authorization string) (EventKind, error)
}

// Handler serves the consultation's JSON API (api/openapi.yaml): for now LiveKit's webhook, which carries
// no session. Errors are {"error": "<code>"}; internal error text goes to the log only, and no log line holds
// a token, room, identity or name.
type Handler struct {
	svc Sessions
	log *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Sessions, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Routes declares the consultation's routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/livekit", h.webhook)
}

// webhook reads the body up to 64 KiB and hands it to the service with the Authorization header, which the
// provider verifies before anything else is read: 401 for a bad or missing signature, 200 for an event
// handled (also with no effect, and for a room we do not know), 500 for a handling error, so LiveKit
// delivers it again. The log holds the request id and the event's kind only.
func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	kind, err := h.svc.HandleWebhook(r.Context(), body, r.Header.Get("Authorization"))
	switch {
	case errors.Is(err, ErrInvalidSignature):
		h.log.WarnContext(r.Context(), "livekit webhook refused")
		httpx.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
	case errors.Is(err, ErrUnknownRoom):
		h.log.InfoContext(r.Context(), "unknown room", "kind", kind)
		httpx.JSON(w, http.StatusOK, struct{}{})
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	default:
		h.log.InfoContext(r.Context(), "livekit webhook", "kind", kind)
		httpx.JSON(w, http.StatusOK, struct{}{})
	}
}
