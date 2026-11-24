package invoice_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/invoice"
)

func (s *stub) SendQuote(_ context.Context, c invoice.Caller, d invoice.QuoteDraft) (invoice.Quote, error) {
	s.caller, s.qdraft = c, d
	return s.quote, s.err
}

func (s *stub) Quotes(_ context.Context, c invoice.Caller, appointmentID string) ([]invoice.Quote, error) {
	s.caller, s.id = c, appointmentID
	return []invoice.Quote{s.quote}, s.err
}

func (s *stub) ReadQuote(_ context.Context, c invoice.Caller, id string) (invoice.Quote, error) {
	s.caller, s.id = c, id
	return s.quote, s.err
}

func (s *stub) MyQuotes(_ context.Context, c invoice.Caller) ([]invoice.Quote, error) {
	s.caller = c
	return []invoice.Quote{s.quote}, s.err
}

func (s *stub) QuotePDF(_ context.Context, c invoice.Caller, id string) ([]byte, error) {
	s.caller, s.id = c, id
	return []byte("%PDF-synthetic"), s.err
}

func (s *stub) ClinicQuotePDF(_ context.Context, c invoice.Caller, id string) ([]byte, error) {
	s.caller, s.id = c, "clinic:"+id
	return []byte("%PDF-synthetic"), s.err
}

func sentQuote() invoice.Quote {
	return invoice.Quote{
		IssuedAt: at, ValidUntil: at.Add(30 * 24 * time.Hour), ID: "q-1", OrganizationID: org, PatientID: patientA, AppointmentID: visit,
		Note: "Placeholder note", Currency: "BAM", State: invoice.QuoteSent, TotalMinor: 5000,
		Lines: []invoice.Line{{ChargeItemDefinitionID: exam, Name: "Placeholder examination", Quantity: 1, UnitPriceMinor: 5000, TotalMinor: 5000}},
	}
}

// POST /clinic/quotes decodes the SendQuoteRequest and answers the Quote schema with 201.
func TestHandler_SendQuote(t *testing.T) {
	s := &stub{quote: sentQuote()}
	body := `{"patientId":"` + patientA + `","appointmentId":"` + visit + `","note":"Placeholder note","lines":[{"serviceId":"` + exam + `","quantity":2,"discountMinor":100}]}`
	rec := serve(t, s, &adminC, http.MethodPost, "/clinic/quotes", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if d := s.qdraft; d.PatientID != patientA || d.AppointmentID != visit || d.Note != "Placeholder note" || len(d.Picks) != 1 ||
		d.Picks[0].DefinitionID != exam || d.Picks[0].Quantity != 2 || d.Picks[0].DiscountMinor != 100 {
		t.Errorf("draft = %+v", d)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{
		"id": "q-1", "organizationId": org, "patientId": patientA, "appointmentId": visit, "state": "sent", "note": "Placeholder note",
		"currency": "BAM", "totalMinor": 5000.0, "issuedAt": "2026-11-21T09:00:00Z", "validUntil": "2026-12-21T09:00:00Z",
	} {
		if got[k] != want {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
	if lines, _ := got["lines"].([]any); len(lines) != 1 || lines[0].(map[string]any)["serviceId"] != exam {
		t.Errorf("lines = %v", got["lines"])
	}
	for _, tc := range []struct {
		err    error
		body   string
		code   string
		status int
	}{
		{body: `{"patientId":"x","lines":[],"extra":1}`, code: "invalid_request", status: http.StatusBadRequest},
		{err: invoice.ErrInvalid, body: body, code: "invalid_quote", status: http.StatusBadRequest},
		{err: invoice.ErrForbidden, body: body, code: "forbidden", status: http.StatusForbidden},
		{err: invoice.ErrNotFound, body: body, code: "not_found", status: http.StatusNotFound},
		{err: errors.New("boom"), body: body, code: "internal error", status: http.StatusInternalServerError},
	} {
		rec := serve(t, &stub{err: tc.err}, &adminC, http.MethodPost, "/clinic/quotes", tc.body)
		if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.code) {
			t.Errorf("%v: %d %s, want %d %s", tc.err, rec.Code, rec.Body, tc.status, tc.code)
		}
	}
}

// The clinic's list and read, the patient's list and PDF; who may call which.
func TestHandler_QuoteReads(t *testing.T) {
	s := &stub{quote: sentQuote()}
	for _, tc := range []struct {
		caller *invoice.Caller
		target string
		status int
	}{
		{&adminC, "/clinic/quotes", http.StatusOK},
		{&adminC, "/clinic/quotes/q-1", http.StatusOK},
		{&patientC, "/me/quotes", http.StatusOK},
		{&patientC, "/me/quotes/q-1/pdf", http.StatusOK},
		{&adminC, "/clinic/quotes/q-1/pdf", http.StatusOK},
		{nil, "/clinic/quotes/q-1/pdf", http.StatusUnauthorized},
		{&patientC, "/clinic/quotes/q-1/pdf", http.StatusNotFound},
		{nil, "/clinic/quotes", http.StatusUnauthorized},
		{nil, "/me/quotes", http.StatusUnauthorized},
		{&patientC, "/clinic/quotes", http.StatusNotFound},
		{&adminC, "/me/quotes/q-1/pdf", http.StatusNotFound},
		{&invoice.Caller{AccountID: "acct-none"}, "/me/quotes", http.StatusForbidden},
	} {
		rec := serve(t, s, tc.caller, http.MethodGet, tc.target, "")
		if rec.Code != tc.status {
			t.Errorf("GET %s as %v = %d, want %d: %s", tc.target, tc.caller, rec.Code, tc.status, rec.Body)
			continue
		}
		if tc.status == http.StatusOK && rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("GET %s: Cache-Control = %q", tc.target, rec.Header().Get("Cache-Control"))
		}
	}
	rec := serve(t, s, &patientC, http.MethodGet, "/me/quotes/q-1/pdf", "")
	if rec.Header().Get("Content-Type") != "application/pdf" || s.id != "q-1" || !strings.Contains(rec.Header().Get("Content-Disposition"), "quote.pdf") {
		t.Errorf("PDF: %v, id %q", rec.Header(), s.id)
	}
	rec = serve(t, s, &adminC, http.MethodGet, "/clinic/quotes/q-1/pdf", "")
	if rec.Header().Get("Content-Type") != "application/pdf" || s.id != "clinic:q-1" || !strings.Contains(rec.Header().Get("Content-Disposition"), "quote.pdf") {
		t.Errorf("clinic PDF: %v, id %q", rec.Header(), s.id)
	}
	if serve(t, s, &adminC, http.MethodGet, "/clinic/quotes?appointmentId="+visit, ""); s.id != visit {
		t.Errorf("the visit asked = %q, want %q", s.id, visit)
	}
	for _, target := range []string{"/clinic/quotes/q-x", "/clinic/quotes/q-x/pdf"} {
		if rec = serve(t, &stub{err: invoice.ErrNotFound}, &adminC, http.MethodGet, target, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: an unknown quote = %d, want 404", target, rec.Code)
		}
	}
}
