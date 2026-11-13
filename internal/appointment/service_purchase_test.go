package appointment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/payments"
	"github.com/Shmeeshmee/salus/pkg/payments/paymentstest"
)

// fakePrices prices every slot at amount BAM.
type fakePrices struct {
	err    error
	amount int64
}

func (p fakePrices) Price(context.Context, slot.Bookable) (int64, string, error) {
	return p.amount, "BAM", p.err
}

func pricedFixture(amount int64) (*fixture, *paymentstest.Provider) {
	f := newFixture()
	provider := &paymentstest.Provider{}
	f.svc = appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now}, Consents: f.consents, Slots: f.slots,
		Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs, Stats: f.stats,
		Prices: fakePrices{amount: amount}, Payments: provider,
	})
	return f, provider
}

// Without a price nothing is charged: today's booking (review, finding 4).
func TestPurchase_ChargesNothingWithoutAPrice(t *testing.T) {
	f, provider := pricedFixture(0)
	if _, err := f.svc.Book(t.Context(), me, slotFree); err != nil {
		t.Fatal(err)
	}
	if got := provider.Charged(); len(got) != 0 {
		t.Errorf("charged %+v, want nothing", got)
	}
}

// A priced booking is charged before its transaction under the patient's and the slot's key.
func TestPurchase_ChargesAPricedBookingBeforeItsTransaction(t *testing.T) {
	f, provider := pricedFixture(5000)
	if _, err := f.svc.Book(t.Context(), me, slotFree); err != nil {
		t.Fatal(err)
	}
	got := provider.Charged()
	if len(got) != 1 || got[0].AmountMinor != 5000 || got[0].Key != "booking:"+patientA+":"+slotFree || provider.Refunded(got[0].ID) != 0 {
		t.Errorf("charged %+v, want 5000 under the booking's key, not refunded", got)
	}
}

// A booking that fails after the charge is refunded in full; a declined payment books nothing, audited.
func TestPurchase_RefundsWhenTheBookingThenFailsAndRefusesADeclinedPayment(t *testing.T) {
	f, provider := pricedFixture(5000)
	delete(f.consents.state, patientA+clinicA) // no active consent: refused inside the transaction
	if _, err := f.svc.Book(t.Context(), me, slotFree); !errors.Is(err, appointment.ErrConsentRequired) {
		t.Fatalf("Book() = %v, want ErrConsentRequired", err)
	}
	got := provider.Charged()
	if len(got) != 1 || provider.Refunded(got[0].ID) != 5000 {
		t.Errorf("charged %+v, refunded %d; want the charge refunded in full", got, provider.Refunded(got[0].ID))
	}

	f, provider = pricedFixture(5000)
	provider.Decline = true
	if _, err := f.svc.Book(t.Context(), me, slotFree); !errors.Is(err, appointment.ErrPaymentDeclined) || len(f.repo.appts) != 0 {
		t.Errorf("Book() = %v with %d appointments, want ErrPaymentDeclined and none", err, len(f.repo.appts))
	}
	checkEvents(t, "declined", f.audit, event(me, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure))

	f, provider = pricedFixture(5000)
	provider.Err = errors.New("the provider is down")
	if _, err := f.svc.Book(t.Context(), me, slotFree); err == nil || errors.Is(err, appointment.ErrPaymentDeclined) || len(f.repo.appts) != 0 {
		t.Errorf("Book() with the provider down = %v, want a failure and no appointment", err)
	}
}

// cancellingProvider cancels the request once it has charged, and records whether the refund's context was done.
type cancellingProvider struct {
	*paymentstest.Provider
	cancel     context.CancelFunc
	refundDone bool
}

func (p *cancellingProvider) Charge(ctx context.Context, c payments.Charge) (payments.Receipt, error) {
	r, err := p.Provider.Charge(ctx, c)
	p.cancel()
	return r, err
}

func (p *cancellingProvider) Refund(ctx context.Context, r payments.Refund) error {
	p.refundDone = ctx.Err() != nil
	return p.Provider.Refund(ctx, r)
}

// A request cancelled after the charge still refunds a booking that then fails (review of the fixes, finding 2).
func TestPurchase_RefundsEvenWhenTheRequestIsCancelled(t *testing.T) {
	f, _ := pricedFixture(5000)
	ctx, cancel := context.WithCancel(t.Context())
	provider := &cancellingProvider{Provider: &paymentstest.Provider{}, cancel: cancel}
	f.svc = appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now}, Consents: f.consents, Slots: f.slots,
		Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs, Stats: f.stats,
		Prices: fakePrices{amount: 5000}, Payments: provider,
	})
	delete(f.consents.state, patientA+clinicA)
	if _, err := f.svc.Book(ctx, me, slotFree); !errors.Is(err, appointment.ErrConsentRequired) {
		t.Fatalf("Book() = %v, want ErrConsentRequired", err)
	}
	got := provider.Charged()
	if len(got) != 1 || provider.Refunded(got[0].ID) != 5000 || provider.refundDone {
		t.Errorf("charged %+v, refunded with a done context %t; want the charge refunded on a live context", got, provider.refundDone)
	}
}
