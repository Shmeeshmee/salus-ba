package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// historyPorts are what the history is built on: the patient's timeline (SAL-39) and the practitioner's view of
// the patient's history with the pre-visit summary (SAL-40). timeline holds the questionnaires' sources.
type historyPorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	encounters   *encounter.Service
	documents    *composition.Service
	clinics      *organization.Service
	roles        *practitionerrole.Service
	appointments *appointment.Service
	allergies    *allergyintolerance.Service
	medications  *medicationstatement.Service
	caller       func(ctx context.Context) (appointment.Caller, bool, error)
	timeline     history.Ports
}

// newHistory builds the history, the patient's timeline and the practitioner's view with the summary, and returns
// their routes.
func newHistory(p historyPorts, log *slog.Logger) []func(*http.ServeMux) {
	ports := p.timeline
	ports.Visits, ports.Documents, ports.Clinics, ports.Roles, ports.Audit, ports.Tx = p.encounters, p.documents, p.clinics, p.roles, p.audit, p.tx
	ports.Encounters, ports.Compositions, ports.ScopeClinics, ports.RoleSpecialties = p.encounters, p.documents, p.clinics, p.roles
	ports.Appointments, ports.Allergies, ports.Medications = p.appointments, p.allergies, p.medications
	ports.Conditions = condition.NewReads(condition.ReadPorts{ // the diagnoses' confirmation (SAL-19's Conditions)
		Repo: condition.NewPostgresReadRepository(p.pool), Encounters: p.encounters, Audit: p.audit, Tx: p.tx,
	})
	svc := history.NewService(ports)
	return []func(*http.ServeMux){history.NewHTTPHandler(svc, p.caller, log).Routes, history.NewPractitionerHandler(svc, p.caller, log).Routes}
}
