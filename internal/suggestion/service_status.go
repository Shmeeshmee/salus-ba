package suggestion

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// SetStatus records what the practitioner did with a suggestion of the encounter (shown, dismissed or pinned),
// under the encounter's stream's rule (its practitioner with the patient's active consent), audited as their
// update, refusals included, and then publishes the suggestion with its status on the encounter's stream, so
// every open view and the suggestions worker see it. The status is stored before the event is published: the
// stream consumer never changes a stored status (Record). The same status again stores nothing new and is
// published again. ErrInvalid for a status or id the contract does not allow (unaudited), ErrNotFound for an
// encounter the caller may not see or a suggestion that is not the encounter's, ErrConsentRequired after a
// revocation and ErrForbidden to anyone else who sees the encounter.
func (s *Service) SetStatus(ctx context.Context, c appointment.Caller, encounterID, id string, status Status) (Event, error) {
	if !fhir.IsUUID(id) {
		return Event{}, invalid("id")
	}
	if status != StatusShown && status != StatusDismissed && status != StatusPinned {
		return Event{}, invalid("status")
	}
	var e Event
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		disclosed, err := s.Records.Allowed(ctx, c, encounterID)
		if refused = refusal(err); refused != nil {
			return s.Audit.Record(ctx, callerEvent(c, disclosed, auditevent.OutcomeMinorFailure, 0))
		}
		if err != nil {
			return err
		}
		e, _, err = s.Repo.SetStatus(ctx, disclosed, id, status)
		switch {
		case errors.Is(err, ErrNotFound):
			refused = ErrNotFound
			return s.Audit.Record(ctx, callerEvent(c, disclosed, auditevent.OutcomeMinorFailure, 0))
		case err != nil:
			return err
		}
		return s.Audit.Record(ctx, callerEvent(c, disclosed, auditevent.OutcomeSuccess, 1))
	})
	switch {
	case err != nil:
		return Event{}, fmt.Errorf("suggestion: set status: %w", err)
	case refused != nil:
		return Event{}, refused
	}
	if err := s.publish(ctx, e); err != nil {
		return Event{}, err
	}
	return e, nil
}

// publish puts the event on its encounter's stream.
func (s *Service) publish(ctx context.Context, e Event) error {
	b, err := wire.Encode(MessageType, MessageVersion, e)
	if err == nil {
		err = s.Publisher.Publish(ctx, e.EncounterID, b)
	}
	if err != nil {
		return fmt.Errorf("suggestion: publish: %w", err)
	}
	return nil
}

// refusal maps the stream rule's refusal in err to the suggestion's, or nil when err is none.
func refusal(err error) error {
	switch {
	case errors.Is(err, transcript.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, transcript.ErrConsentRequired):
		return ErrConsentRequired
	case errors.Is(err, transcript.ErrForbidden):
		return ErrForbidden
	}
	return nil
}

// callerEvent is the caller's audit event of a status change of count suggestions of Encounter/id ("" when
// none was disclosed), for treatment.
func callerEvent(c appointment.Caller, id string, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         auditevent.ActionUpdate,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     entityName,
		EntityCount:    count,
	}
	if id != "" {
		e.EntityReference = "Encounter/" + id
	}
	return e
}
