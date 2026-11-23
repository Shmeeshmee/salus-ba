package invoice_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/invoice"
)

// stub answers every call with its receipt, report or error, and keeps what it was asked.
type stub struct {
	err     error
	draft   invoice.Draft
	period  invoice.Period
	id      string
	caller  invoice.Caller
	method  invoice.PaymentMethod
	receipt invoice.Invoice
	report  invoice.Report
}

func (s *stub) Issue(_ context.Context, c invoice.Caller, d invoice.Draft) (invoice.Invoice, error) {
	s.caller, s.draft = c, d
	return s.receipt, s.err
}

func (s *stub) Read(_ context.Context, c invoice.Caller, id string) (invoice.Invoice, error) {
	s.caller, s.id = c, id
	return s.receipt, s.err
}

func (s *stub) List(_ context.Context, c invoice.Caller, p invoice.Period) ([]invoice.Invoice, error) {
	s.caller, s.period = c, p
	return []invoice.Invoice{s.receipt}, s.err
}

func (s *stub) Reverse(_ context.Context, c invoice.Caller, id string, m invoice.PaymentMethod) (invoice.Invoice, error) {
	s.caller, s.id, s.method = c, id, m
	return s.receipt, s.err
}

func (s *stub) Retry(_ context.Context, c invoice.Caller, id string) (invoice.Invoice, error) {
	s.caller, s.id = c, id
	return s.receipt, s.err
}

func (s *stub) PDF(_ context.Context, c invoice.Caller, id string) ([]byte, error) {
	s.caller, s.id = c, id
	return []byte("%PDF-synthetic"), s.err
}

func (s *stub) Mine(_ context.Context, c invoice.Caller) ([]invoice.Invoice, error) {
	s.caller = c
	return []invoice.Invoice{s.receipt}, s.err
}

func (s *stub) Report(_ context.Context, c invoice.Caller, p invoice.Period) (invoice.Report, error) {
	s.caller, s.period = c, p
	return s.report, s.err
}

// serve sends the request as caller (nil: nobody signed in) to the handler on the stub.
func serve(t *testing.T, s *stub, caller *invoice.Caller, method, target, body string) *httptest.ResponseRecorder {
	return serveWith(t, s, caller, httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body)))
}

// serveWith sends req as caller to the handler on the stub.
func serveWith(t *testing.T, s *stub, caller *invoice.Caller, req *http.Request) *httptest.ResponseRecorder {
	h := invoice.NewHTTPHandler(s, func(context.Context) (invoice.Caller, bool, error) {
		if caller == nil {
			return invoice.Caller{}, false, nil
		}
		return *caller, true, nil
	}, slog.New(slog.DiscardHandler))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func fiscalized() invoice.Invoice {
	vat := int32(1700)
	return invoice.Invoice{
		IssuedAt: at, FiscalizedAt: at, ID: "inv-1", OrganizationID: org, PatientID: patientA, AppointmentID: visit, ReversedByID: "inv-2",
		SellerLegalName: theSeller.LegalName, SellerTaxID: theSeller.TaxID, SellerAddress: theSeller.Address, IssuedByAccountID: admin,
		Currency: "BAM", FiscalID: "TEST-1", FiscalCode: "code-1", FiscalQR: "TEST:qr-1", Kind: invoice.KindReceipt,
		PaymentMethod: invoice.PaymentCash, FiscalStatus: invoice.FiscalFiscalized, Number: 7, TotalMinor: 1170,
		Lines: []invoice.Line{{ID: "line-1", ChargeItemDefinitionID: lab, Name: "Placeholder laboratory item", Quantity: 1, UnitPriceMinor: 1170, TotalMinor: 1170, VATRateBP: &vat}},
	}
}

const window = "?from=2026-11-21T00:00:00Z&to=2026-11-22T00:00:00Z"

// The Receipt schema, exactly: no seller, issuer or QR payload beyond the contract, absent fields left out.
func TestHTTP_ReadWritesTheReceiptSchema(t *testing.T) {
	rec := serve(t, &stub{receipt: fiscalized()}, &adminC, http.MethodGet, "/clinic/receipts/inv-1", "")
	want := `{"fiscalizedAt":"2026-11-21T09:00:00Z","issuedAt":"2026-11-21T09:00:00Z","id":"inv-1","patientId":"patient-a",
		"appointmentId":"appt-1","reversedById":"inv-2","currency":"BAM","fiscalId":"TEST-1","fiscalCode":"code-1","kind":"receipt",
		"paymentMethod":"cash","fiscalStatus":"fiscalized","lines":[{"vatRateBp":1700,"serviceId":"svc-lab",
		"name":"Placeholder laboratory item","unitPriceMinor":1170,"discountMinor":0,"totalMinor":1170,"quantity":1}],"number":7,"totalMinor":1170}`
	var got, wanted any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal([]byte(want), &wanted)
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(wanted)
	if rec.Code != http.StatusOK || string(g) != string(w) || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s\nwant %s", rec.Code, g, w)
	}
}

func TestHTTP_IssueAndReverseDecodeTheRequest(t *testing.T) {
	s := &stub{receipt: fiscalized()}
	body := `{"kind":"receipt","paymentMethod":"card","appointmentId":"appt-1","lines":[{"serviceId":"svc-exam","quantity":2,"discountMinor":500}]}`
	rec := serve(t, s, &adminC, http.MethodPost, "/clinic/receipts", body)
	if rec.Code != http.StatusCreated || s.draft.Kind != invoice.KindReceipt || s.draft.PaymentMethod != invoice.PaymentCard || s.draft.AppointmentID != visit ||
		len(s.draft.Picks) != 1 || s.draft.Picks[0].DefinitionID != exam || s.draft.Picks[0].Quantity != 2 || s.draft.Picks[0].DiscountMinor != 500 || s.caller != adminC {
		t.Errorf("POST = %d, draft %+v", rec.Code, s.draft)
	}
	rec = serve(t, s, &adminC, http.MethodPost, "/clinic/receipts/inv-1/reverse", `{"paymentMethod":"cash"}`)
	if rec.Code != http.StatusCreated || s.id != "inv-1" || s.method != invoice.PaymentCash {
		t.Errorf("reverse = %d, id %q, method %q", rec.Code, s.id, s.method)
	}
	for name, tt := range map[string]struct{ target, body string }{
		"an unknown field":          {"/clinic/receipts", `{"kind":"receipt","total":1}`},
		"no JSON":                   {"/clinic/receipts", `receipt`},
		"two values":                {"/clinic/receipts", `{} {}`},
		"an unknown payment method": {"/clinic/receipts/inv-1/reverse", `{"paymentMethod":"barter"}`},
	} {
		if rec := serve(t, s, &adminC, http.MethodPost, tt.target, tt.body); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_request") {
			t.Errorf("%s: %d %s, want 400 invalid_request", name, rec.Code, rec.Body)
		}
	}
}

func TestHTTP_MapsWhoAsksAndTheErrors(t *testing.T) {
	noProfile := invoice.Caller{AccountID: "acct-new"}
	tests := []struct {
		err          error
		caller       *invoice.Caller
		name         string
		method, path string
		body, code   string
		status       int
	}{
		{name: "nobody", method: http.MethodGet, path: "/clinic/receipts" + window, status: 401, code: "unauthenticated"},
		{name: "nobody, own receipts", method: http.MethodGet, path: "/me/receipts", status: 401, code: "unauthenticated"},
		{name: "a patient on a clinic route", caller: &patientC, method: http.MethodGet, path: "/clinic/receipts/inv-1/pdf", status: 404, code: "not_found"},
		{name: "staff on the patient's PDF", caller: &adminC, method: http.MethodGet, path: "/me/receipts/inv-1/pdf", status: 404, code: "not_found"},
		{name: "no profile yet", caller: &noProfile, method: http.MethodGet, path: "/me/receipts", status: 403, code: "profile_required"},
		{name: "staff's own receipts", caller: &adminC, err: invoice.ErrForbidden, method: http.MethodGet, path: "/me/receipts", status: 403, code: "forbidden"},
		{name: "a list without a period", caller: &adminC, method: http.MethodGet, path: "/clinic/receipts", status: 400, code: "invalid_request"},
		{name: "a period that ends before it begins", caller: &adminC, method: http.MethodGet, path: "/clinic/reports/receipts?from=2026-11-22T00:00:00Z&to=2026-11-21T00:00:00Z", status: 400, code: "invalid_request"},
		{name: "an unknown report format", caller: &adminC, method: http.MethodGet, path: "/clinic/reports/receipts" + window + "&format=xls", status: 400, code: "invalid_request"},
		{name: "not the admin", caller: &doctorC, err: invoice.ErrForbidden, method: http.MethodGet, path: "/clinic/receipts" + window, status: 403, code: "forbidden"},
		{name: "no such receipt", caller: &adminC, err: invoice.ErrNotFound, method: http.MethodGet, path: "/clinic/receipts/inv-9", status: 404, code: "not_found"},
		{name: "an invalid receipt", caller: &adminC, err: invoice.ErrInvalid, method: http.MethodPost, path: "/clinic/receipts", body: `{}`, status: 400, code: "invalid_receipt"},
		{name: "billing details missing", caller: &adminC, err: invoice.ErrSellerIncomplete, method: http.MethodPost, path: "/clinic/receipts", body: `{}`, status: 409, code: "billing_details_missing"},
		{name: "reversed already", caller: &adminC, err: invoice.ErrAlreadyReversed, method: http.MethodPost, path: "/clinic/receipts/inv-1/reverse", body: `{"paymentMethod":"cash"}`, status: 409, code: "already_reversed"},
		{name: "its report failed", caller: &adminC, err: invoice.ErrReportFailed, method: http.MethodPost, path: "/clinic/receipts/inv-1/reverse", body: `{"paymentMethod":"cash"}`, status: 409, code: "fiscal_report_failed"},
		{name: "a fault", caller: &adminC, err: errors.New("synthetic fault with a secret"), method: http.MethodGet, path: "/clinic/receipts/inv-1", status: 500},
	}
	for _, tt := range tests {
		rec := serve(t, &stub{err: tt.err}, tt.caller, tt.method, tt.path, tt.body)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.code) || strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("%s: %d %s, want %d %s", tt.name, rec.Code, rec.Body, tt.status, tt.code)
		}
	}
}

func TestHTTP_ListsReportsAndPDFs(t *testing.T) {
	s := &stub{receipt: fiscalized(), report: invoice.Report{
		Currency: "BAM", Language: "bs", Count: 3, TotalMinor: 9500,
		ByPaymentMethod: []invoice.MethodTotal{{PaymentMethod: invoice.PaymentCash, Count: 2}, {PaymentMethod: invoice.PaymentCard, Count: 1, TotalMinor: 9500}},
		ByService:       []invoice.ServiceTotal{{ChargeItemDefinitionID: exam, Name: "=Placeholder; \"examination\"", Quantity: 2, TotalMinor: 9500}},
	}}
	rec := serve(t, s, &adminC, http.MethodGet, "/clinic/receipts"+window, "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), `[{"`) || !s.period.From.Equal(at.Add(-9*time12h/12)) || s.period.To.Sub(s.period.From) != 2*time12h {
		t.Errorf("list = %d %s, period %+v", rec.Code, rec.Body, s.period)
	}
	rec = serve(t, s, &patientC, http.MethodGet, "/me/receipts", "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), `[{"`) || s.caller != patientC {
		t.Errorf("mine = %d %s", rec.Code, rec.Body)
	}
	rec = serve(t, s, &adminC, http.MethodGet, "/clinic/reports/receipts"+window, "")
	for _, want := range []string{`"count":3`, `"totalMinor":9500`, `"currency":"BAM"`, `"byPaymentMethod":[{"paymentMethod":"cash","count":2,"totalMinor":0}`, `"serviceId":"svc-exam"`, `"quantity":2`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("the report lacks %s: %s", want, rec.Body)
		}
	}
	rec = serve(t, s, &adminC, http.MethodGet, "/clinic/reports/receipts"+window+"&format=csv", "")
	// Semicolons and decimal commas, as Excel reads a file with Bosnian settings; a field holding either, a quote or a
	// line break is quoted (RFC 4180, with ; as the separator).
	wantCSV := "Vrsta;Šifra;Naziv;Broj računa;Količina;Iznos;Valuta\nUkupno;;;3;;95,00;BAM\nNačin plaćanja;cash;Gotovina;2;;0,00;BAM\n" +
		"Način plaćanja;card;Kartica;1;;95,00;BAM\nUsluga;svc-exam;\"'=Placeholder; \"\"examination\"\"\";;2;95,00;BAM\n"
	if rec.Body.String() != wantCSV || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/csv") {
		t.Errorf("csv = %q (%s)\nwant %q", rec.Body, rec.Header().Get("Content-Type"), wantCSV)
	}
	s.report = invoice.Report{Currency: "BAM", Language: "en", Count: 1, TotalMinor: -105}
	rec = serve(t, s, &adminC, http.MethodGet, "/clinic/reports/receipts"+window+"&format=csv", "")
	if want := "Section;Code;Name;Receipts;Quantity;Amount;Currency\nTotal;;;1;;-1,05;BAM\n"; rec.Body.String() != want {
		t.Errorf("csv in English = %q, want %q", rec.Body, want)
	}
	for name, tt := range map[string]struct {
		caller *invoice.Caller
		path   string
	}{"the clinic's": {&adminC, "/clinic/receipts/inv-1/pdf"}, "the patient's": {&patientC, "/me/receipts/inv-1/pdf"}} {
		rec = serve(t, s, tt.caller, http.MethodGet, tt.path, "")
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/pdf" || !strings.HasPrefix(rec.Body.String(), "%PDF-") ||
			rec.Header().Get("Cache-Control") != "no-store" || s.id != "inv-1" {
			t.Errorf("%s PDF = %d %s", name, rec.Code, rec.Header())
		}
	}
}

// The retry answers 202 with the receipt; the error of a receipt that did not fail is 409 fiscal_report_not_failed.
func TestHTTP_Retry(t *testing.T) {
	s := &stub{receipt: fiscalized()}
	rec := serve(t, s, &adminC, http.MethodPost, "/clinic/receipts/inv-1/retry", "")
	if rec.Code != http.StatusAccepted || s.id != "inv-1" || s.caller != adminC || !strings.Contains(rec.Body.String(), `"id":"inv-1"`) {
		t.Errorf("retry = %d %s", rec.Code, rec.Body)
	}
	s.err = invoice.ErrNotFailed
	if rec := serve(t, s, &adminC, http.MethodPost, "/clinic/receipts/inv-1/retry", ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "fiscal_report_not_failed") {
		t.Errorf("retry of a receipt that did not fail = %d %s", rec.Code, rec.Body)
	}
}

// The Idempotency-Key header goes into the draft; a key that is no UUID is refused.
func TestHTTP_IssueTakesTheIdempotencyKey(t *testing.T) {
	const key = "e1000000-0000-4000-8000-0000000000e1"
	body := `{"kind":"receipt","paymentMethod":"cash","appointmentId":"appt-1","lines":[{"serviceId":"svc-exam","quantity":1}]}`
	for name, tt := range map[string]struct {
		header, want string
		status       int
	}{
		"a UUID": {key, key, http.StatusCreated}, "none": {"", "", http.StatusCreated}, "no UUID": {"twice", "", http.StatusBadRequest},
	} {
		s := &stub{receipt: fiscalized()}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/clinic/receipts", strings.NewReader(body))
		if tt.header != "" {
			req.Header.Set("Idempotency-Key", tt.header)
		}
		rec := serveWith(t, s, &adminC, req)
		if rec.Code != tt.status || s.draft.IdempotencyKey != tt.want {
			t.Errorf("%s: POST = %d, key %q; want %d %q", name, rec.Code, s.draft.IdempotencyKey, tt.status, tt.want)
		}
	}
}
