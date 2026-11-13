package api

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// waitlistPorts sets the waitlist's ports of the appointment service p (SAL-35): its store, the offered slots, the
// offers' expiries, and notification's deliveries for their notices (Deliver in the offer's transaction: the patient's
// channels, quiet hours and SMS) and for the quiet hours an offer passes over. The api makes no offer itself (the
// worker's jobs do), but its appointment service is complete.
func waitlistPorts(p *appointment.Ports, pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, jobs *river.Client[pgx.Tx],
	clinics *organization.Service, slots *slot.Service,
) {
	offers := notification.NewService(notification.Ports{
		Clinics: clinics, Deliveries: notification.NewPostgresDeliveryRepository(pool), DeliveryJobs: notification.NewRiverDeliveryJobs(jobs),
		Devices: notification.NewPostgresRepository(pool), Rules: appointmentRules(pool, audit, tx), Audit: audit, Tx: tx, Clock: clock.System{},
	})
	p.Waitlist, p.Offered, p.OfferJobs = appointment.NewPostgresWaitlistRepository(pool), slots, appointment.NewRiverOfferJobs(jobs)
	p.Notices, p.Quiet = offers, offers
}
