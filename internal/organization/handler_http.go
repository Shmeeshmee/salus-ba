package organization

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxSettingsBody bounds a chat settings request: 21 stretches and the numbers.
const maxSettingsBody = 4096

// Chats is what the chat settings' handler calls (Chat).
type Chats interface {
	Settings(ctx context.Context, c Caller) (ChatSettings, error)
	SetSettings(ctx context.Context, c Caller, in ChatSettings) (ChatSettings, error)
	Hours(ctx context.Context, c Caller, id string) (ChatHours, error)
}

// HTTPHandler serves the clinic's chat settings and hours (api/openapi.yaml, tag chat). Errors are
// {"error": "<code>"}.
type HTTPHandler struct {
	svc    Chats
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc; caller resolves the signed-in session (with whether it is the
// clinic's admin, or the patient's profile).
func NewHTTPHandler(svc Chats, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/chat-settings", h.settings)
	mux.HandleFunc("PUT /clinic/chat-settings", h.setSettings)
	mux.HandleFunc("GET /organizations/{id}/chat-hours", h.hours)
}

func (h *HTTPHandler) settings(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Settings(r.Context(), c)
	h.write(w, r, s, err)
}

// settingsJSON is the ChatSettings schema as a request: every field required.
type settingsJSON struct {
	Hours             *[]ChatHour `json:"hours"`
	ResponseTimeHours *int        `json:"response_time_hours"`
	EmailAfterMinutes *int        `json:"email_after_minutes"`
}

func (h *HTTPHandler) setSettings(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in settingsJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSettingsBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.Hours == nil || in.ResponseTimeHours == nil || in.EmailAfterMinutes == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s, err := h.svc.SetSettings(r.Context(), c, ChatSettings{Hours: *in.Hours, ResponseTimeHours: *in.ResponseTimeHours, EmailAfterMinutes: *in.EmailAfterMinutes})
	h.write(w, r, s, err)
}

func (h *HTTPHandler) hours(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Hours(r.Context(), c, r.PathValue("id"))
	h.write(w, r, s, err)
}

// signedIn returns the caller, or answers 401 (or 500) and false.
func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Caller{}, false
	}
	return c, true
}

// write answers v, or the code of err.
func (h *HTTPHandler) write(w http.ResponseWriter, r *http.Request, v any, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidChatSettings):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		httpx.JSON(w, http.StatusOK, v)
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
