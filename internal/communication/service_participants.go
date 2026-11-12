package communication

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// entityParticipant is the audit's entity name of an add, an invitation, a removal or a decision.
const entityParticipant = "conversation_participant"

// AddParticipant adds a practitioner role to the conversation id as its treating practitioner asks
// (docs/plans/chat.md, SAL-28) and returns the participant: a role of the clinic, active, at once; a role of another
// clinic in the public directory, pending until the patient consents (DecideInvitation), before which that doctor
// sees nothing. They read the whole history, or the messages after in.FromSeq (at most the conversation's last). A
// system message names the role, its clinic and the reason, in the conversation's order, published after the commit;
// the add is audited. ErrInvalid for a request the contract refuses or a role that may not be added,
// ErrAlreadyParticipant for one who takes part or is invited already, ErrForbidden for anyone but the treating
// practitioner or for a side thread; access as enter's for a write.
func (s *Service) AddParticipant(ctx context.Context, c Caller, id string, in AddParticipant) (Participant, error) {
	if err := in.validate(); err != nil {
		return Participant{}, err
	}
	invited := c.OrganizationID != "" && in.OrganizationID != c.OrganizationID // a patient is refused in the transaction
	if invited {
		if err := s.listed(ctx, in); err != nil {
			return Participant{}, err
		}
	}
	var out Participant
	var msg Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.manage(ctx, c, id)
		if err != nil {
			return err
		}
		if !invited {
			if err := s.member(ctx, e.conv.OrganizationID, in.PractitionerRoleID); err != nil {
				return err
			}
		}
		if slices.ContainsFunc(e.conv.Participants, func(p Participant) bool { return p.PractitionerRoleID == in.PractitionerRoleID }) {
			return ErrAlreadyParticipant
		}
		p := Participant{
			ConversationID: id, PractitionerRoleID: in.PractitionerRoleID, OrganizationID: in.OrganizationID, AddedBy: c.AccountID,
			AddedAt: s.Clock.Now().UTC(), Role: RolePractitioner, Status: StatusActive, Reason: in.Reason,
		}
		if in.FromSeq != nil {
			if *in.FromSeq > e.conv.LastSeq {
				return invalid("from_seq")
			}
			p.HistoryFromSeq = *in.FromSeq
		}
		code := SystemParticipantAdded
		if invited {
			p.Status, code = StatusPendingConsent, SystemParticipantInvited
		}
		if out, err = s.Repo.InsertParticipant(ctx, p); err != nil {
			return err
		}
		args := map[string]string{"practitioner_role_id": p.PractitionerRoleID, "organization_id": p.OrganizationID, "reason": string(p.Reason)}
		if msg, err = s.system(ctx, e.conv, code, args); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate, "Patient/"+e.conv.PatientID, entityParticipant, 1)
	})
	if err != nil {
		return Participant{}, wrap("add participant", s.refusal(ctx, c, id, err))
	}
	s.publish(ctx, MessageType, msg)
	return out, nil
}

// validate checks the request's ids, reason and history.
func (in AddParticipant) validate() error {
	switch {
	case !validUUID(in.PractitionerRoleID):
		return invalid("practitioner_role_id")
	case !validUUID(in.OrganizationID):
		return invalid("organization_id")
	case !in.Reason.valid():
		return invalid("reason")
	case in.History == HistoryFull && in.FromSeq == nil:
	case in.History == HistoryFrom && in.FromSeq != nil && *in.FromSeq >= 0:
	case in.History == HistoryFrom || in.History == HistoryFull:
		return invalid("from_seq")
	default:
		return invalid("history")
	}
	return nil
}

// listed checks that the role of another clinic is in the public directory (an active role of a listed clinic with
// an active practitioner), read in its own transaction before the conversation's: it is public, and the directory's
// scope is not the chat's.
func (s *Service) listed(ctx context.Context, in AddParticipant) error {
	roles, err := s.Directory.PublicByOrganization(ctx, in.OrganizationID)
	if err != nil {
		return wrap("add participant", err)
	}
	if !slices.ContainsFunc(roles, func(r practitionerrole.PractitionerRole) bool {
		return r.ID == in.PractitionerRoleID && r.OrganizationID == in.OrganizationID && r.Active
	}) {
		return invalid("practitioner_role_id")
	}
	return nil
}

// member checks that the role is an active role of the clinic (the conversation's, whose scope the transaction has).
func (s *Service) member(ctx context.Context, org, roleID string) error {
	role, err := s.Directory.Lookup(ctx, org, roleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound) || (err == nil && !role.Active):
		return invalid("practitioner_role_id")
	case err != nil:
		return err
	}
	return nil
}

// manage enters the conversation id, locked, for its treating practitioner (access.ManagesChat): never a side thread,
// whose doctors are those it was opened with.
func (s *Service) manage(ctx context.Context, c Caller, id string) (entered, error) {
	e, err := s.enter(ctx, c, id, true, true)
	switch {
	case err != nil:
		return entered{}, err
	case e.conv.StaffOnly || !access.ManagesChat(e.who.viewer, e.facts, e.conv.PractitionerRoleID):
		return entered{}, ErrForbidden
	}
	return e, nil
}

// RemoveParticipant removes the practitioner participantID from the conversation id, as its treating practitioner
// asks: an active one or an invitation, never the patient or the treating practitioner. They leave its side threads
// with it and see nothing of it from the commit. A system message names the role; the removal is audited.
// ErrNotFound for a participant who is not in it; access as AddParticipant's.
func (s *Service) RemoveParticipant(ctx context.Context, c Caller, id, participantID string) error {
	var msg Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.manage(ctx, c, id)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(e.conv.Participants, func(p Participant) bool { return p.ID == participantID })
		switch {
		case i < 0:
			return ErrNotFound
		case e.conv.Participants[i].Role != RolePractitioner || e.conv.Participants[i].PractitionerRoleID == e.conv.PractitionerRoleID:
			return ErrForbidden
		}
		msg, err = s.leave(ctx, c, e.conv, e.conv.Participants[i], SystemParticipantRemoved)
		return err
	})
	if err != nil {
		return wrap("remove participant", s.refusal(ctx, c, id, err))
	}
	s.publish(ctx, MessageType, msg)
	return nil
}

// leave ends the participant p of conv with the system message code, audited as an update by c.
func (s *Service) leave(ctx context.Context, c Caller, conv Conversation, p Participant, code string) (Message, error) {
	if err := s.Repo.RemoveParticipant(ctx, p.ID, s.Clock.Now().UTC()); err != nil {
		return Message{}, err
	}
	msg, err := s.system(ctx, conv, code, map[string]string{"practitioner_role_id": p.PractitionerRoleID})
	if err != nil {
		return Message{}, err
	}
	return msg, s.record(ctx, c, auditevent.ActionUpdate, auditevent.CodeUpdate, "Patient/"+conv.PatientID, entityParticipant, 1)
}

// Invitations returns the calling patient's pending invitations of doctors from other clinics, audited as one search.
// ErrForbidden to staff.
func (s *Service) Invitations(ctx context.Context, c Caller) ([]Invitation, error) {
	if c.OrganizationID != "" {
		return nil, wrap("invitations", s.refusal(ctx, c, "", ErrForbidden))
	}
	var out []Invitation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.identify(ctx, c); err != nil {
			return err
		}
		var err error
		if out, err = s.Repo.Invitations(ctx, c.PatientID); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionExecute, auditevent.CodeSearchType, "", entityParticipant, int64(len(out)))
	})
	if err != nil {
		return nil, wrap("invitations", s.refusal(ctx, c, "", err))
	}
	return out, nil
}

// DecideInvitation records the calling patient's decision on the invitation id (a pending participant of their
// conversation). A grant is their FHIR Consent for that conversation and role (consent.GrantChat) and makes the
// doctor active, which needs the care consent with the clinic (access.GrantsChatGuest); a decline ends the
// invitation, whatever the care consent. A system message names the role; the decision is audited, all in one
// transaction. ErrInvalid for another decision, ErrNotFound for no pending invitation of theirs, ErrConsentRequired
// for a grant without the care consent, ErrForbidden to staff.
func (s *Service) DecideInvitation(ctx context.Context, c Caller, id string, d Decision) error {
	if d != DecisionGrant && d != DecisionDecline {
		return invalid("decision")
	}
	var msg Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, p, err := s.invitation(ctx, c, id, StatusPendingConsent, d == DecisionGrant)
		if err != nil {
			return err
		}
		if d == DecisionDecline {
			msg, err = s.leave(ctx, c, e.conv, p, SystemConsentDeclined)
			return err
		}
		if !access.GrantsChatGuest(e.who.viewer, e.facts) {
			return ErrConsentRequired
		}
		if _, err := s.ChatConsents.GrantChat(ctx, patientCaller(c), consent.ChatGrant{
			ConversationID: e.conv.ID, OrganizationID: e.conv.OrganizationID, PractitionerRoleID: p.PractitionerRoleID,
		}); err != nil {
			return err
		}
		if err := s.Repo.ActivateParticipant(ctx, p.ID); err != nil {
			return err
		}
		if msg, err = s.system(ctx, e.conv, SystemConsentGranted, map[string]string{"practitioner_role_id": p.PractitionerRoleID}); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionUpdate, auditevent.CodeUpdate, "Patient/"+e.conv.PatientID, entityParticipant, 1)
	})
	if err != nil {
		return wrap("decide invitation", s.refusal(ctx, c, "", err))
	}
	s.publish(ctx, MessageType, msg)
	return nil
}

// RevokeInvitation revokes the calling patient's grant of the invitation id: their chat consent ends and the doctor
// leaves the conversation and its side threads, from the commit (an open socket at its next check). A system message
// names the role; the revocation is audited. ErrNotFound for no granted invitation of theirs, ErrForbidden to staff.
func (s *Service) RevokeInvitation(ctx context.Context, c Caller, id string) error {
	var msg Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, p, err := s.invitation(ctx, c, id, StatusActive, false)
		switch {
		case err != nil:
			return err
		case p.OrganizationID == e.conv.OrganizationID: // a doctor of the clinic, not an invitation
			return ErrNotFound
		}
		_, err = s.ChatConsents.RevokeChat(ctx, patientCaller(c), e.conv.ID, p.PractitionerRoleID)
		switch {
		case errors.Is(err, consent.ErrInactive):
			return ErrNotFound
		case err != nil:
			return err
		}
		msg, err = s.leave(ctx, c, e.conv, p, SystemParticipantRemoved)
		return err
	})
	if err != nil {
		return wrap("revoke invitation", s.refusal(ctx, c, "", err))
	}
	s.publish(ctx, MessageType, msg)
	return nil
}

// invitation is the calling patient's practitioner participant id of status in one of their conversations, which it
// enters locked (for a write when write is set). ErrForbidden to staff, ErrNotFound otherwise.
func (s *Service) invitation(ctx context.Context, c Caller, id string, status ParticipantStatus, write bool) (entered, Participant, error) {
	if c.OrganizationID != "" {
		return entered{}, Participant{}, ErrForbidden
	}
	if !validUUID(id) {
		return entered{}, Participant{}, ErrNotFound
	}
	if _, err := s.identify(ctx, c); err != nil {
		return entered{}, Participant{}, err
	}
	p, err := s.Repo.Participant(ctx, id)
	switch {
	case err != nil:
		return entered{}, Participant{}, err
	case p.Role != RolePractitioner || p.Status != status:
		return entered{}, Participant{}, ErrNotFound
	}
	e, err := s.enter(ctx, c, p.ConversationID, write, true)
	if err != nil {
		return entered{}, Participant{}, err
	}
	if !e.who.patient() || e.conv.PatientID != c.PatientID { // not on row-level security alone
		return entered{}, Participant{}, ErrNotFound
	}
	return e, p, nil
}

// patientCaller is the patient c as the consent domain names them.
func patientCaller(c Caller) consent.Caller {
	return consent.Caller{AccountID: c.AccountID, PatientID: c.PatientID}
}

// system appends the system message code with its opaque arguments to conv (locked by the transaction), on Send's
// path: numbered and timed like any message.
func (s *Service) system(ctx context.Context, conv Conversation, code string, args map[string]string) (Message, error) {
	return s.appendMessage(ctx, conv, Message{SenderRole: RoleSystem, Kind: KindSystem, SystemCode: code, SystemArgs: args, ClientMessageID: newUUID()})
}

// newUUID is a random (version 4) UUID: a system message's client id, which no client repeats.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand never fails
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
