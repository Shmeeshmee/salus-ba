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

// maxBillingBody bounds a billing details request: four short fields.
const maxBillingBody = 2048

// BillingService is what the billing details' handler calls (Billing).
type BillingService interface {
	Details(ctx context.Context, c Caller) (BillingDetails, error)
	SetDetails(ctx context.Context, c Caller, in BillingDetails) (BillingDetails, error)
}

// BillingHandler serves the clinic's billing details (api/openapi.yaml, tag billing). Errors are {"error": "<code>"}.
type BillingHandler struct {
	svc    BillingService
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewBillingHandler returns the handler on svc; caller resolves the signed-in session.
func NewBillingHandler(svc BillingService, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *BillingHandler {
	return &BillingHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *BillingHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/billing-details", h.details)
	mux.HandleFunc("PUT /clinic/billing-details", h.setDetails)
}

// billingJSON is the BillingDetails schema; vatRegistered is required in a request.
type billingJSON struct {
	VATRegistered *bool  `json:"vatRegistered"`
	LegalName     string `json:"legalName,omitempty"`
	TaxID         string `json:"taxId,omitempty"`
	BillingEmail  string `json:"billingEmail,omitempty"`
}

func (h *BillingHandler) details(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	if err != nil || !ok {
		h.write(w, r, BillingDetails{}, err, !ok)
		return
	}
	d, err := h.svc.Details(r.Context(), c)
	h.write(w, r, d, err, false)
}

func (h *BillingHandler) setDetails(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	if err != nil || !ok {
		h.write(w, r, BillingDetails{}, err, !ok)
		return
	}
	var in billingJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBillingBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.VATRegistered == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	d, err := h.svc.SetDetails(r.Context(), c, BillingDetails{LegalName: in.LegalName, TaxID: in.TaxID, BillingEmail: in.BillingEmail, VATRegistered: *in.VATRegistered})
	h.write(w, r, d, err, false)
}

// write answers d, or the code of err, or 401 for a caller who is signed out.
func (h *BillingHandler) write(w http.ResponseWriter, r *http.Request, d BillingDetails, err error, signedOut bool) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidBillingDetails):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case signedOut:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	default:
		w.Header().Set("Cache-Control", "no-store")
		httpx.JSON(w, http.StatusOK, billingJSON{VATRegistered: &d.VATRegistered, LegalName: d.LegalName, TaxID: d.TaxID, BillingEmail: d.BillingEmail})
	}
}
