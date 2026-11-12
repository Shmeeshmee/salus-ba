package communication

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// ReplayPage is the most messages one After gives: a reconnect's replay goes out a page at a time.
const ReplayPage = 100

// Subscription is where a socket starts: after From, and never before the caller's history_from_seq.
type Subscription struct {
	From           Cursor
	HistoryFromSeq int64
}

// Subscribe checks that the caller may follow the conversation's live messages and returns where the socket
// starts: after the cursor the client resumes from, or, without one, after the conversation's latest message
// (the client has loaded its page). The subscription is audited as an operation on the patient's chat.
// ErrInvalidCursor for a cursor the API did not give; access as enter's for a read.
func (s *Service) Subscribe(ctx context.Context, c Caller, id, after string) (Subscription, error) {
	var from Cursor
	var history int64
	if after != "" {
		var err error
		if from, err = ParseCursor(after); err != nil {
			return Subscription{}, err
		}
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		if after == "" {
			from = Cursor{SentAt: e.conv.LastSentAt, Seq: e.conv.LastSeq}
		}
		history = e.fromSeq()
		return s.record(ctx, c, auditevent.ActionExecute, auditevent.CodeOperation, "Patient/"+e.conv.PatientID, entityConversation, 1)
	})
	if err != nil {
		return Subscription{}, wrap("subscribe", s.refusal(ctx, c, id, err))
	}
	return Subscription{From: from, HistoryFromSeq: history}, nil
}

// After returns up to ReplayPage of the conversation's messages after the cursor, oldest first, checking the
// caller's access again (a revocation ends the socket at its next call); the rows it returns are audited as one
// read. The socket calls it to replay after a reconnect, to fill a gap in the live messages, and every few seconds.
func (s *Service) After(ctx context.Context, c Caller, id string, after Cursor) ([]Message, error) {
	var out []Message
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		if e.conv.LastSeq <= after.Seq {
			return nil // nothing new: the common case of the periodic check, one query fewer
		}
		if out, err = s.Repo.After(ctx, id, after, e.fromSeq(), ReplayPage); err != nil || len(out) == 0 {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Patient/"+e.conv.PatientID, entityCommunication, int64(len(out)))
	})
	if err != nil {
		return nil, wrap("after", err)
	}
	return out, nil
}

// PartitionsAhead is how many months of message partitions the daily job keeps, from this month (ADR 0012).
const PartitionsAhead = 3

// PartitionsArgs is the communication_partitions job the scheduler enqueues daily, and once when it starts
// leading. No arguments.
type PartitionsArgs struct{}

// Kind names the job in river_job.
func (PartitionsArgs) Kind() string { return "communication_partitions" }

// EnsurePartitions creates the message partitions of this month and the PartitionsAhead - 1 after it that do not
// exist yet: idempotent, so a missed day costs nothing while one of the months ahead remains.
func (s *Service) EnsurePartitions(ctx context.Context) error {
	return wrap("partitions", s.Repo.AddPartitions(ctx, s.Clock.Now().UTC(), PartitionsAhead))
}
