package fiscal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

func receipt() fiscal.Receipt {
	return fiscal.Receipt{
		IssuedAt: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC), Reference: "r-1", SellerTaxID: "4200000000000",
		Currency: "BAM", PaymentMethod: fiscal.PaymentCash, Number: 1, TotalMinor: 5000,
		Lines: []fiscal.Line{{Name: "Placeholder exam", Quantity: 1, UnitPriceMinor: 5000, TotalMinor: 5000}},
	}
}

func TestCheck_TakesAWellFormedReceiptOrReversal(t *testing.T) {
	if err := fiscal.Check(receipt()); err != nil {
		t.Errorf("Check(receipt) = %v, want nil", err)
	}
	r := receipt()
	r.Reverses, r.TotalMinor, r.Lines[0].Quantity, r.Lines[0].TotalMinor = "F-1", -5000, -1, -5000
	if err := fiscal.Check(r); err != nil {
		t.Errorf("Check(reversal) = %v, want nil", err)
	}
	// A discount lowers a line's amount, and a reversal's by as much.
	d := receipt()
	d.Lines[0].DiscountMinor, d.Lines[0].TotalMinor, d.TotalMinor = 500, 4500, 4500
	if err := fiscal.Check(d); err != nil {
		t.Errorf("Check(discounted) = %v, want nil", err)
	}
	d.Reverses, d.Lines[0].Quantity, d.Lines[0].TotalMinor, d.TotalMinor = "F-1", -1, -4500, -4500
	if err := fiscal.Check(d); err != nil {
		t.Errorf("Check(discounted reversal) = %v, want nil", err)
	}
}

func TestCheck_RefusesAMalformedReceipt(t *testing.T) {
	for name, change := range map[string]func(*fiscal.Receipt){
		"no reference":       func(r *fiscal.Receipt) { r.Reference = "" },
		"no seller":          func(r *fiscal.Receipt) { r.SellerTaxID = "" },
		"no number":          func(r *fiscal.Receipt) { r.Number = 0 },
		"no time":            func(r *fiscal.Receipt) { r.IssuedAt = time.Time{} },
		"no lines":           func(r *fiscal.Receipt) { r.Lines = nil },
		"a currency":         func(r *fiscal.Receipt) { r.Currency = "KM" },
		"a payment method":   func(r *fiscal.Receipt) { r.PaymentMethod = "cheque" },
		"lines do not total": func(r *fiscal.Receipt) { r.TotalMinor = 1 },
		"a line's quantity":  func(r *fiscal.Receipt) { r.Lines[0].Quantity = 0 },
		"a line's name":      func(r *fiscal.Receipt) { r.Lines[0].Name = "" },
		"a line's total":     func(r *fiscal.Receipt) { r.Lines[0].TotalMinor, r.TotalMinor = 4000, 4000 },
	} {
		r := receipt()
		r.Lines = append([]fiscal.Line(nil), r.Lines...)
		change(&r)
		if err := fiscal.Check(r); !errors.Is(err, fiscal.ErrInvalidReceipt) {
			t.Errorf("%s: Check() = %v, want ErrInvalidReceipt", name, err)
		}
	}
}
