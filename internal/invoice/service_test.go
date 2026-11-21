package invoice_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal/fiscaltest"
)

// The price list's service is the Prices port, wired in as it is (no adapter).
var _ invoice.Prices = (*chargeitemdefinition.Service)(nil)

// Step 0 is the contract only: every method refuses until package C builds it (docs/plans/clinic-cms.md).
func TestService_RefusesEveryMethodUntilPackageC(t *testing.T) {
	clk := clock.Fixed{T: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)}
	svc := invoice.NewService(nil, chargeitemdefinition.NewService(nil, clk), &fiscaltest.Reporter{Clock: clk}, clk)
	ctx, c := t.Context(), invoice.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	_, issue := svc.Issue(ctx, c, invoice.Draft{Kind: invoice.KindReceipt, PaymentMethod: invoice.PaymentCash})
	_, read := svc.Read(ctx, c, "i")
	_, list := svc.List(ctx, c, invoice.Period{From: clk.T, To: clk.T.Add(24 * time.Hour)})
	_, reverse := svc.Reverse(ctx, c, "i", invoice.PaymentCash)
	_, pdf := svc.PDF(ctx, c, "i")
	_, mine := svc.Mine(ctx, invoice.Caller{AccountID: "p", PatientID: "p"})
	fiscalize := svc.Fiscalize(ctx, "o", "i")
	for name, err := range map[string]error{"Issue": issue, "Read": read, "List": list, "Reverse": reverse, "PDF": pdf, "Mine": mine, "Fiscalize": fiscalize} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s() = %v, want errors.ErrUnsupported", name, err)
		}
	}
}

// The fiscal job carries opaque ids only.
func TestFiscalizeArgs_IsTheInvoiceFiscalizeJob(t *testing.T) {
	if got := (invoice.FiscalizeArgs{InvoiceID: "i", OrganizationID: "o"}).Kind(); got != "invoice_fiscalize" {
		t.Errorf("Kind() = %q, want invoice_fiscalize", got)
	}
}
