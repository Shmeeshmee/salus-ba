package communication

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// OpenThread opens a doctor-only side thread of the conversation id (docs/plans/chat.md, SAL-28) and returns it with
// its participants: the conversation's active practitioners at that moment, who keep it (a doctor added later is not
// in it, one removed from the conversation leaves it). It belongs to the conversation's clinic, patient and treating
// practitioner, is kept in the record like any conversation, and the patient never sees it (row-level security and
// enter). Opened by a practitioner who may write in the conversation or a granted guest (access.OpensChatThread),
// audited. ErrInvalid for a side thread's own side thread, ErrForbidden to the patient and anyone else; access as
// enter's for a write.
func (s *Service) OpenThread(ctx context.Context, c Caller, id string) (Conversation, error) {
	var out Conversation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.threadsOf(ctx, c, id, true)
		if err != nil {
			return err
		}
		now := s.Clock.Now().UTC()
		var parts []Participant
		for _, p := range e.conv.Participants {
			if p.Role != RolePractitioner || p.Status != StatusActive {
				continue
			}
			part := Participant{
				PractitionerRoleID: p.PractitionerRoleID, OrganizationID: p.OrganizationID, AddedBy: c.AccountID, AddedAt: now,
				Role: RolePractitioner, Status: StatusActive,
			}
			if p.PractitionerRoleID == e.who.role { // the opener first: a guest's row-level security sees the thread through it
				parts = append([]Participant{part}, parts...)
			} else {
				parts = append(parts, part)
			}
		}
		out, err = s.Repo.CreateThread(ctx, Conversation{
			Created: now, OrganizationID: e.conv.OrganizationID, PatientID: e.conv.PatientID, PractitionerRoleID: e.conv.PractitionerRoleID,
			EncounterID: e.conv.EncounterID, ParentID: e.conv.ID, CreatedBy: c.AccountID, StaffOnly: true,
		}, parts)
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate, "Patient/"+out.PatientID, entityConversation, 1)
	})
	if err != nil {
		return Conversation{}, wrap("open thread", s.refusal(ctx, c, id, err))
	}
	return out, nil
}

// Threads returns the side threads of the conversation id the calling practitioner takes part in, newest first, with
// their unread counts, audited as one search; access as OpenThread's for a read.
func (s *Service) Threads(ctx context.Context, c Caller, id string) ([]Conversation, error) {
	var out []Conversation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.threadsOf(ctx, c, id, false)
		if err != nil {
			return err
		}
		if out, err = s.Repo.Threads(ctx, id, e.who.role); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionExecute, auditevent.CodeSearchType, "Patient/"+e.conv.PatientID, entityConversation, int64(len(out)))
	})
	if err != nil {
		return nil, wrap("threads", s.refusal(ctx, c, id, err))
	}
	return out, nil
}

// threadsOf enters the conversation id for a practitioner who may open its side threads: never the patient, never
// a side thread's.
func (s *Service) threadsOf(ctx context.Context, c Caller, id string, write bool) (entered, error) {
	e, err := s.enter(ctx, c, id, write, false)
	switch {
	case err != nil:
		return entered{}, err
	case e.who.patient() || !access.OpensChatThread(e.who.viewer, e.facts, e.in, e.granted):
		return entered{}, ErrForbidden
	case e.conv.StaffOnly:
		return entered{}, invalid("id")
	}
	return e, nil
}
