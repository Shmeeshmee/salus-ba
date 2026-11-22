package invoice

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody is the largest request body read: a receipt of some hundred lines.
const maxBody = 64 << 10

// Receipts is what the HTTPHandler calls: Service.
type Receipts interface {
	Issue(ctx context.Context, c Caller, d Draft) (Invoice, error)
	Read(ctx context.Context, c Caller, id string) (Invoice, error)
	List(ctx context.Context, c Caller, p Period) ([]Invoice, error)
	Reverse(ctx context.Context, c Caller, id string, method PaymentMethod) (Invoice, error)
	PDF(ctx context.Context, c Caller, id string) ([]byte, error)
	Mine(ctx context.Context, c Caller) ([]Invoice, error)
	Report(ctx context.Context, c Caller, p Period) (Report, error)
}

// HTTPHandler serves the receipts on the JSON API (api/openapi.yaml, tag billing).
type HTTPHandler struct {
	svc    Receipts
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the HTTPHandler; caller gives who is signed in, false for nobody.
func NewHTTPHandler(svc Receipts, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the receipts' routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/receipts", h.staff(h.list))
	mux.HandleFunc("POST /clinic/receipts", h.staff(h.issue))
	mux.HandleFunc("GET /clinic/receipts/{id}", h.staff(h.read))
	mux.HandleFunc("POST /clinic/receipts/{id}/reverse", h.staff(h.reverse))
	mux.HandleFunc("GET /clinic/receipts/{id}/pdf", h.staff(h.pdf))
	mux.HandleFunc("GET /clinic/reports/receipts", h.staff(h.report))
	mux.HandleFunc("GET /me/receipts", h.mine)
	mux.HandleFunc("GET /me/receipts/{id}/pdf", h.minePDF)
}

// lineJSON, receiptJSON and the request types are the ReceiptLine, Receipt, IssueReceiptRequest, ReceiptPick and
// ReverseReceiptRequest schemas.
type lineJSON struct {
	VATRateBP      *int32 `json:"vatRateBp,omitempty"`
	ServiceID      string `json:"serviceId,omitempty"`
	Name           string `json:"name"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	DiscountMinor  int64  `json:"discountMinor"`
	TotalMinor     int64  `json:"totalMinor"`
	Quantity       int32  `json:"quantity"`
}

type receiptJSON struct {
	FiscalizedAt  *time.Time    `json:"fiscalizedAt,omitempty"`
	IssuedAt      time.Time     `json:"issuedAt"`
	ID            string        `json:"id"`
	PatientID     string        `json:"patientId,omitempty"`
	CompanyName   string        `json:"companyName,omitempty"`
	CompanyTaxID  string        `json:"companyTaxId,omitempty"`
	AppointmentID string        `json:"appointmentId,omitempty"`
	ReversesID    string        `json:"reversesId,omitempty"`
	ReversedByID  string        `json:"reversedById,omitempty"`
	Currency      string        `json:"currency"`
	FiscalID      string        `json:"fiscalId,omitempty"`
	FiscalCode    string        `json:"fiscalCode,omitempty"`
	Kind          Kind          `json:"kind"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	FiscalStatus  FiscalStatus  `json:"fiscalStatus"`
	Lines         []lineJSON    `json:"lines"`
	Number        int64         `json:"number"`
	TotalMinor    int64         `json:"totalMinor"`
}

type pickJSON struct {
	ServiceID     string `json:"serviceId"`
	DiscountMinor int64  `json:"discountMinor"`
	Quantity      int32  `json:"quantity"`
}

type issueJSON struct {
	Kind          Kind          `json:"kind"`
	PatientID     string        `json:"patientId"`
	CompanyName   string        `json:"companyName"`
	CompanyTaxID  string        `json:"companyTaxId"`
	AppointmentID string        `json:"appointmentId"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	Lines         []pickJSON    `json:"lines"`
}

func toJSON(inv Invoice) receiptJSON {
	out := receiptJSON{
		IssuedAt: inv.IssuedAt, ID: inv.ID, PatientID: inv.PatientID, CompanyName: inv.CompanyName, CompanyTaxID: inv.CompanyTaxID,
		AppointmentID: inv.AppointmentID, ReversesID: inv.ReversesID, ReversedByID: inv.ReversedByID, Currency: inv.Currency,
		FiscalID: inv.FiscalID, FiscalCode: inv.FiscalCode, Kind: inv.Kind, PaymentMethod: inv.PaymentMethod,
		FiscalStatus: inv.FiscalStatus, Number: inv.Number, TotalMinor: inv.TotalMinor, Lines: make([]lineJSON, len(inv.Lines)),
	}
	if !inv.FiscalizedAt.IsZero() {
		out.FiscalizedAt = &inv.FiscalizedAt
	}
	for i, l := range inv.Lines {
		out.Lines[i] = lineJSON{
			VATRateBP: l.VATRateBP, ServiceID: l.ChargeItemDefinitionID, Name: l.Name, UnitPriceMinor: l.UnitPriceMinor,
			DiscountMinor: l.DiscountMinor, TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		}
	}
	return out
}

func listJSON(list []Invoice) []receiptJSON {
	out := make([]receiptJSON, len(list))
	for i, inv := range list {
		out[i] = toJSON(inv)
	}
	return out
}

// staff wraps a clinic route: 401 for nobody, 404 for an account that is staff of no clinic (a patient).
func (h *HTTPHandler) staff(next func(http.ResponseWriter, *http.Request, Caller)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := h.signedIn(w, r)
		switch {
		case !ok:
		case c.OrganizationID == "":
			writeCode(w, http.StatusNotFound, "not_found")
		default:
			next(w, r, c)
		}
	}
}

// list answers GET /clinic/receipts?from=&to=.
func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request, c Caller) {
	p, ok := period(r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	list, err := h.svc.List(r.Context(), c, p)
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

// issue answers POST /clinic/receipts.
func (h *HTTPHandler) issue(w http.ResponseWriter, r *http.Request, c Caller) {
	var in issueJSON
	if !decodeJSON(w, r, &in) {
		return
	}
	d := Draft{
		PatientID: in.PatientID, CompanyName: strings.TrimSpace(in.CompanyName), CompanyTaxID: in.CompanyTaxID,
		AppointmentID: in.AppointmentID, Kind: in.Kind, PaymentMethod: in.PaymentMethod,
	}
	for _, p := range in.Lines {
		d.Picks = append(d.Picks, chargeitemdefinition.Pick{DefinitionID: p.ServiceID, DiscountMinor: p.DiscountMinor, Quantity: p.Quantity})
	}
	inv, err := h.svc.Issue(r.Context(), c, d)
	if h.failed(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusCreated, toJSON(inv))
}

// read answers GET /clinic/receipts/{id}.
func (h *HTTPHandler) read(w http.ResponseWriter, r *http.Request, c Caller) {
	inv, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(inv))
}

// reverse answers POST /clinic/receipts/{id}/reverse.
func (h *HTTPHandler) reverse(w http.ResponseWriter, r *http.Request, c Caller) {
	var in struct {
		PaymentMethod PaymentMethod `json:"paymentMethod"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validMethod(in.PaymentMethod) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	inv, err := h.svc.Reverse(r.Context(), c, r.PathValue("id"), in.PaymentMethod)
	if h.failed(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusCreated, toJSON(inv))
}

// pdf answers GET /clinic/receipts/{id}/pdf.
func (h *HTTPHandler) pdf(w http.ResponseWriter, r *http.Request, c Caller) {
	b, err := h.svc.PDF(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	writePDF(w, b)
}

// report answers GET /clinic/reports/receipts?from=&to=&format=.
func (h *HTTPHandler) report(w http.ResponseWriter, r *http.Request, c Caller) {
	p, ok := period(r)
	format := r.URL.Query().Get("format")
	if !ok || (format != "" && format != "json" && format != "csv") {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	rep, err := h.svc.Report(r.Context(), c, p)
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="receipts-report.csv"`)
		if err := writeCSV(w, rep); err != nil {
			h.log.ErrorContext(r.Context(), "invoice: the report's CSV was cut short", "error", err)
		}
		return
	}
	httpx.JSON(w, http.StatusOK, reportJSON(rep))
}

// reportJSON is the ReceiptsReport schema.
func reportJSON(rep Report) map[string]any {
	type method struct {
		PaymentMethod PaymentMethod `json:"paymentMethod"`
		Count         int           `json:"count"`
		TotalMinor    int64         `json:"totalMinor"`
	}
	type service struct {
		ServiceID  string `json:"serviceId,omitempty"`
		Name       string `json:"name"`
		Quantity   int64  `json:"quantity"`
		TotalMinor int64  `json:"totalMinor"`
	}
	methods, services := make([]method, len(rep.ByPaymentMethod)), make([]service, len(rep.ByService))
	for i, m := range rep.ByPaymentMethod {
		methods[i] = method(m)
	}
	for i, s := range rep.ByService {
		services[i] = service{ServiceID: s.ChargeItemDefinitionID, Name: s.Name, Quantity: s.Quantity, TotalMinor: s.TotalMinor}
	}
	return map[string]any{
		"from": rep.Period.From, "to": rep.Period.To, "count": rep.Count, "totalMinor": rep.TotalMinor, "currency": rep.Currency,
		"byPaymentMethod": methods, "byService": services,
	}
}

// writeCSV writes the report for a spreadsheet: a header of codes, then the total, a row per payment method and a
// row per service, amounts in the currency's minor unit. A name that a spreadsheet would read as a formula is
// quoted with an apostrophe.
func writeCSV(w io.Writer, rep Report) error {
	cw := csv.NewWriter(w)
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	rows := [][]string{
		{"section", "key", "name", "count", "quantity", "total_minor", "currency"},
		{"total", "", "", n(int64(rep.Count)), "", n(rep.TotalMinor), rep.Currency},
	}
	for _, m := range rep.ByPaymentMethod {
		rows = append(rows, []string{"payment_method", string(m.PaymentMethod), "", n(int64(m.Count)), "", n(m.TotalMinor), rep.Currency})
	}
	for _, s := range rep.ByService {
		name := s.Name
		if name != "" && strings.ContainsAny(name[:1], "=+-@\t\r") {
			name = "'" + name
		}
		rows = append(rows, []string{"service", s.ChargeItemDefinitionID, name, "", n(s.Quantity), n(s.TotalMinor), rep.Currency})
	}
	if err := cw.WriteAll(rows); err != nil {
		return err
	}
	return cw.Error()
}

// mine answers GET /me/receipts.
func (h *HTTPHandler) mine(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if c.OrganizationID == "" && c.PatientID == "" {
		writeCode(w, http.StatusForbidden, "profile_required")
		return
	}
	list, err := h.svc.Mine(r.Context(), c)
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

// minePDF answers GET /me/receipts/{id}/pdf: a patient's own receipt, 404 to anyone else.
func (h *HTTPHandler) minePDF(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if c.OrganizationID != "" || c.PatientID == "" {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	b, err := h.svc.PDF(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	writePDF(w, b)
}

func writePDF(w http.ResponseWriter, b []byte) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="receipt.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, _ = w.Write(b)
}

// period reads the from and to times (RFC 3339) of a half-open range; false when one is missing or malformed, or to
// is not after from.
func period(r *http.Request) (Period, bool) {
	v := r.URL.Query()
	from, errFrom := time.Parse(time.RFC3339, v.Get("from"))
	to, errTo := time.Parse(time.RFC3339, v.Get("to"))
	return Period{From: from, To: to}, errFrom == nil && errTo == nil && to.After(from)
}

func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	}
	return c, ok && err == nil
}

// decodeJSON reads the body into v; 400 invalid_request for one that is unreadable, has an unknown field or more
// than one value.
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
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrInvalid, "invalid_receipt", http.StatusBadRequest},
	{ErrAlreadyReversed, "already_reversed", http.StatusConflict},
	{ErrSellerIncomplete, "billing_details_missing", http.StatusConflict},
}

// failed writes the error's status and reports whether there was one.
func (h *HTTPHandler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	for _, e := range errorCodes {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return true
		}
	}
	httpx.InternalError(w, r, h.log, err)
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
