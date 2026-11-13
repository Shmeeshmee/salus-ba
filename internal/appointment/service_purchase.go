package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/payments"
)

// Prices gives the price of booking the slot b in the currency's minor unit, and the ISO 4217 currency; zero for a
// slot that costs nothing. The payments epic (SAL-23) implements it; until then no Service has one.
type Prices interface {
	Price(ctx context.Context, b slot.Bookable) (amountMinor int64, currency string, err error)
}

// purchase pays for the booking of the slot b by c under the idempotency key, before the booking's transaction (a
// direct booking's, a waitlist acceptance's: docs/plans/appointments.md "Payments", ADR 0015), and returns the
// receipt; none (and no call to the provider) without a price, which is every booking before SAL-23, so nothing
// changes for today's booking. payments.ErrDeclined passes through.
func (s *Service) purchase(ctx context.Context, c Caller, b slot.Bookable, key string) (payments.Receipt, error) {
	if s.Prices == nil {
		return payments.Receipt{}, nil
	}
	amount, currency, err := s.Prices.Price(ctx, b)
	switch {
	case err != nil:
		return payments.Receipt{}, fmt.Errorf("price: %w", err)
	case amount == 0:
		return payments.Receipt{}, nil
	case s.Payments == nil:
		return payments.Receipt{}, errors.New("a price without a payment provider")
	}
	r, err := s.Payments.Charge(ctx, payments.Charge{Key: key, AmountMinor: amount, Currency: currency, Customer: c.PatientID})
	if err != nil && !errors.Is(err, payments.ErrDeclined) {
		return payments.Receipt{}, fmt.Errorf("charge: %w", err)
	}
	return r, err
}

// refund gives back the whole of a purchase whose booking then failed; nothing for none.
func (s *Service) refund(ctx context.Context, r payments.Receipt) error {
	if r.ID == "" {
		return nil
	}
	return s.Payments.Refund(ctx, payments.Refund{Key: "refund:" + r.Key, ReceiptID: r.ID, AmountMinor: r.AmountMinor})
}

// settle decides what a patient's cancellation (and, with SAL-34, a reschedule) of a owes or is owed: the seam of
// purchase protection (SAL-37). Nothing is ever paid before SAL-23, so nothing is refunded.
func (s *Service) settle(_ context.Context, _ Appointment) error { return nil }
