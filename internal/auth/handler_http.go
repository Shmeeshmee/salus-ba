package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body; an e-mail and a 128-character password fit many times over.
const maxBody = 4 << 10

// AccountService is what the handlers call (Service).
type AccountService interface {
	Register(ctx context.Context, email, password string) (Account, error)
	Me(ctx context.Context, id string) (Account, error)
	Login(ctx context.Context, email, password, clientIP string) (LoginResult, error)
	Enrol(ctx context.Context, challenge string, sess Session) (Enrolment, error)
	Confirm(ctx context.Context, challenge string, sess Session, code string) (LoginResult, error)
	Verify(ctx context.Context, challenge, code string) (LoginResult, error)
	Logout(ctx context.Context, sess Session) error
	Authenticate(ctx context.Context, token string) (Session, error)
	Invite(ctx context.Context, sess Session, in Invite) (Invite, error)
	ListInvites(ctx context.Context, sess Session) ([]Invite, error)
	AcceptInvite(ctx context.Context, token, password string) (Acceptance, error)
}

// Handler serves the auth endpoints of api/openapi.yaml and the session middleware. Errors are
// {"error": "<code>"} with a fixed code per case; internal error text goes to the log only.
type Handler struct {
	svc AccountService
	log *slog.Logger
	// trustedProxies may set X-Forwarded-For (TRUSTED_PROXY_CIDRS); from anyone else it is ignored.
	trustedProxies []netip.Prefix
}

// NewHandler returns the handler on svc.
func NewHandler(svc AccountService, log *slog.Logger, trustedProxies []netip.Prefix) *Handler {
	return &Handler{svc: svc, log: log, trustedProxies: trustedProxies}
}

// Routes declares the auth routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("GET /auth/me", h.me)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.HandleFunc("POST /auth/mfa/enrol", h.enrol)
	mux.HandleFunc("POST /auth/mfa/confirm", h.confirm)
	mux.HandleFunc("POST /auth/mfa/verify", h.verify)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("POST /invites", h.invite)
	mux.HandleFunc("GET /invites", h.listInvites)
	mux.HandleFunc("POST /invites/accept", h.acceptInvite)
}

// accountJSON is the Account schema: the password hash never crosses this boundary.
type accountJSON struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           Role   `json:"role"`
	OrganizationID string `json:"organizationId,omitempty"`
	PractitionerID string `json:"practitionerId,omitempty"`
	MFAEnrolled    bool   `json:"mfaEnrolled"`
}

func toJSON(a Account) accountJSON {
	return accountJSON{
		ID: a.ID, Email: a.Email, Role: a.Role, OrganizationID: a.OrganizationID, PractitionerID: a.PractitionerID, MFAEnrolled: a.MFAEnrolled(),
	}
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

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalidEmail, "invalid_email", http.StatusBadRequest},
	{ErrInvalidPassword, "invalid_password", http.StatusBadRequest},
	{ErrEmailTaken, "email_taken", http.StatusConflict},
	{ErrNotFound, "unauthenticated", http.StatusUnauthorized}, // the session's account is gone
	{ErrUnauthenticated, "unauthenticated", http.StatusUnauthorized},
	{ErrWrongCredentials, "wrong_credentials", http.StatusUnauthorized},
	{ErrInvalidChallenge, "invalid_challenge", http.StatusUnauthorized},
	{ErrWrongCode, "wrong_code", http.StatusUnauthorized},
	{ErrAlreadyEnrolled, "already_enrolled", http.StatusConflict},
	{ErrNotEnrolled, "not_enrolled", http.StatusConflict},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrInvalidRole, "invalid_role", http.StatusBadRequest},
	{ErrInvalidName, "invalid_name", http.StatusBadRequest},
	{ErrInvalidInvite, "invalid_invite", http.StatusNotFound},
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var limited *RateLimitedError
	if errors.As(err, &limited) {
		secs := int64((limited.RetryAfter + time.Second - 1) / time.Second) // rounded up, at least 1
		w.Header().Set("Retry-After", strconv.FormatInt(max(secs, 1), 10))
		writeCode(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
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
