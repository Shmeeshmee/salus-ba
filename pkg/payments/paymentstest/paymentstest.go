// Package paymentstest is the in-memory payments.Provider: the fake of the tests (docs/plans/appointments.md, SAL-35)
// and of any binary before the payments epic adds a provider. It takes no money. Safe for concurrent use.
package paymentstest

import (
	"context"
	"strconv"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/payments"
)

// Provider keeps its receipts by key and what is refunded of each. A repeated key returns the first receipt (and a
// repeated refund key does nothing more), as a provider's idempotency keys do; a key repeated with another amount,
// currency or customer is payments.ErrInvalid, as a provider refuses a reused key with other parameters. Decline makes every charge
// payments.ErrDeclined; Err, when set, fails every call.
type Provider struct {
	Err      error
	byKey    map[string]payments.Receipt
	byID     map[string]payments.Receipt
	refunded map[string]int64  // by receipt id
	refunds  map[string]bool   // by refund key
	customer map[string]string // by charge key
	n        int
	mu       sync.Mutex
	Decline  bool
}

// Charge takes c, or returns the receipt of its key.
func (p *Provider) Charge(_ context.Context, c payments.Charge) (payments.Receipt, error) {
	if err := payments.Check(c); err != nil {
		return payments.Receipt{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case p.Err != nil:
		return payments.Receipt{}, p.Err
	case p.Decline:
		return payments.Receipt{}, payments.ErrDeclined
	}
	if r, ok := p.byKey[c.Key]; ok {
		if r.AmountMinor != c.AmountMinor || r.Currency != c.Currency || p.customer[c.Key] != c.Customer {
			return payments.Receipt{}, payments.ErrInvalid
		}
		return r, nil
	}
	if p.byKey == nil {
		p.byKey, p.byID, p.refunded, p.refunds = map[string]payments.Receipt{}, map[string]payments.Receipt{}, map[string]int64{}, map[string]bool{}
		p.customer = map[string]string{}
	}
	p.customer[c.Key] = c.Customer
	p.n++
	r := payments.Receipt{ID: "fake-receipt-" + strconv.Itoa(p.n), Key: c.Key, Currency: c.Currency, AmountMinor: c.AmountMinor}
	p.byKey[c.Key], p.byID[r.ID] = r, r
	return r, nil
}

// Refund gives back r.AmountMinor of its receipt, at most what is left of it, once per key.
func (p *Provider) Refund(_ context.Context, r payments.Refund) error {
	if r.Key == "" || r.AmountMinor <= 0 {
		return payments.ErrInvalid
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Err != nil {
		return p.Err
	}
	receipt, ok := p.byID[r.ReceiptID]
	switch {
	case !ok:
		return payments.ErrUnknownReceipt
	case p.refunds[r.Key]:
		return nil
	case p.refunded[r.ReceiptID]+r.AmountMinor > receipt.AmountMinor:
		return payments.ErrInvalid
	}
	p.refunds[r.Key] = true
	p.refunded[r.ReceiptID] += r.AmountMinor
	return nil
}

// Charged returns the receipts taken, in no order.
func (p *Provider) Charged() []payments.Receipt {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]payments.Receipt, 0, len(p.byKey))
	for _, r := range p.byKey {
		out = append(out, r)
	}
	return out
}

// Refunded returns how much of the receipt id was given back.
func (p *Provider) Refunded(id string) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refunded[id]
}
