package paymentstest_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/payments"
	"github.com/Shmeeshmee/salus/pkg/payments/paymentstest"
)

var _ payments.Provider = (*paymentstest.Provider)(nil)

var charge = payments.Charge{Key: "offer-1", Currency: "BAM", Customer: "c1", AmountMinor: 5000}

func TestProvider_ChargesOncePerKeyAndRefundsAtMostTheAmount(t *testing.T) {
	t.Parallel()
	p := &paymentstest.Provider{}
	r, err := p.Charge(t.Context(), charge)
	if err != nil || r.AmountMinor != 5000 || r.Key != "offer-1" || r.ID == "" {
		t.Fatalf("Charge() = %+v, %v", r, err)
	}
	if again, err := p.Charge(t.Context(), charge); err != nil || again != r || len(p.Charged()) != 1 {
		t.Errorf("a repeat = %+v, %v with %d receipts; want the first receipt only", again, err, len(p.Charged()))
	}
	if err := p.Refund(t.Context(), payments.Refund{Key: "rf-1", ReceiptID: r.ID, AmountMinor: 3000}); err != nil {
		t.Fatal(err)
	}
	if err := p.Refund(t.Context(), payments.Refund{Key: "rf-1", ReceiptID: r.ID, AmountMinor: 3000}); err != nil || p.Refunded(r.ID) != 3000 {
		t.Errorf("a repeated refund = %v, refunded %d; want nothing more", err, p.Refunded(r.ID))
	}
	if err := p.Refund(t.Context(), payments.Refund{Key: "rf-2", ReceiptID: r.ID, AmountMinor: 2001}); !errors.Is(err, payments.ErrInvalid) {
		t.Errorf("a refund past the amount = %v", err)
	}
	if err := p.Refund(t.Context(), payments.Refund{Key: "rf-3", ReceiptID: "nope", AmountMinor: 1}); !errors.Is(err, payments.ErrUnknownReceipt) {
		t.Errorf("an unknown receipt = %v", err)
	}
	if err := p.Refund(t.Context(), payments.Refund{ReceiptID: r.ID, AmountMinor: 1}); !errors.Is(err, payments.ErrInvalid) {
		t.Errorf("a refund without a key = %v", err)
	}
}

func TestProvider_RefusesInvalidChargesAndDeclinesOrFailsAsTold(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]payments.Charge{
		"no key": {Currency: "BAM", Customer: "c", AmountMinor: 1}, "zero": {Key: "k", Currency: "BAM", Customer: "c"},
		"currency": {Key: "k", Currency: "bam", Customer: "c", AmountMinor: 1}, "no customer": {Key: "k", Currency: "BAM", AmountMinor: 1},
		"long currency": {Key: "k", Currency: "BAMX", Customer: "c", AmountMinor: 1},
	} {
		if _, err := (&paymentstest.Provider{}).Charge(t.Context(), c); !errors.Is(err, payments.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (&paymentstest.Provider{Decline: true}).Charge(t.Context(), charge); !errors.Is(err, payments.ErrDeclined) {
		t.Errorf("declined: %v", err)
	}
	down := errors.New("down")
	if _, err := (&paymentstest.Provider{Err: down}).Charge(t.Context(), charge); !errors.Is(err, down) {
		t.Errorf("down: %v", err)
	}
	if err := (&paymentstest.Provider{Err: down}).Refund(t.Context(), payments.Refund{Key: "k", ReceiptID: "r", AmountMinor: 1}); !errors.Is(err, down) {
		t.Errorf("refund down: %v", err)
	}
}

func TestProvider_IsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	p := &paymentstest.Provider{}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := p.Charge(t.Context(), charge); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := len(p.Charged()); n != 1 {
		t.Errorf("%d receipts for one key, want 1", n)
	}
}
