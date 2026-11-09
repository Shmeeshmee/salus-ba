package transcript

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

type redisPublisher struct {
	rdb *goredis.Client
}

// NewRedisPublisher returns the Publisher on Redis: XADD with MAXLEN ~ StreamMaxLen, then the stream's TTL set
// to StreamTTL, in one transaction.
func NewRedisPublisher(rdb *goredis.Client) Publisher {
	return &redisPublisher{rdb: rdb}
}

func (p *redisPublisher) Publish(ctx context.Context, encounterID string, envelope []byte) error {
	key := StreamKey(encounterID)
	pipe := p.rdb.TxPipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{Stream: key, MaxLen: StreamMaxLen, Approx: true, Values: []any{StreamField, envelope}})
	pipe.Expire(ctx, key, StreamTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("transcript: publish: %w", err)
	}
	return nil
}
