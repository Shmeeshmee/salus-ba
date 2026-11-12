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

// ReceiptsKey is the hash of a conversation's receipts: participant id to the number they have read up to.
func ReceiptsKey(conversationID string) string { return "chat:" + conversationID + ":receipts" }

func (p *redisPresence) AllowTyping(ctx context.Context, conversationID, participantID string) (bool, error) {
	ok, err := p.rdb.SetNX(ctx, TypingKey(conversationID, participantID), 1, TypingEvery).Result()
	if err != nil {
		return false, fmt.Errorf("communication: typing: %w", err)
	}
	return ok, nil
}

// setReceipt raises the participant's field of the hash to the number, never lowers it, and gives the hash its TTL.
var setReceipt = goredis.NewScript(`
local current = tonumber(redis.call('HGET', KEYS[1], ARGV[1]) or '0')
if tonumber(ARGV[2]) > current then
	redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
end
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return 1`)

func (p *redisPresence) SetReceipt(ctx context.Context, conversationID, participantID string, seq int64) error {
	err := setReceipt.Run(ctx, p.rdb, []string{ReceiptsKey(conversationID)}, participantID, seq, ReceiptTTL.Milliseconds()).Err()
	if err != nil {
		return fmt.Errorf("communication: receipt: %w", err)
	}
	return nil
}

func (p *redisPresence) Receipts(ctx context.Context, conversationID string) (map[string]int64, error) {
	all, err := p.rdb.HGetAll(ctx, ReceiptsKey(conversationID)).Result()
	if err != nil {
		return nil, fmt.Errorf("communication: receipts: %w", err)
	}
	out := make(map[string]int64, len(all))
	for id, v := range all {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue // not ours: nothing else writes the hash
		}
		out[id] = n
	}
	return out, nil
}
