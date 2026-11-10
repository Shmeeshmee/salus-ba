package transcript

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
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

// Encounters gives the transcript its encounter (encounter.Service): for the system path that has only the
// encounter's id, scoping the transaction to its clinic; for a caller, with their attendance of its
// appointment. Neither is audited: the transcript audits its own decision.
type Encounters interface {
	Lookup(ctx context.Context, id string) (encounter.Encounter, error)
	Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error)
}

// Consents gives the state of a patient's consent with a clinic (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Sharing tells whether a clinic gives its patients a copy of their transcripts (organization.Service).
type Sharing interface {
	TranscriptCopy(ctx context.Context, organizationID string) (bool, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo       Repository
	Stream     Stream
	Encounters Encounters
	Consents   Consents
	Sharing    Sharing
	Audit      Auditor
	Tx         Transactor
	// Terms tags the stored transcript as it is read (SAL-13); without it, nothing is tagged.
	Terms Tagger
}

// Service holds the transcript's rules (docs/plans/consult-intelligence.md): only final segments are stored,
// once, and only while the patient's consent is active; only the encounter's practitioner, with that consent,
// follows the encounter's stream and reads its transcript, and its patient reads their copy of the transcript
// while the clinic shares it (access.TranscriptCopy). Every write, read and refusal is audited.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// entityName names the table in the audit events of the segments' writes and reads.
const entityName = "transcript_segment"

// Store stores the final segment e of the encounter's transcript for the agent (the stream consumer, which
// has only the event), in the encounter's clinic scope, and audits the write; a partial segment is only pushed
// live and is ignored. It is idempotent: a segment stored already (the stream delivers at least once) is
// neither stored nor audited again. ErrInvalid for an event the contract does not allow, ErrNotFound for an
// encounter the clinic's scope does not show, and ErrConsentRequired, audited, once the patient's care or
// transcription consent with the clinic is no longer active (access.Transcribed, SAL-12), also for a
// consultation that started before the revocation.
func (s *Service) Store(ctx context.Context, agent string, e Event) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if !e.IsFinal {
		return nil
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
		case !access.Transcribed(consent):
			refused = ErrConsentRequired
			return s.Audit.Record(ctx, systemEvent(agent, enc, auditevent.OutcomeMinorFailure, 0))
		}
		created, err := s.Repo.Insert(ctx, e)
		if err != nil || !created {
			return err
		}
		return s.Audit.Record(ctx, systemEvent(agent, enc, auditevent.OutcomeSuccess, 1))
	})
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("transcript: store: %w", err)
	}
	return refused
}

// systemEvent is the agent's audit event of a write of count segments to the encounter's transcript, for
// operations.
func systemEvent(agent string, enc encounter.Encounter, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          auditevent.ActionCreate,
		Outcome:         outcome,
		AgentName:       agent,
		OrganizationID:  enc.OrganizationID,
		EntityReference: "Encounter/" + enc.ID,
		EntityName:      entityName,
		EntityCount:     count,
	}
}
