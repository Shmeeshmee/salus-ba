package suggestion

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Encounters gives the system path, which has only the encounter's id, the encounter, scoping the transaction
// to its clinic, unaudited (encounter.Service).
type Encounters interface {
	Lookup(ctx context.Context, id string) (encounter.Encounter, error)
}

// Consents gives the state of a patient's consent with a clinic (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo       Repository
	Encounters Encounters
	Consents   Consents
	Audit      Auditor
	Tx         Transactor
}

// Service holds the suggestions' rules (docs/plans/consult-intelligence.md): every suggestion shown to the
// practitioner is stored with what they did with it, only while the patient's consent is active, every write
// and refusal audited.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// entityName names the table in the audit events of the suggestions' writes.
const entityName = "suggestion"

// Record stores the suggestion event e for the agent (the stream consumer, which has only the event), in the
// encounter's clinic scope: a new suggestion is stored, a later event of it changes its status only, and each
// write is audited. It is idempotent: an event delivered again (the stream delivers at least once) changes
// nothing and is not audited again. ErrInvalid for an event the contract does not allow, ErrNotFound for an
// encounter the clinic's scope does not show, and ErrConsentRequired, audited, once the patient's consent with
// the clinic is no longer active.
func (s *Service) Record(ctx context.Context, agent string, e Event) error {
	if err := e.Validate(); err != nil {
		return err
	}
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		enc, err := s.Encounters.Lookup(ctx, e.EncounterID)
		if err != nil {
			return err
		}
		consent, err := s.Consents.State(ctx, enc.PatientID, enc.OrganizationID)
		switch {
		case err != nil:
			return err
		case !consent.Active:
			refused = ErrConsentRequired
			return s.Audit.Record(ctx, systemEvent(agent, enc, auditevent.CodeCreate, auditevent.OutcomeMinorFailure, 0))
		}
		saved, err := s.Repo.Save(ctx, e)
		switch {
		case err != nil:
			return err
		case saved == SavedCreated:
			return s.Audit.Record(ctx, systemEvent(agent, enc, auditevent.CodeCreate, auditevent.OutcomeSuccess, 1))
		case saved == SavedUpdated:
			return s.Audit.Record(ctx, systemEvent(agent, enc, auditevent.CodeUpdate, auditevent.OutcomeSuccess, 1))
		}
		return nil
	})
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("suggestion: record: %w", err)
	}
	return refused
}

// systemEvent is the agent's audit event of interaction (a create or an update) on count suggestions of the
// encounter, for operations.
func systemEvent(agent string, enc encounter.Encounter, interaction string, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	action := auditevent.ActionCreate
	if interaction == auditevent.CodeUpdate {
		action = auditevent.ActionUpdate
	}
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         outcome,
		AgentName:       agent,
		OrganizationID:  enc.OrganizationID,
		EntityReference: "Encounter/" + enc.ID,
		EntityName:      entityName,
		EntityCount:     count,
	}
}
