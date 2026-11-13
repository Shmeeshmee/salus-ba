// Package payments is the port of charging a patient for a booking (docs/plans/appointments.md, ADR 0015): a Charge
// with the caller's idempotency key, and a Refund of a Receipt. paymentstest has the in-memory fake; no provider's
// adapter exists before the payments epic (SAL-23), and none is wired: no price exists in the model yet, so nothing is
// charged. A provider is a processor and needs its DPA and an entry in docs/privacy/processors.md first. Nothing here
// logs; errors never carry an amount, a customer or a key.
package payments

import (
	"context"
	"errors"
)

// Charge is a payment to take: the idempotency key (a repeat with the same key takes nothing more and returns the
// first receipt), the amount in the currency's minor unit, the ISO 4217 currency, and the provider's customer.
type Charge struct {
	Key         string
	Currency    string
	Customer    string
	AmountMinor int64
}

// Receipt is a charge taken.
type Receipt struct {
	ID          string
	Key         string
	Currency    string
	AmountMinor int64
}

// Refund gives back up to the receipt's amount, once per key.
type Refund struct {
	Key         string
	ReceiptID   string
	AmountMinor int64
}

// Provider takes charges and refunds them.
type Provider interface {
	Charge(ctx context.Context, c Charge) (Receipt, error)
	Refund(ctx context.Context, r Refund) error
}

var (
	// ErrInvalid is returned for a charge or refund without a key, a positive amount or a currency, or a refund of
	// more than is left of its receipt.
	ErrInvalid = errors.New("payments: invalid charge or refund")
	// ErrDeclined is returned when the provider declines the charge: nothing was taken.
	ErrDeclined = errors.New("payments: declined")
	// ErrUnknownReceipt is returned for a refund of a receipt the provider does not know.
	ErrUnknownReceipt = errors.New("payments: unknown receipt")
)

// Check is nil for a charge a provider may take: a key, a positive amount, a currency of three capital letters and a
// customer.
func Check(c Charge) error {
	if c.Key == "" || c.AmountMinor <= 0 || !currency(c.Currency) || c.Customer == "" {
		return ErrInvalid
	}
	return nil
}

func currency(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
