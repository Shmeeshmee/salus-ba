package videosession

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxWebhookBody bounds a LiveKit webhook's body (SPEC.md §2 "Webhook endpoint").
const maxWebhookBody = 64 << 10

// Sessions is what the handler calls (Service).
type Sessions interface {
	Join(ctx context.Context, c appointment.Caller, appointmentID string) (Join, error)
	End(ctx context.Context, c appointment.Caller, appointmentID string) (Status, error)
	Status(ctx context.Context, c appointment.Caller, appointmentID string) (Status, error)
	HandleWebhook(ctx context.Context, body []byte, authorization string) (EventKind, error)
}

// Handler serves the consultation's JSON API (api/openapi.yaml): the join, the end and the status for the
// signed-in participants, and LiveKit's webhook, which carries no session. Errors are {"error": "<code>"};
// internal error text goes to the log only, and no log line holds a token, room, identity or name.
type Handler struct {
	svc Sessions
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Sessions, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the consultation's routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /appointments/{id}/consultation/join", h.join)
	mux.HandleFunc("POST /appointments/{id}/consultation/end", h.end)
	mux.HandleFunc("GET /appointments/{id}/consultation", h.status)
	mux.HandleFunc("POST /webhooks/livekit", h.webhook)
}

// joinJSON is the ConsultationJoin schema of api/openapi.yaml.
type joinJSON struct {
	URL      string `json:"url"`
	Token    string `json:"token"`
	Expires  string `json:"expiresAt"`
	PeerName string `json:"peerName"`
	Role     Role   `json:"role"`
}

// statusJSON is the ConsultationStatus schema: the times and the reason only when set.
type statusJSON struct {
	State                State     `json:"state"`
	PatientJoinedAt      string    `json:"patientJoinedAt,omitempty"`
	PractitionerJoinedAt string    `json:"practitionerJoinedAt,omitempty"`
	EndedAt              string    `json:"endedAt,omitempty"`
	EndReason            EndReason `json:"endReason,omitempty"`
}

// instant is t in UTC as RFC 3339, empty for the zero time.
func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toStatusJSON(s Status) statusJSON {
	return statusJSON{
		State: s.State, PatientJoinedAt: instant(s.PatientJoinedAt), PractitionerJoinedAt: instant(s.PractitionerJoinedAt),
		EndedAt: instant(s.EndedAt), EndReason: s.EndReason,
	}
}

// join answers the caller's token for the appointment's room, never to be stored.
func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	j, err := h.svc.Join(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, joinJSON{URL: j.URL, Token: j.Token, Expires: instant(j.ExpiresAt), PeerName: j.PeerName, Role: j.Role})
}

// end is the practitioner's end of the consultation; it answers the consultation's state.
func (h *Handler) end(w http.ResponseWriter, r *http.Request) {
	h.answerStatus(w, r, h.svc.End)
}

// status answers where the consultation stands.
func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	h.answerStatus(w, r, h.svc.Status)
}

func (h *Handler) answerStatus(w http.ResponseWriter, r *http.Request, fn func(context.Context, appointment.Caller, string) (Status, error)) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := fn(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toStatusJSON(s))
}

// signedIn returns the caller, or answers 401 for nobody (500 when the caller cannot be resolved).
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
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

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrConsentRequired, "consent_required", http.StatusConflict},
	{ErrJoinWindowNotOpen, "join_window_not_open", http.StatusConflict},
	{ErrConsultationEnded, "consultation_ended", http.StatusConflict},
	{ErrPractitionerDidNotJoin, "practitioner_did_not_join", http.StatusConflict},
	{ErrNotStarted, "consultation_not_started", http.StatusConflict},
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			writeCode(w, c.status, c.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
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
