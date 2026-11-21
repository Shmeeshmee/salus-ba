package fiscaltest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
	"github.com/Shmeeshmee/salus/pkg/fiscal/fiscaltest"
)

var at = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

func receipt(ref string) fiscal.Receipt {
	return fiscal.Receipt{
		IssuedAt: at, Reference: ref, SellerTaxID: "4200000000000", Currency: "BAM", PaymentMethod: fiscal.PaymentCard,
		Number: 1, TotalMinor: 100, Lines: []fiscal.Line{{Name: "Placeholder", Quantity: 1, UnitPriceMinor: 100, TotalMinor: 100}},
	}
}

// The fake answers deterministic test values from the reference, the same answer for a reference reported twice, and
// counts every call: the test that a receipt is never reported twice reads Reported.
func TestReporter_FiscalizesOnceAndCountsEveryCall(t *testing.T) {
	r := &fiscaltest.Reporter{Clock: clock.Fixed{T: at}}
	first, err := r.Report(t.Context(), receipt("r-1"))
	if err != nil || first.ID == "" || first.Code == "" || first.QR == "" || !first.At.Equal(at) {
		t.Fatalf("Report() = %+v, %v; want test values", first, err)
	}
	again, err := r.Report(t.Context(), receipt("r-1"))
	if err != nil || again != first {
		t.Errorf("Report() again = %+v, %v; want the first answer %+v", again, err, first)
	}
	other, _ := r.Report(t.Context(), receipt("r-2"))
	if other.ID == first.ID || other.Code == first.Code {
		t.Errorf("another receipt got %+v, the same as %+v", other, first)
	}
	if got := r.Reported("r-1"); got != 2 {
		t.Errorf("Reported(r-1) = %d, want 2 calls", got)
	}
	if got := r.Fiscalized(); got != 2 {
		t.Errorf("Fiscalized() = %d, want 2 receipts", got)
	}
}

// A reversal names the fiscal id of a receipt the fake fiscalized, and of no other; it is fiscalized once too.
func TestReporter_ReversesAFiscalizedReceiptOnly(t *testing.T) {
	r := &fiscaltest.Reporter{Clock: clock.Fixed{T: at}}
	orig, _ := r.Report(t.Context(), receipt("r-1"))
	rev := receipt("r-2")
	rev.Reverses, rev.TotalMinor, rev.Lines[0].Quantity, rev.Lines[0].TotalMinor = orig.ID, -100, -1, -100
	if got, err := r.Reverse(t.Context(), rev); err != nil || got.ID == "" || got.ID == orig.ID {
		t.Errorf("Reverse() = %+v, %v; want its own fiscal id", got, err)
	}
	again := rev
	again.Reference = "r-5"
	if _, err := r.Reverse(t.Context(), again); !errors.Is(err, fiscal.ErrRejected) {
		t.Errorf("a second reversal of the receipt = %v, want ErrRejected", err)
	}
	rev.Reference, rev.Reverses = "r-3", "unknown"
	if _, err := r.Reverse(t.Context(), rev); !errors.Is(err, fiscal.ErrRejected) {
		t.Errorf("Reverse(unknown) = %v, want ErrRejected", err)
	}
	if _, err := r.Reverse(t.Context(), receipt("r-4")); !errors.Is(err, fiscal.ErrInvalidReceipt) {
		t.Errorf("Reverse() without Reverses = %v, want ErrInvalidReceipt", err)
	}
	if _, err := r.Report(t.Context(), rev); !errors.Is(err, fiscal.ErrInvalidReceipt) {
		t.Errorf("Report() of a reversal = %v, want ErrInvalidReceipt", err)
	}
}

// Err fails every call that passes the checks, as an unreachable authority would, and the call is still counted.
func TestReporter_FailsAsTold(t *testing.T) {
	down := errors.New("synthetic outage")
	r := &fiscaltest.Reporter{Err: down, Clock: clock.Fixed{T: at}}
	if _, err := r.Report(t.Context(), receipt("r-1")); !errors.Is(err, down) {
		t.Errorf("Report() = %v, want the error", err)
	}
	if r.Reported("r-1") != 1 || r.Fiscalized() != 0 {
		t.Errorf("Reported = %d, Fiscalized = %d; want 1 call and none fiscalized", r.Reported("r-1"), r.Fiscalized())
	}
}
