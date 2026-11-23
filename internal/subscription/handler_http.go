package subscription

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Memberships is what the handler calls (Service).
type Memberships interface {
	Current(ctx context.Context, c Caller) (Subscription, error)
	Invoices(ctx context.Context, c Caller) ([]Invoice, error)
	PDF(ctx context.Context, c Caller, id string) ([]byte, error)
}

// HTTPHandler serves the clinic's membership and Salus's invoices to it (api/openapi.yaml, tag billing). Errors are
// {"error": "<code>"}.
type HTTPHandler struct {
	svc    Memberships
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc; caller resolves the signed-in session.
func NewHTTPHandler(svc Memberships, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/subscription", h.current)
	mux.HandleFunc("GET /clinic/subscription/invoices", h.invoices)
	mux.HandleFunc("GET /clinic/subscription/invoices/{id}/pdf", h.pdf)
}

// subscriptionJSON is the Subscription schema.
type subscriptionJSON struct {
	TrialEndsAt        *time.Time `json:"trialEndsAt,omitempty"`
	CurrentPeriodStart *time.Time `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"currentPeriodEnd,omitempty"`
	PastDueSince       *time.Time `json:"pastDueSince,omitempty"`
	PlanCode           string     `json:"planCode"`
	PlanName           string     `json:"planName"`
	Currency           string     `json:"currency"`
	Status             Status     `json:"status"`
	PricePerSeatMinor  int64      `json:"pricePerSeatMinor"`
}

// invoiceJSON is the SubscriptionInvoice schema.
type invoiceJSON struct {
	IssuedAt    time.Time     `json:"issuedAt"`
	DueAt       time.Time     `json:"dueAt"`
	PaidAt      *time.Time    `json:"paidAt,omitempty"`
	ID          string        `json:"id"`
	PeriodStart string        `json:"periodStart"`
	PeriodEnd   string        `json:"periodEnd"`
	Currency    string        `json:"currency"`
	Status      InvoiceStatus `json:"status"`
	Number      int64         `json:"number"`
	AmountMinor int64         `json:"amountMinor"`
	Seats       int32         `json:"seats"`
}

func (h *HTTPHandler) current(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Current(r.Context(), c)
	h.write(w, r, subscriptionJSON{
		TrialEndsAt: optional(s.TrialEndsAt), CurrentPeriodStart: optional(s.CurrentPeriodStart), CurrentPeriodEnd: optional(s.CurrentPeriodEnd),
		PastDueSince: optional(s.PastDueSince), PlanCode: s.PlanCode, PlanName: s.Plan.Name, Currency: s.Plan.Currency, Status: s.Status,
		PricePerSeatMinor: s.Plan.PricePerSeatMinor,
	}, err)
}

func (h *HTTPHandler) invoices(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Invoices(r.Context(), c)
	out := make([]invoiceJSON, 0, len(list))
	for _, inv := range list {
		out = append(out, invoiceJSON{
			IssuedAt: inv.IssuedAt.UTC(), DueAt: inv.DueAt.UTC(), PaidAt: optional(inv.PaidAt), ID: inv.ID,
			PeriodStart: inv.PeriodStart.Format(time.DateOnly), PeriodEnd: inv.PeriodEnd.Format(time.DateOnly),
			Currency: inv.Currency, Status: inv.Status, Number: inv.Number, AmountMinor: inv.AmountMinor, Seats: inv.Seats,
		})
	}
	h.write(w, r, out, err)
}

func (h *HTTPHandler) pdf(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	b, err := h.svc.PDF(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.write(w, r, nil, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="salus-invoice.pdf"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
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

// optional is t in UTC, nil for the zero time.
func optional(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	t = t.UTC()
	return &t
}
