package consent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body; a grant fits many times over.
const maxBody = 1 << 10

// Consents is what the handler calls (Service).
type Consents interface {
	Grant(ctx context.Context, c Caller, organizationID string) (Consent, error)
	Revoke(ctx context.Context, c Caller, id string) (Consent, error)
	Mine(ctx context.Context, c Caller) ([]Consent, error)
}

// Handler serves the patient's consents in the JSON API (api/openapi.yaml). Errors are {"error": "<code>"}
// with a fixed code per case; internal error text goes to the log only.
type Handler struct {
	svc Consents
	// caller is who is signed in for the request, and false for nobody (from the session and the patient's
	// profile, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Consents, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the consent routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /consents", h.grant)
	mux.HandleFunc("POST /consents/{id}/revoke", h.revoke)
	mux.HandleFunc("GET /me/consents", h.mine)
}

// consentJSON is the Consent schema of api/openapi.yaml.
type consentJSON struct {
	DateTime       time.Time  `json:"dateTime"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	Status         Status     `json:"status"`
}

// listJSON is the ConsentList schema.
type listJSON struct {
	Consents []consentJSON `json:"consents"`
}

// grantRequest is the GrantConsentRequest schema.
type grantRequest struct {
	OrganizationID string `json:"organizationId"`
}

func toJSON(c Consent) consentJSON {
	out := consentJSON{ID: c.ID, OrganizationID: c.OrganizationID, Status: c.Status, DateTime: c.DateTime.UTC()}
	if !c.RevokedAt.IsZero() {
		at := c.RevokedAt.UTC()
		out.RevokedAt = &at
	}
	return out
}

func (h *Handler) grant(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req grantRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.OrganizationID == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.Grant(r.Context(), c, req.OrganizationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	write(w, http.StatusCreated, toJSON(out))
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Revoke(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	write(w, http.StatusOK, toJSON(out))
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.Mine(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := listJSON{Consents: make([]consentJSON, 0, len(all))}
	for _, x := range all {
		out.Consents = append(out.Consents, toJSON(x))
	}
	write(w, http.StatusOK, out)
}

// signedIn returns the caller, or answers 401 for nobody (500 when the caller cannot be resolved).
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
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

// write sends v, never to be cached: whom a patient consults is health data.
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, v)
}

// decodeJSON reads one JSON object with known fields only, or answers 400 invalid_request.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrClinicNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrActive, "consent_active", http.StatusConflict},
	{ErrInactive, "consent_inactive", http.StatusConflict},
	{ErrProfileRequired, "profile_required", http.StatusConflict},
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
