package api

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// practiceSources are the services the online practice report reads.
type practiceSources struct {
	clinics      *organization.Service
	schedules    *schedule.Service
	appointments *appointment.Service
}

// practicePorts are the online practice report's sources (clinic-cms E): the clinic for its zone, its schedules, its
// appointments and its receipts. The receipts are read through an invoice.Service of their own that only lists.
func practicePorts(p clinicstats.Ports, pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, src practiceSources) clinicstats.Ports {
	p.Clock, p.Clinics, p.Schedules, p.Visits = clock.System{}, src.clinics, src.schedules, src.appointments
	p.Receipts = invoice.NewService(invoice.Ports{Repo: invoice.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}})
	return p
}
