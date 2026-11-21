// Package fiscal is the port of fiscalization (docs/plans/clinic-cms.md, ADR 0016): a receipt reported to the tax
// authority, which answers with its identifier, verification code and QR payload. The FBiH Law on Fiscalization of
// Transactions (Sl. novine FBiH 9/26) has receipts reported through the Tax Administration's ESET; that adapter comes
// with package G once the technical specification and test access exist (Q6). fiscaltest has the in-memory fake, the
// only engine until then (internal/config LoadFiscal). It knows nothing of clinics or patients: a receipt carries the
// seller's tax id, amounts and item names, never a patient. Nothing here logs; errors never carry a receipt's content.
package fiscal

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// PaymentMethod is how a receipt was paid.
type PaymentMethod string

// The payment methods a clinic takes.
const (
	PaymentCash     PaymentMethod = "cash"
	PaymentCard     PaymentMethod = "card"
	PaymentTransfer PaymentMethod = "transfer"
)

// Line is one item of a receipt: its name as printed, the quantity (negative on a reversal), the unit price and
// discount in the currency's minor unit, the VAT rate in basis points (nil for exempt) and the line's total.
type Line struct {
	VATRateBP      *int32
	Name           string
	UnitPriceMinor int64
	DiscountMinor  int64
	TotalMinor     int64
	Quantity       int32
}

// Receipt is what is reported: the caller's opaque reference (the receipt's id, so a retried report is recognised),
// the seller's tax id (JIB), the receipt's number in the seller's sequence, when it was issued, how it was paid, its
// currency (ISO 4217), lines and total, and for a reversal the fiscal id of the receipt it reverses.
type Receipt struct {
	IssuedAt      time.Time
	Reference     string
	SellerTaxID   string
	Currency      string
	Reverses      string // the fiscal id of the receipt a reversal reverses; empty on a receipt
	PaymentMethod PaymentMethod
	Lines         []Line
	Number        int64
	TotalMinor    int64
}

// Fiscalized is the authority's answer: its identifier of the receipt, the verification code and the QR payload
// printed on it, and when it was fiscalized.
type Fiscalized struct {
	At   time.Time
	ID   string
	Code string
	QR   string
}

// Reporter reports receipts and their reversals.
type Reporter interface {
	// Report fiscalizes a receipt (Reverses empty).
	Report(ctx context.Context, r Receipt) (Fiscalized, error)
	// Reverse fiscalizes a reversal (Reverses names the receipt it reverses).
	Reverse(ctx context.Context, r Receipt) (Fiscalized, error)
}

var (
	// ErrInvalidReceipt is returned for a receipt Check refuses, before anything is sent.
	ErrInvalidReceipt = errors.New("fiscal: invalid receipt")
	// ErrRejected is returned when the authority refuses the receipt for good: a retry fails the same way.
	ErrRejected = errors.New("fiscal: the authority rejected the receipt")
)

var currency = regexp.MustCompile(`^[A-Z]{3}$`)

// Check refuses, with ErrInvalidReceipt, a receipt without a reference, seller, number, time or lines, with an unknown
// currency or payment method, a line without a name, a line whose quantity is not positive on a receipt and negative
// on a reversal, a discount above the line's amount, a line whose total is not its quantity times its unit price less
// its discount (on a reversal, the discount given back), or lines that do not add up to the total.
func Check(r Receipt) error {
	if r.Reference == "" || r.SellerTaxID == "" || r.Number <= 0 || r.IssuedAt.IsZero() || len(r.Lines) == 0 ||
		!currency.MatchString(r.Currency) {
		return ErrInvalidReceipt
	}
	switch r.PaymentMethod {
	case PaymentCash, PaymentCard, PaymentTransfer:
	default:
		return ErrInvalidReceipt
	}
	var total int64
	for _, l := range r.Lines {
		if !validLine(l, r.Reverses != "") {
			return ErrInvalidReceipt
		}
		total += l.TotalMinor
	}
	if total != r.TotalMinor {
		return ErrInvalidReceipt
	}
	return nil
}

// validLine reports whether a line is named, of a positive quantity on a receipt and a negative one on a reversal,
// with a discount of at most its amount, and totals to its quantity times its unit price less the discount.
func validLine(l Line, reversal bool) bool {
	qty := int64(l.Quantity)
	if reversal {
		qty = -qty
	}
	return l.Name != "" && qty > 0 && l.UnitPriceMinor >= 0 && l.DiscountMinor >= 0 &&
		l.DiscountMinor <= qty*l.UnitPriceMinor && l.TotalMinor == lineTotal(l)
}

// lineTotal is a line's quantity times its unit price, less its discount on a sale and plus it on a reversal.
func lineTotal(l Line) int64 {
	amount := int64(l.Quantity) * l.UnitPriceMinor
	if l.Quantity < 0 {
		return amount + l.DiscountMinor
	}
	return amount - l.DiscountMinor
}
