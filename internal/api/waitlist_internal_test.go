package api

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// The api's appointment service gets every port of the waitlist's offers, quiet hours included (SAL-35 at SAL-33's
// merge): New passes its Ports through waitlistPorts.
func TestWaitlistPorts_GiveTheOffersTheirDeliveriesAndQuietHours(t *testing.T) {
	t.Parallel()
	var p appointment.Ports
	waitlistPorts(&p, nil, &auditevent.Service{}, &postgres.Transactor{}, nil, &organization.Service{}, &slot.Service{})
	if p.Waitlist == nil || p.Offered == nil || p.OfferJobs == nil || p.Notices == nil || p.Reach == nil {
		t.Errorf("ports %+v, want the waitlist's all set", p)
	}
}
