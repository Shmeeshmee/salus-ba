package subscription_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/subscription"
)

type stubMemberships struct {
	err      error
	invoices []subscription.Invoice
	caller   subscription.Caller
	current  subscription.Subscription
}

func (s *stubMemberships) Current(_ context.Context, c subscription.Caller) (subscription.Subscription, error) {
	s.caller = c
	return s.current, s.err
}

func (s *stubMemberships) Invoices(_ context.Context, c subscription.Caller) ([]subscription.Invoice, error) {
	s.caller = c
	return s.invoices, s.err
}

func serve(t *testing.T, svc *stubMemberships, signedIn bool, callerErr error, log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	h := subscription.NewHTTPHandler(svc, func(context.Context) (subscription.Caller, bool, error) { return admin, signedIn, callerErr },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	return rec
}

func TestHTTPHandler_ServesTheMembershipAndItsInvoices(t *testing.T) {
	due := time.Date(2026, 12, 15, 1, 0, 0, 0, time.UTC)
	svc := &stubMemberships{
		current: subscription.Subscription{
			PlanCode: "basic", Status: subscription.StatusPastDue, PastDueSince: due, CurrentPeriodStart: dec1, CurrentPeriodEnd: jan1,
			Plan: subscription.Plan{Code: "basic", Name: "Basic", Currency: "BAM", PricePerSeatMinor: 10000},
		},
		invoices: []subscription.Invoice{{
			ID: "0d000000-0000-4000-8000-000000000001", Number: 7, Seats: 3, AmountMinor: 30000, Currency: "BAM", Status: subscription.InvoiceOpen,
			PeriodStart: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			IssuedAt: dec1.Add(2 * time.Hour), DueAt: due,
		}},
	}
	rec := serve(t, svc, true, nil, &bytes.Buffer{}, "/clinic/subscription")
	want := `{"currentPeriodStart":"2026-11-30T23:00:00Z","currentPeriodEnd":"2026-12-31T23:00:00Z","pastDueSince":"2026-12-15T01:00:00Z","planCode":"basic","planName":"Basic","currency":"BAM","status":"past_due","pricePerSeatMinor":10000}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET /clinic/subscription = %d %s, want %s uncached", rec.Code, rec.Body, want)
	}
	if svc.caller != admin {
		t.Errorf("caller = %+v, want the signed-in admin", svc.caller)
	}
	rec = serve(t, svc, true, nil, &bytes.Buffer{}, "/clinic/subscription/invoices")
	want = `[{"issuedAt":"2026-12-01T01:00:00Z","dueAt":"2026-12-15T01:00:00Z","id":"0d000000-0000-4000-8000-000000000001","periodStart":"2026-12-01","periodEnd":"2027-01-01","currency":"BAM","status":"open","number":7,"amountMinor":30000,"seats":3}]` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET /clinic/subscription/invoices = %d %s, want %s", rec.Code, rec.Body, want)
	}
	svc.invoices = nil
	if rec = serve(t, svc, true, nil, &bytes.Buffer{}, "/clinic/subscription/invoices"); rec.Body.String() != "[]\n" {
		t.Errorf("no invoices: %s, want an empty array", rec.Body)
	}
}

func TestHTTPHandler_MapsRefusalsToCodes(t *testing.T) {
	for name, tc := range map[string]struct {
		err, callerErr error
		want           string
		code           int
		signedOut      bool
	}{
		"signed out":   {signedOut: true, code: http.StatusUnauthorized, want: `{"error":"unauthenticated"}`},
		"session down": {callerErr: errors.New("secret detail"), code: http.StatusInternalServerError},
		"not admin":    {err: subscription.ErrForbidden, code: http.StatusForbidden, want: `{"error":"forbidden"}`},
		"no clinic":    {err: subscription.ErrNotFound, code: http.StatusNotFound, want: `{"error":"not_found"}`},
		"broken":       {err: errors.New("secret detail"), code: http.StatusInternalServerError},
	} {
		for _, path := range []string{"/clinic/subscription", "/clinic/subscription/invoices"} {
			rec := serve(t, &stubMemberships{err: tc.err}, !tc.signedOut, tc.callerErr, &bytes.Buffer{}, path)
			body := strings.TrimSpace(rec.Body.String())
			if rec.Code != tc.code || (tc.want != "" && body != tc.want) || strings.Contains(body, "secret") {
				t.Errorf("%s %s = %d %s, want %d %s", name, path, rec.Code, body, tc.code, tc.want)
			}
		}
	}
}
