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

// historyPorts are what the practitioner's view of the patient's history is built on (SAL-40).
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
	answers      answers
}

// newHistory builds the practitioner's view of the patient's history and the pre-visit summary, and returns their
// routes.
func newHistory(p historyPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := history.NewService(history.Ports{
		Tx: p.tx, Audit: p.audit, Encounters: p.encounters, Compositions: p.documents, Responses: p.answers.responses,
		Questionnaires: p.answers.questionnaires, Clinics: p.clinics, Roles: p.roles, Appointments: p.appointments,
		Allergies: p.allergies, Medications: p.medications,
		Conditions: condition.NewReads(condition.ReadPorts{ // the diagnoses' confirmation (SAL-19's Conditions)
			Repo: condition.NewPostgresReadRepository(p.pool), Encounters: p.encounters, Audit: p.audit, Tx: p.tx,
		}),
	})
	return history.NewHTTPHandler(svc, p.caller, log).Routes
}
