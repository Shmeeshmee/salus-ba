package api

import (
	"context"
	"errors"
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
	caller       func(ctx context.Context) (appointment.Caller, bool, error)
	answers      answers
}

// newHistory builds the practitioner's view of the patient's history and the pre-visit summary, and returns their
// routes.
func newHistory(p historyPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := history.NewService(history.Ports{
		Tx: p.tx, Audit: p.audit, Encounters: p.encounters, Compositions: p.documents, Responses: p.answers.responses,
		Questionnaires: p.answers.questionnaires, Clinics: p.clinics, Roles: p.roles, Appointments: p.appointments,
		Allergies: noAllergies{}, Medications: noMedications{},
		Conditions: condition.NewReads(condition.ReadPorts{ // the diagnoses' confirmation (SAL-19's Conditions)
			Repo: condition.NewPostgresReadRepository(p.pool), Encounters: p.encounters, Audit: p.audit, Tx: p.tx,
		}),
	})
	return history.NewHTTPHandler(svc, p.caller, log).Routes
}

// noAllergies and noMedications stand in for the allergies' and medications' services until SAL-41 brings them
// (docs/plans/patient-history.md "Merge order": SAL-41 merges first and this branch wires them then). They fail
// closed: the summary shows both sections as unavailable, never as "none recorded"
// (TestHistory_TheProfileStandInsGoOnceSAL41IsHere keeps them off main).
type noAllergies struct{}

// errProfileNotWired is the stand-ins' failure.
var errProfileNotWired = errors.New("api: the allergies and medications services are not wired yet (SAL-41)")

func (noAllergies) Seen(context.Context, appointment.Caller, string) ([]allergyintolerance.Seen, error) {
	return nil, errProfileNotWired
}

type noMedications struct{}

func (noMedications) Seen(context.Context, appointment.Caller, string) ([]medicationstatement.Seen, error) {
	return nil, errProfileNotWired
}
