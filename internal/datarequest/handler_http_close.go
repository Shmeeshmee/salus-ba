package datarequest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// closeMaxBody bounds the closure's request: a password of at most 128 characters and its JSON.
const closeMaxBody = 1024

// Closer is what the CloseHandler calls (Closure).
type Closer interface {
	CloseAccount(ctx context.Context, p Patient, password string) (Request, error)
}

// CloseHandler serves the patient's account closure (api/openapi.yaml, closeMyAccount). The password is never logged.
type CloseHandler struct {
	svc    Closer
	caller func(ctx context.Context) (Patient, bool, error)
	log    *slog.Logger
}

// NewCloseHandler returns the handler on svc; caller resolves the signed-in session's account.
func NewCloseHandler(svc Closer, caller func(ctx context.Context) (Patient, bool, error), log *slog.Logger) *CloseHandler {
	return &CloseHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the route on mux.
func (h *CloseHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /me/account/close", h.close)
}

func (h *CloseHandler) close(w http.ResponseWriter, r *http.Request) {
	p, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	case !ok:
		closeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Password *string `json:"password"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, closeMaxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.Password == nil {
		closeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	_, err = h.svc.CloseAccount(r.Context(), p, *in.Password)
	var limited *auth.RateLimitedError
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrWrongPassword):
		closeCode(w, http.StatusForbidden, "wrong_credentials")
	case errors.As(err, &limited): // too many wrong passwords; this one was not checked
		secs := int64((limited.RetryAfter + time.Second - 1) / time.Second) // rounded up, at least 1
		w.Header().Set("Retry-After", strconv.FormatInt(max(secs, 1), 10))
		closeCode(w, http.StatusTooManyRequests, "rate_limited")
	case errors.Is(err, ErrNotFound): // a staff account, or one closed already
		closeCode(w, http.StatusForbidden, "forbidden")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

func closeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
