package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/careplan"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// carePlanPorts are what the treatment plans' routes are built from.
type carePlanPorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	appointments *appointment.Service
	care         *appointment.Facts
	roles        *practitionerrole.Service
	consents     *consent.Service
	prices       *chargeitemdefinition.Service
	accounts     *auth.Service
	patients     *patient.Service
}

// carePlanRoutes are the treatment plans' (docs/clinic-cms/WAVE6.md U6, package TP1): laid out, booked and cancelled
// by the clinic's admin and the practitioners who have or had a visit with the patient, read by their patient once
// booked. The visits are the appointments'.
func carePlanRoutes(p carePlanPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := careplan.NewService(careplan.Ports{
		Repo: careplan.NewPostgresRepository(p.pool), Tx: p.tx, Audit: p.audit, Consents: p.consents, Prices: p.prices,
		Visits: p.appointments, Care: p.care, Roles: p.roles, Clock: clock.System{},
	})
	return careplan.NewHTTPHandler(svc, carePlanCaller(p.accounts, p.patients), log).Routes
}

// carePlanCaller is the plans' view of the signed-in session: for staff, whether the account is the clinic's admin,
// a practitioner or its receptionist (refused: no clinical content); for a patient, their profile's id.
func carePlanCaller(accounts *auth.Service, patients *patient.Service) func(ctx context.Context) (careplan.Caller, bool, error) {
	caller := deskCaller(accounts, patients)
	return func(ctx context.Context) (careplan.Caller, bool, error) {
		c, ok, err := caller(ctx)
		if c.OrganizationID == "" {
			return careplan.Caller{Patient: careplan.Patient{AccountID: c.AccountID, PatientID: c.PatientID}}, ok, err
		}
		return careplan.Caller{Staff: careplan.Staff{
			AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID,
			Admin: c.Admin, Practitioner: c.PractitionerID != "", Reception: c.Reception,
		}}, ok, err
	}
}
