package worker

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The worker's appointment service, which makes the offers (waitlist_slot_freed, waitlist_offer_expire), gets their
// deliveries, the quiet hours and the patients' accounts (SAL-35 at SAL-33's merge).
func TestNewRevocations_GiveTheOffersTheirDeliveriesAndQuietHours(t *testing.T) {
	t.Parallel()
	p := newRevocations(nil, nil, clock.System{}).Ports
	if p.Waitlist == nil || p.Offered == nil || p.OfferJobs == nil || p.Notices == nil || p.Quiet == nil || p.Rules == nil || p.Patients == nil ||
		p.Roles == nil || p.Clinics == nil {
		t.Errorf("ports %+v, want the offers' all set", p)
	}
}
