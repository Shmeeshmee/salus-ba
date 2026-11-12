package communication

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/wire"
)

// What the participants show each other live (owner, 2026-11-12; SAL-30): read receipts and the typing indicator,
// ephemeral, in Redis only (pub/sub on the conversation's channel, keys with a TTL where state is needed), never
// stored in Postgres. A participant's own read position is stored (MarkRead); what the others see of it is not.
const (
	// TypingEvery is how often a participant's typing goes out, at most.
	TypingEvery = time.Second
	// TypingFor is how long the apps show a typing indicator after its last frame (api/openapi.yaml).
	TypingFor = 5 * time.Second
	// ReceiptTTL is how long a participant's receipt is kept for a socket that opens later: past it, a receipt says
	// little the conversation's own order does not.
	ReceiptTTL = 7 * 24 * time.Hour
)

// Typing tells the conversation's other sockets that the participant types: at most once per TypingEvery (the
// Presence's key with that TTL), else nothing. The socket calls it for its caller's participant, whose access it
// checks again every CheckEvery.
func (s *Service) Typing(ctx context.Context, conversationID, participantID string) error {
	if s.Presence == nil {
		return nil
	}
	ok, err := s.Presence.AllowTyping(ctx, conversationID, participantID)
	if err != nil || !ok {
		return wrap("typing", err)
	}
	frame, err := wire.Encode(TypingType, FrameVersion, Typing{ParticipantID: participantID})
	if err != nil {
		return wrap("typing", err)
	}
	return wrap("typing", s.Publisher.Publish(ctx, ChannelKey(conversationID), frame))
}

// receipt tells the conversation's sockets how far the participant has now read, and keeps it for ReceiptTTL for a
// socket that opens later; a failure is logged, without anything of the message: a receipt is a courtesy.
func (s *Service) receipt(ctx context.Context, conversationID, participantID string, seq int64) {
	if s.Presence == nil {
		return
	}
	err := s.Presence.SetReceipt(ctx, conversationID, participantID, seq)
	if err == nil {
		var frame []byte
		if frame, err = wire.Encode(ReceiptType, FrameVersion, Receipt{ParticipantID: participantID, Seq: seq}); err == nil {
			err = s.Publisher.Publish(ctx, ChannelKey(conversationID), frame)
		}
	}
	if err != nil {
		s.Log.WarnContext(ctx, "chat receipt failed", "conversation_id", conversationID, "error", err.Error())
	}
}

// Receipts are how far each participant has read, as kept for ReceiptTTL, ordered by participant: a socket sends
// them once it has replayed. The socket has checked the caller's access.
func (s *Service) Receipts(ctx context.Context, conversationID string) ([]Receipt, error) {
	if s.Presence == nil {
		return nil, nil
	}
	kept, err := s.Presence.Receipts(ctx, conversationID)
	if err != nil {
		return nil, wrap("receipts", err)
	}
	out := make([]Receipt, 0, len(kept))
	for id, seq := range kept {
		out = append(out, Receipt{ParticipantID: id, Seq: seq})
	}
	slices.SortFunc(out, func(a, b Receipt) int { return strings.Compare(a.ParticipantID, b.ParticipantID) })
	return out, nil
}
