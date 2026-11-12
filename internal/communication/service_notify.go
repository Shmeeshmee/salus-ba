package communication

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// MessageSentArgs is the chat_message_sent job (docs/plans/chat.md, SAL-29): enqueued in a new message's
// transaction, worked by the notifier, which tells the conversation's other participants (push, the delayed e-mail)
// and the clinic's webhooks. Opaque ids and the message's number only, never its text or a name.
type MessageSentArgs struct {
	ConversationID      string `json:"conversation_id"`
	OrganizationID      string `json:"organization_id"`
	MessageID           string `json:"message_id"`
	SenderParticipantID string `json:"sender_participant_id"`
	Seq                 int64  `json:"seq"`
}

// Kind names the job in the river_job table.
func (MessageSentArgs) Kind() string { return "chat_message_sent" }

// Jobs enqueues the chat's background jobs in the transaction in the context (River's InsertTx), so a job exists
// exactly when the message that causes it commits.
type Jobs interface {
	EnqueueMessageSent(ctx context.Context, a MessageSentArgs) error
}

// NoticeConversation returns the conversation id of the clinic org with the participants who may read it now and
// their read positions, and its last message's number, for the job agent that tells them of a new message: in the
// clinic's scope, so nothing after the patient's revocation (ErrNotFound), audited as a read of the patient's
// participants with the job as agent. Who may read it follows SAL-28's rules (noticed).
func (s *Service) NoticeConversation(ctx context.Context, agent, org, id string) (Conversation, error) {
	if !validUUID(id) {
		return Conversation{}, ErrNotFound
	}
	var out Conversation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.clinicConversation(ctx, org, id); err != nil {
			return err
		}
		if out.Participants, err = s.noticed(ctx, out); err != nil {
			return err
		}
		return s.recordJob(ctx, agent, org, "Patient/"+out.PatientID, "conversation_participant", int64(len(out.Participants)))
	})
	if err != nil {
		return Conversation{}, wrap("notice conversation", err)
	}
	return out, nil
}

// NoticeMessage returns the message messageID of the conversation id of the clinic org, for the job agent that
// sends it to the clinic's own webhook (with its clinical content turned on): in the clinic's scope, so nothing after
// the patient's revocation (ErrNotFound), audited as a read of the message with the job as agent.
func (s *Service) NoticeMessage(ctx context.Context, agent, org, id, messageID string) (Message, error) {
	if !validUUID(id) || !validUUID(messageID) {
		return Message{}, ErrNotFound
	}
	var out Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.clinicConversation(ctx, org, id); err != nil {
			return err
		}
		if err := s.Repo.SetConversation(ctx, id); err != nil {
			return err
		}
		var err error
		if out, err = s.Repo.Message(ctx, id, messageID); err != nil {
			return err
		}
		return s.recordJob(ctx, agent, org, "Communication/"+out.ID, "", 0)
	})
	if err != nil {
		return Message{}, wrap("notice message", err)
	}
	return out, nil
}

// noticed is the participants of conv who may read it now (SAL-28): active ones only, never one waiting for the
// patient's consent; a doctor of another clinic only while the patient's chat consent for them is active (the care
// consent with the clinic is clinicConversation's); never the patient in a doctor-only side thread.
func (s *Service) noticed(ctx context.Context, conv Conversation) ([]Participant, error) {
	parts, err := s.Repo.Participants(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	out := parts[:0]
	for _, p := range parts {
		switch {
		case p.Status != StatusActive, conv.StaffOnly && p.Role == RolePatient:
			continue
		case p.Role == RolePractitioner && p.OrganizationID != "" && p.OrganizationID != conv.OrganizationID:
			granted, err := s.ChatConsents.ChatGranted(ctx, root(conv), p.PractitionerRoleID)
			if err != nil {
				return nil, err
			}
			if !granted {
				continue
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// clinicConversation scopes the transaction to the clinic org and reads the conversation id there; ErrNotFound
// when the scope does not show it, it is another clinic's, or the patient's care consent with the clinic is not
// active (as the clinic's row-level security answers; checked here too, not on it alone).
func (s *Service) clinicConversation(ctx context.Context, org, id string) (Conversation, error) {
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return Conversation{}, err
	}
	conv, err := s.Repo.Conversation(ctx, id, false)
	switch {
	case err != nil:
		return Conversation{}, err
	case conv.OrganizationID != org:
		return Conversation{}, ErrNotFound
	}
	state, err := s.Consents.State(ctx, conv.PatientID, org)
	switch {
	case err != nil:
		return Conversation{}, err
	case !state.Active:
		return Conversation{}, ErrNotFound
	}
	return conv, nil
}

// recordJob audits a read by the job agent for the clinic org: ref the resource, name and count the rows.
func (s *Service) recordJob(ctx context.Context, agent, org, ref, name string, count int64) error {
	err := s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          auditevent.ActionRead,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		OrganizationID:  org,
		EntityReference: ref,
		EntityName:      name,
		EntityCount:     count,
	})
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	return nil
}

// errNoJobs is returned by a send on a Service built without its Jobs port (a composition error).
var errNoJobs = errors.New("communication: no Jobs port")
