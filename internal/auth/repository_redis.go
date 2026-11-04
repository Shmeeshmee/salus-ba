package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Redis holds temporary state only (ADR 0005): every key below has a TTL, and none is the only copy of
// anything durable. Keys are named by hashes, never by an e-mail, IP or token.

type redisChallenges struct {
	rdb *goredis.Client
}

// NewRedisChallengeStore returns the ChallengeStore on rdb: a hash mfa:challenge:<sha256 hex> with the
// account id and the count of wrong codes.
func NewRedisChallengeStore(rdb *goredis.Client) ChallengeStore {
	return &redisChallenges{rdb: rdb}
}

func challengeKey(hash []byte) string { return "mfa:challenge:" + hex.EncodeToString(hash) }

func (c *redisChallenges) Put(ctx context.Context, hash []byte, accountID string, ttl time.Duration) error {
	key := challengeKey(hash)
	_, err := c.rdb.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		p.HSet(ctx, key, "account", accountID)
		p.PExpire(ctx, key, ttl)
		return nil
	})
	return err
}

func (c *redisChallenges) Get(ctx context.Context, hash []byte) (string, error) {
	id, err := c.rdb.HGet(ctx, challengeKey(hash), "account").Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrInvalidChallenge
	}
	return id, err
}

// failScript counts a wrong code on an existing challenge only (HINCRBY on a missing key would create one
// without a TTL) and deletes it at the limit.
var failScript = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
local n = redis.call('HINCRBY', KEYS[1], 'fails', 1)
if n >= tonumber(ARGV[1]) then redis.call('DEL', KEYS[1]) end
return n`)

func (c *redisChallenges) Fail(ctx context.Context, hash []byte, limit int) error {
	return failScript.Run(ctx, c.rdb, []string{challengeKey(hash)}, limit).Err()
}

func (c *redisChallenges) Delete(ctx context.Context, hash []byte) error {
	return c.rdb.Del(ctx, challengeKey(hash)).Err()
}

type redisCounters struct {
	rdb *goredis.Client
}

// NewRedisCounters returns the Counters on rdb, each a key ratelimit:<key> whose TTL is the rest of its
// window.
func NewRedisCounters(rdb *goredis.Client) Counters {
	return &redisCounters{rdb: rdb}
}

func (c *redisCounters) Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	key = "ratelimit:" + key
	var n *goredis.IntCmd
	var left *goredis.DurationCmd
	_, err := c.rdb.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		n = p.Incr(ctx, key)
		p.Do(ctx, "PEXPIRE", key, window.Milliseconds(), "NX") // the first attempt starts the window; later ones do not move it
		left = p.PTTL(ctx, key)
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return n.Val(), left.Val(), nil
}

func (c *redisCounters) Get(ctx context.Context, key string) (int64, time.Duration, error) {
	key = "ratelimit:" + key
	var n *goredis.StringCmd
	var left *goredis.DurationCmd
	_, err := c.rdb.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		n = p.Get(ctx, key)
		left = p.PTTL(ctx, key)
		return nil
	})
	if errors.Is(err, goredis.Nil) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	count, err := n.Int64()
	return count, left.Val(), err
}
