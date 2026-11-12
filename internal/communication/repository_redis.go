package communication

import (
	"context"
	"fmt"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

type redisPresence struct {
	rdb *goredis.Client
}

// NewRedisPresence returns the Presence on Redis (ADR 0005: temporary state, every key with a TTL).
func NewRedisPresence(rdb *goredis.Client) Presence {
	return &redisPresence{rdb: rdb}
}

// TypingKey is the key that holds back a participant's next typing frame for TypingEvery.
func TypingKey(conversationID, participantID string) string {
	return "chat:" + conversationID + ":typing:" + participantID
}

// ReceiptKey is the key of how far a participant of a conversation has read, with its own ReceiptTTL.
func ReceiptKey(conversationID, participantID string) string {
	return "chat:" + conversationID + ":receipt:" + participantID
}

func (p *redisPresence) AllowTyping(ctx context.Context, conversationID, participantID string) (bool, error) {
	ok, err := p.rdb.SetNX(ctx, TypingKey(conversationID, participantID), 1, TypingEvery).Result()
	if err != nil {
		return false, fmt.Errorf("communication: typing: %w", err)
	}
	return ok, nil
}

// setReceipt raises the participant's receipt to the number, never lowers it, and gives it its TTL.
var setReceipt = goredis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if tonumber(ARGV[1]) > current then
	redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
else
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 1`)

func (p *redisPresence) SetReceipt(ctx context.Context, conversationID, participantID string, seq int64) error {
	err := setReceipt.Run(ctx, p.rdb, []string{ReceiptKey(conversationID, participantID)}, seq, ReceiptTTL.Milliseconds()).Err()
	if err != nil {
		return fmt.Errorf("communication: receipt: %w", err)
	}
	return nil
}

func (p *redisPresence) Receipts(ctx context.Context, conversationID string, participantIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(participantIDs))
	if len(participantIDs) == 0 {
		return out, nil
	}
	keys := make([]string, len(participantIDs))
	for i, id := range participantIDs {
		keys[i] = ReceiptKey(conversationID, id)
	}
	vals, err := p.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("communication: receipts: %w", err)
	}
	for i, v := range vals {
		s, ok := v.(string)
		if !ok {
			continue // none kept, or expired
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			out[participantIDs[i]] = n
		}
	}
	return out, nil
}
