package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body; an e-mail and a 128-character password fit many times over.
const maxBody = 4 << 10

// AccountService is what the handlers call (Service).
type AccountService interface {
	Register(ctx context.Context, email, password string) (Account, error)
	Me(ctx context.Context, id string) (Account, error)
}

// Handler serves the account endpoints of api/openapi.yaml. Errors are {"error": "<code>"} with a fixed code
// per case; internal error text goes to the log only.
type Handler struct {
	svc AccountService
	log *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc AccountService, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Routes declares the account routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("GET /auth/me", h.me)
}

// accountJSON is the Account schema: the password hash never crosses this boundary.
type accountJSON struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           Role   `json:"role"`
	OrganizationID string `json:"organizationId,omitempty"`
	MFAEnrolled    bool   `json:"mfaEnrolled"`
}

func toJSON(a Account) accountJSON {
	return accountJSON{ID: a.ID, Email: a.Email, Role: a.Role, OrganizationID: a.OrganizationID, MFAEnrolled: a.MFAEnrolled()}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	a, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toJSON(a))
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	id := AccountIDFrom(r.Context())
	if id == "" {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	a, err := h.svc.Me(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toJSON(a))
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

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		writeCode(w, http.StatusBadRequest, "invalid_email")
	case errors.Is(err, ErrInvalidPassword):
		writeCode(w, http.StatusBadRequest, "invalid_password")
	case errors.Is(err, ErrEmailTaken):
		writeCode(w, http.StatusConflict, "email_taken")
	case errors.Is(err, ErrNotFound): // the session's account is gone
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
