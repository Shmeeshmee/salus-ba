package encounter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
	RecordAll(ctx context.Context, es []auditevent.AuditEvent) error
}

// Appointments gives the encounter its appointment (appointment.Service): unaudited in the caller's scope
// for the lifecycle, with the caller's access facts for a read, and the consultation's outcome.
type Appointments interface {
	Lookup(ctx context.Context, id string) (appointment.Appointment, error)
	Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error)
	RecordConsultationOutcome(ctx context.Context, agent, id string, patientJoined bool) error
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo         Repository
	Audit        Auditor
	Tx           Transactor
	Clock        clock.Clock
	Appointments Appointments
}

// Service holds the encounter rules (SPEC.md §2 "Encounter", "Ending", "Outcome").
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// Start starts the appointment's encounter in progress at at (the practitioner's first join), with the
// appointment's patient, role and clinic, for the agent (a system path: the webhook, a job) in the caller's
// transaction and clinic scope, and returns it. It is idempotent: when the appointment has an encounter
// already, it is given back unchanged, also when finished. Audited as a create when created. ErrNotFound when
// the scope does not show the appointment.
func (s *Service) Start(ctx context.Context, agent, appointmentID string, at time.Time) (Encounter, error) {
	var out Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		a, err := s.Appointments.Lookup(ctx, appointmentID)
		if err != nil {
			return err
		}
		var created bool
		out, created, err = s.Repo.Start(ctx, Encounter{
			AppointmentID: a.ID, OrganizationID: a.OrganizationID, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID,
			Status: StatusInProgress, PeriodStart: at,
		})
		if err != nil || !created {
			return err
		}
		return s.recordSystem(ctx, agent, out.OrganizationID, auditevent.CodeCreate, out.ID)
	})
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return Encounter{}, ErrNotFound
	case err != nil:
		return Encounter{}, fmt.Errorf("encounter: start: %w", err)
	}
	return out, nil
}

// Finish finishes the appointment's encounter in progress at at, for the agent in the caller's transaction
// and clinic scope, audited as an update, and records the appointment's outcome (fulfilled when the patient
// joined, else noshow) through it. It is idempotent: a finished encounter never changes, and an encounter
// never started (the practitioner never joined) is not created; either way nothing is recorded.
func (s *Service) Finish(ctx context.Context, agent, appointmentID string, at time.Time, patientJoined bool) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, changed, err := s.Repo.Finish(ctx, appointmentID, at)
		if err != nil || !changed {
			return err
		}
		if err := s.recordSystem(ctx, agent, e.OrganizationID, auditevent.CodeUpdate, e.ID); err != nil {
			return err
		}
		return s.Appointments.RecordConsultationOutcome(ctx, agent, appointmentID, patientJoined)
	})
	if err != nil {
		return fmt.Errorf("encounter: finish: %w", err)
	}
	return nil
}

// recordSystem audits the agent's interaction on Encounter/id in the clinic org, for operations.
func (s *Service) recordSystem(ctx context.Context, agent, org, interaction, id string) error {
	e := entry(interaction, id, auditevent.OutcomeSuccess, auditevent.CodeHOPERAT)
	e.AgentName, e.OrganizationID = agent, org
	return s.Audit.Record(ctx, e)
}

// record audits interaction by c on Encounter/id ("" when none was disclosed: no id from the request is
// stored). The purpose is the patient's own request for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c appointment.Caller, interaction, id string, outcome auditevent.Outcome) error {
	return s.Audit.Record(ctx, callerEntry(c, interaction, id, outcome))
}

// callerEntry is the audit event of interaction by c on Encounter/id ("" for none), as record writes it.
func callerEntry(c appointment.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	purpose := auditevent.CodePATRQT
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
	}
	e := entry(interaction, id, outcome, purpose)
	e.AgentAccountID, e.OrganizationID = c.AccountID, c.OrganizationID
	return e
}

// entry is the audit event of a FHIR restful interaction on Encounter/id.
func entry(interaction, id string, outcome auditevent.Outcome, purpose string) auditevent.AuditEvent {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	}
	e := auditevent.AuditEvent{
		Type:    auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype: auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose: auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:  action,
		Outcome: outcome,
	}
	if id != "" {
		e.EntityReference = "Encounter/" + id
	}
	return e
}
