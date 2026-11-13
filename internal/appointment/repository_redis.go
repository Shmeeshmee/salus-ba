package appointment

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Redis holds temporary state only: the link-lookup counters, ratelimit:<key>, each with the TTL of its window. Keys
// are named by an HMAC of the address, never the address or a token.
type redisLinkLimiter struct {
	rdb *goredis.Client
}

// NewRedisLinkLimiter returns the LinkLimiter on rdb.
func NewRedisLinkLimiter(rdb *goredis.Client) LinkLimiter {
	return &redisLinkLimiter{rdb: rdb}
}

func (l *redisLinkLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	key = "ratelimit:" + key
	var n *goredis.IntCmd
	_, err := l.rdb.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		n = p.Incr(ctx, key)
		p.Do(ctx, "PEXPIRE", key, window.Milliseconds(), "NX") // the first lookup starts the window; later ones do not move it
		return nil
	})
	if err != nil {
		return false, err
	}
	return n.Val() <= int64(limit), nil
}
