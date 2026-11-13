package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// appointmentRules is the clinic's appointment settings as the appointment service reads them (For): the reminders'
// offsets at booking (SAL-33).
func appointmentRules(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor) *organization.Appointments {
	return organization.NewAppointments(organization.AppointmentsPorts{
		Repo: organization.NewPostgresAppointmentsRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
	})
}

// reminderRoutes are the routes of SAL-33: the patient's notification settings and the admin's SMS usage
// (notification). The api delivers nothing: the notifier does.
func reminderRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, accounts *auth.Service, patients *patient.Service, log *slog.Logger) func(*http.ServeMux) {
	settings := notification.NewService(notification.Ports{
		Deliveries: notification.NewPostgresDeliveryRepository(pool), Devices: notification.NewPostgresRepository(pool), Profiles: patients,
		Audit: audit, Tx: tx, Clock: clock.System{},
	})
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (notification.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return notification.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return notification.NewSettingsHandler(settings, caller, log).Routes
}
