package communication

import (
	"context"
	"errors"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Messages returns a page of the conversation's messages before the cursor (nil: the newest), newest first, of
// limit messages (0: PageSize; at most MaxPageSize), and the cursor of the older page, audited as one read of the
// rows it returns. A participant who joined later reads from their history_from_seq (SAL-28).
func (s *Service) Messages(ctx context.Context, c Caller, id string, before *Cursor, limit int) (Page, error) {
	switch {
	case limit == 0:
		limit = PageSize
	case limit < 1 || limit > MaxPageSize:
		return Page{}, invalid("limit")
	}
	var page Page
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		msgs, err := s.Repo.Messages(ctx, id, before, e.fromSeq(), limit+1)
		if err != nil {
			return err
		}
		if len(msgs) > limit {
			msgs = msgs[:limit]
			next := msgs[limit-1].Cursor()
			page.Next = &next
		}
		page.Messages = msgs
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Patient/"+e.conv.PatientID, entityCommunication, int64(len(msgs)))
	})
	if err != nil {
		return Page{}, wrap("messages", s.refusal(ctx, c, id, err))
	}
	return page, nil
}

// IdempotencyWindow is how far back from a conversation's last message a send looks for a repeat of its client id
// (ADR 0012): a client retries within seconds; the bound keeps the lookup to one or two monthly partitions, where an
// unbounded one reads every month the conversation ever had.
const IdempotencyWindow = 24 * time.Hour

// Send writes the caller's text in the conversation and returns it, and whether it is new (docs/plans/chat.md
// "Ordering", "Idempotent send"). Under the conversation's lock it finds a repeat of the client's id first, within
// IdempotencyWindow, and returns that message, unchanged and not published again; otherwise the message gets the next number and a time
// that never goes back, the sender has read it, the write is audited, and after the commit it goes to the live
// channel. ErrInvalid for a body or client id the contract refuses; access as enter's for a write.
func (s *Service) Send(ctx context.Context, c Caller, id, clientMessageID, body string) (Message, bool, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return Message{}, false, err
	}
	if !validUUID(clientMessageID) {
		return Message{}, false, invalid("client_message_id")
	}
	var out Message
	var created bool
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, true, true)
		if err != nil {
			return err
		}
		out, err = s.Repo.MessageByClientID(ctx, id, e.part.ID, clientMessageID, e.conv.LastSentAt.Add(-IdempotencyWindow))
		if !errors.Is(err, ErrNotFound) {
			return err // a repeat, or a failure
		}
		sent := s.Clock.Now().UTC().Truncate(time.Microsecond)
		if sent.Before(e.conv.LastSentAt) {
			sent = e.conv.LastSentAt
		}
		out, err = s.Repo.InsertMessage(ctx, Message{
			SentAt: sent, ConversationID: id, OrganizationID: e.conv.OrganizationID, PatientID: e.conv.PatientID,
			StaffOnly: e.conv.StaffOnly, Seq: e.conv.LastSeq + 1, SenderParticipantID: e.part.ID, SenderAccountID: c.AccountID,
			SenderRole: e.part.Role, ClientMessageID: clientMessageID, Kind: KindText, Body: body, VersionID: 1,
		})
		if err != nil {
			return err
		}
		if err := s.Repo.Advance(ctx, id, out.Seq, out.SentAt); err != nil {
			return err
		}
		if err := s.Repo.MarkRead(ctx, e.part.ID, out.Seq); err != nil {
			return err
		}
		created = true
		return s.record(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate, "Communication/"+out.ID, "", 0)
	})
	if err != nil {
		return Message{}, false, wrap("send", s.refusal(ctx, c, id, err))
	}
	if created {
		s.publish(ctx, MessageType, out)
	}
	return out, created, nil
}

// Edit replaces the body of the caller's own text message when version is its current version, and returns it
// with the next version; the earlier body is kept (communication_revision) and the edit published. ErrForbidden
// for someone else's message or a system message, ErrVersionConflict for a stale version, ErrNotFound for a
// message the caller may not read; access as enter's for a write.
func (s *Service) Edit(ctx context.Context, c Caller, id, messageID, body string, version int32) (Message, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	if !validUUID(messageID) {
		return Message{}, ErrNotFound
	}
	var out Message
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, true, false)
		if err != nil {
			return err
		}
		m, err := s.Repo.Message(ctx, id, messageID)
		switch {
		case err != nil:
			return err
		case m.Seq <= e.fromSeq():
			return ErrNotFound
		case m.Kind != KindText || m.SenderParticipantID != e.part.ID:
			return ErrForbidden
		case m.VersionID != version:
			return ErrVersionConflict
		}
		edited := s.Clock.Now().UTC().Truncate(time.Microsecond)
		m.Body, m.VersionID, m.EditedAt = body, m.VersionID+1, edited
		if err := s.Repo.UpdateBody(ctx, m); err != nil {
			return err
		}
		out = m
		return s.record(ctx, c, auditevent.ActionUpdate, auditevent.CodeUpdate, "Communication/"+m.ID, "", 0)
	})
	if err != nil {
		return Message{}, wrap("edit", s.refusal(ctx, c, id, err))
	}
	s.publish(ctx, MessageEditedType, out)
	return out, nil
}

// Revisions returns the earlier bodies of the message, oldest first, for a caller who may read it, audited.
func (s *Service) Revisions(ctx context.Context, c Caller, id, messageID string) ([]Revision, error) {
	if !validUUID(messageID) {
		return nil, ErrNotFound
	}
	var out []Revision
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		m, err := s.Repo.Message(ctx, id, messageID)
		switch {
		case err != nil:
			return err
		case m.Seq <= e.fromSeq():
			return ErrNotFound
		}
		if out, err = s.Repo.Revisions(ctx, id, messageID); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Communication/"+m.ID, "communication_revision", int64(len(out)))
	})
	if err != nil {
		return nil, wrap("revisions", s.refusal(ctx, c, id, err))
	}
	return out, nil
}

// MarkRead moves the caller's read position in the conversation to seq (owner, 2026-11-12); it never goes back,
// and seq is at most the conversation's last message. Not audited: the position is the reader's own, not the
// patient's data.
func (s *Service) MarkRead(ctx context.Context, c Caller, id string, seq int64) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		if seq < 0 || seq > e.conv.LastSeq {
			return invalid("seq")
		}
		return s.Repo.MarkRead(ctx, e.part.ID, seq)
	})
	return wrap("mark read", s.refusal(ctx, c, id, err))
}
