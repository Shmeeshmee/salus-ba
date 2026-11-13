package notification

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// maxBody bounds a request: a device token of 4096 bytes, or a webhook's URL of 2048, and their JSON.
const maxBody = 8192

// Devices and Webhooks are what the handler calls (Chat).
type Devices interface {
	RegisterDevice(ctx context.Context, c Caller, p push.Platform, token string) error
	UnregisterDevice(ctx context.Context, c Caller, p push.Platform, token string) error
	Webhooks(ctx context.Context, c Caller) ([]Endpoint, error)
	CreateWebhook(ctx context.Context, c Caller, in NewEndpoint) (Endpoint, string, error)
	DeleteWebhook(ctx context.Context, c Caller, id string) error
}

// HTTPHandler serves the app's device registration and the admin's webhooks (api/openapi.yaml, tag chat). Errors
// are {"error": "<code>"}; no token, URL or secret is logged.
type HTTPHandler struct {
	svc    Devices
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc; caller resolves the signed-in session, with whether it is the clinic's
// admin.
func NewHTTPHandler(svc Devices, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /me/devices", h.device(Devices.RegisterDevice))
	mux.HandleFunc("DELETE /me/devices", h.device(Devices.UnregisterDevice))
	mux.HandleFunc("GET /clinic/webhooks", h.webhooks)
	mux.HandleFunc("POST /clinic/webhooks", h.createWebhook)
	mux.HandleFunc("DELETE /clinic/webhooks/{id}", h.deleteWebhook)
}

// deviceJSON is the PushDevice schema.
type deviceJSON struct {
	Platform *string `json:"platform"`
	Token    *string `json:"token"`
}

func (h *HTTPHandler) device(call func(Devices, context.Context, Caller, push.Platform, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := h.signedIn(w, r)
		if !ok {
			return
		}
		var in deviceJSON
		if !decode(w, r, &in) || in.Platform == nil || in.Token == nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		if err := call(h.svc, r.Context(), c, push.Platform(*in.Platform), *in.Token); err != nil {
			h.writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// webhookListJSON is the WebhookList schema.
type webhookListJSON struct {
	Webhooks []Endpoint `json:"webhooks"`
}

func (h *HTTPHandler) webhooks(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Webhooks(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if list == nil {
		list = []Endpoint{}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, webhookListJSON{Webhooks: list})
}

// newWebhookJSON is the NewWebhook schema: url and events required, include_clinical off unless given.
type newWebhookJSON struct {
	URL             *string  `json:"url"`
	Events          []string `json:"events"`
	IncludeClinical bool     `json:"include_clinical"`
}

// CreatedWebhook is the CreatedWebhook schema: the endpoint with its secret, shown this once.
type CreatedWebhook struct {
	CreatedAt       time.Time `json:"created_at"`
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Secret          string    `json:"secret"`
	Events          []string  `json:"events"`
	IncludeClinical bool      `json:"include_clinical"`
}

func (h *HTTPHandler) createWebhook(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in newWebhookJSON
	if !decode(w, r, &in) || in.URL == nil || in.Events == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	e, secret, err := h.svc.CreateWebhook(r.Context(), c, NewEndpoint{URL: *in.URL, Events: in.Events, IncludeClinical: in.IncludeClinical})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusCreated, CreatedWebhook{CreatedAt: e.CreatedAt, ID: e.ID, URL: e.URL, Secret: secret, Events: e.Events, IncludeClinical: e.IncludeClinical})
}

func (h *HTTPHandler) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteWebhook(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decode reads exactly one JSON object of known fields, at most maxBody bytes, into v.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && dec.Decode(&struct{}{}) == io.EOF
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

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_request")
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
