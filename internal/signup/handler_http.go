package signup

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a request body: an e-mail, a clinic name and a language fit many times over.
const maxBody = 4 << 10

// Signups is what the handler calls (Service).
type Signups interface {
	Start(ctx context.Context, r Request, clientIP string) error
	Verify(ctx context.Context, token, clientIP string) (Signup, error)
	Setup(ctx context.Context, c Caller) (Setup, error)
	SaveSetup(ctx context.Context, c Caller, in Setup) (Setup, error)
}

// Handler serves the signup and the setup wizard's progress (api/openapi.yaml, tag signup). Errors are
// {"error": "<code>"}.
type Handler struct {
	svc    Signups
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	// trustedProxies may set X-Forwarded-For (TRUSTED_PROXY_CIDRS), for the rate limits.
	trustedProxies []netip.Prefix
}

// NewHandler returns the handler on svc; caller resolves the signed-in staff session.
func NewHandler(svc Signups, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger, trustedProxies []netip.Prefix) *Handler {
	return &Handler{svc: svc, caller: caller, log: log, trustedProxies: trustedProxies}
}

// Routes declares the routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /signup", h.start)
	mux.HandleFunc("POST /signup/verify", h.verify)
	mux.HandleFunc("GET /clinic/setup", h.setup)
	mux.HandleFunc("PUT /clinic/setup", h.saveSetup)
}

// requestJSON is the SignupRequest schema.
type requestJSON struct {
	Email      string `json:"email"`
	ClinicName string `json:"clinicName"`
	Language   string `json:"language"`
}

// setupJSON is the ClinicSetup schema: every step, required.
type setupJSON struct {
	Details      *StepState `json:"details"`
	Billing      *StepState `json:"billing"`
	Hours        *StepState `json:"hours"`
	Practitioner *StepState `json:"practitioner"`
	Services     *StepState `json:"services"`
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var in requestJSON
	if !decode(w, r, &in) {
		return
	}
	err := h.svc.Start(r.Context(), Request(in), httpx.ClientIP(r, h.trustedProxies))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, err := h.svc.Verify(r.Context(), in.Token, httpx.ClientIP(r, h.trustedProxies)); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Setup(r.Context(), c)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(s))
}

func (h *Handler) saveSetup(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in setupJSON
	if !decode(w, r, &in) {
		return
	}
	if in.Details == nil || in.Billing == nil || in.Hours == nil || in.Practitioner == nil || in.Services == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s, err := h.svc.SaveSetup(r.Context(), c, Setup{Steps: map[Step]StepState{
		StepDetails: *in.Details, StepBilling: *in.Billing, StepHours: *in.Hours, StepPractitioner: *in.Practitioner,
		StepServices: *in.Services,
	}})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toJSON(s))
}

func toJSON(s Setup) setupJSON {
	state := func(step Step) *StepState {
		v := s.Steps[step]
		return &v
	}
	return setupJSON{
		Details: state(StepDetails), Billing: state(StepBilling), Hours: state(StepHours),
		Practitioner: state(StepPractitioner), Services: state(StepServices),
	}
}

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

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var limited *RateLimitedError
	switch {
	case errors.As(err, &limited):
		secs := int64((limited.RetryAfter + time.Second - 1) / time.Second) // rounded up, at least 1
		w.Header().Set("Retry-After", strconv.FormatInt(max(secs, 1), 10))
		writeCode(w, http.StatusTooManyRequests, "rate_limited")
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_signup")
	case errors.Is(err, ErrInvalidToken):
		writeCode(w, http.StatusBadRequest, "invalid_verification")
	case errors.Is(err, ErrInvalidSetup):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

// decode reads a JSON body into v, and is false (with the response written) for a body that is not one.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
