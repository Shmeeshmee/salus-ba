package invoice

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Quotes is what the HTTPHandler calls for the treatment quotes (package X1): Service.
type Quotes interface {
	SendQuote(ctx context.Context, c Caller, d QuoteDraft) (Quote, error)
	Quotes(ctx context.Context, c Caller, appointmentID string) ([]Quote, error)
	ReadQuote(ctx context.Context, c Caller, id string) (Quote, error)
	MyQuotes(ctx context.Context, c Caller) ([]Quote, error)
	QuotePDF(ctx context.Context, c Caller, id string) ([]byte, error)
	ClinicQuotePDF(ctx context.Context, c Caller, id string) ([]byte, error)
}

// quoteRoutes declares the quotes' routes on mux.
func (h *HTTPHandler) quoteRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/quotes", h.staff(h.quotes))
	mux.HandleFunc("POST /clinic/quotes", h.staff(h.sendQuote))
	mux.HandleFunc("GET /clinic/quotes/{id}", h.staff(h.readQuote))
	mux.HandleFunc("GET /clinic/quotes/{id}/pdf", h.staff(h.clinicQuotePDF))
	mux.HandleFunc("GET /me/quotes", h.myQuotes)
	mux.HandleFunc("GET /me/quotes/{id}/pdf", h.myQuotePDF)
}

// quoteJSON and sendQuoteJSON are the Quote and SendQuoteRequest schemas.
type quoteJSON struct {
	IssuedAt       time.Time  `json:"issuedAt"`
	ValidUntil     time.Time  `json:"validUntil"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	PatientID      string     `json:"patientId"`
	AppointmentID  string     `json:"appointmentId,omitempty"`
	State          QuoteState `json:"state"`
	Note           string     `json:"note"`
	Currency       string     `json:"currency"`
	Lines          []lineJSON `json:"lines"`
	TotalMinor     int64      `json:"totalMinor"`
}

type sendQuoteJSON struct {
	PatientID     string     `json:"patientId"`
	AppointmentID string     `json:"appointmentId"`
	Note          string     `json:"note"`
	Lines         []pickJSON `json:"lines"`
}

// QuoteJSON is the quote as the API's Quote schema serves it to the patient (the patient's export writes it too, D7).
func QuoteJSON(q Quote) any { return quoteToJSON(q) }

func quoteToJSON(q Quote) quoteJSON {
	out := quoteJSON{
		IssuedAt: q.IssuedAt, ValidUntil: q.ValidUntil, ID: q.ID, OrganizationID: q.OrganizationID, PatientID: q.PatientID,
		AppointmentID: q.AppointmentID, State: q.State, Note: q.Note, Currency: q.Currency, TotalMinor: q.TotalMinor,
		Lines: make([]lineJSON, len(q.Lines)),
	}
	for i, l := range q.Lines {
		out.Lines[i] = lineJSON{
			VATRateBP: l.VATRateBP, ServiceID: l.ChargeItemDefinitionID, Name: l.Name, UnitPriceMinor: l.UnitPriceMinor,
			DiscountMinor: l.DiscountMinor, TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		}
	}
	return out
}

func quotesJSON(list []Quote) []quoteJSON {
	out := make([]quoteJSON, len(list))
	for i, q := range list {
		out[i] = quoteToJSON(q)
	}
	return out
}

// sendQuote answers POST /clinic/quotes.
func (h *HTTPHandler) sendQuote(w http.ResponseWriter, r *http.Request, c Caller) {
	var in sendQuoteJSON
	if !decodeJSON(w, r, &in) {
		return
	}
	d := QuoteDraft{PatientID: in.PatientID, AppointmentID: in.AppointmentID, Note: in.Note}
	for _, p := range in.Lines {
		d.Picks = append(d.Picks, chargeitemdefinition.Pick{DefinitionID: p.ServiceID, DiscountMinor: p.DiscountMinor, Quantity: p.Quantity})
	}
	q, err := h.svc.SendQuote(r.Context(), c, d)
	if h.quoteFailed(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusCreated, quoteToJSON(q))
}

// quotes answers GET /clinic/quotes[?appointmentId=].
func (h *HTTPHandler) quotes(w http.ResponseWriter, r *http.Request, c Caller) {
	list, err := h.svc.Quotes(r.Context(), c, r.URL.Query().Get("appointmentId"))
	if h.quoteFailed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, quotesJSON(list))
}

// readQuote answers GET /clinic/quotes/{id}.
func (h *HTTPHandler) readQuote(w http.ResponseWriter, r *http.Request, c Caller) {
	q, err := h.svc.ReadQuote(r.Context(), c, r.PathValue("id"))
	if h.quoteFailed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, quoteToJSON(q))
}

// myQuotes answers GET /me/quotes.
func (h *HTTPHandler) myQuotes(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if c.OrganizationID == "" && c.PatientID == "" {
		writeCode(w, http.StatusForbidden, "profile_required")
		return
	}
	list, err := h.svc.MyQuotes(r.Context(), c)
	if h.quoteFailed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, quotesJSON(list))
}

// myQuotePDF answers GET /me/quotes/{id}/pdf: a patient's own quote, 404 to anyone else.
func (h *HTTPHandler) myQuotePDF(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if c.OrganizationID != "" || c.PatientID == "" {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	b, err := h.svc.QuotePDF(r.Context(), c, r.PathValue("id"))
	h.writeQuotePDF(w, r, b, err)
}

// clinicQuotePDF answers GET /clinic/quotes/{id}/pdf: a quote of the caller's clinic.
func (h *HTTPHandler) clinicQuotePDF(w http.ResponseWriter, r *http.Request, c Caller) {
	b, err := h.svc.ClinicQuotePDF(r.Context(), c, r.PathValue("id"))
	h.writeQuotePDF(w, r, b, err)
}

// writeQuotePDF writes b as the quote's PDF, or err.
func (h *HTTPHandler) writeQuotePDF(w http.ResponseWriter, r *http.Request, b []byte, err error) {
	if h.quoteFailed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="quote.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, _ = w.Write(b)
}

// quoteFailed is failed with a quote's own code for a draft that is no quote.
func (h *HTTPHandler) quoteFailed(w http.ResponseWriter, r *http.Request, err error) bool {
	if errors.Is(err, ErrInvalid) {
		writeCode(w, http.StatusBadRequest, "invalid_quote")
		return true
	}
	return h.failed(w, r, err)
}
