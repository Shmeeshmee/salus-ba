package transcript

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// streamPage is the most entries one Read gives: a reconnect's replay goes out a page at a time.
const streamPage = 100

type redisStream struct {
	rdb *goredis.Client
}

// NewRedisStream returns the Stream on rdb. Each Read that waits holds one of rdb's connections while it waits.
func NewRedisStream(rdb *goredis.Client) Stream {
	return &redisStream{rdb: rdb}
}

// Read is XREAD COUNT streamPage BLOCK wait on the encounter's stream: the entries strictly after the id after.
func (s *redisStream) Read(ctx context.Context, encounterID, after string, wait time.Duration) ([]Entry, error) {
	res, err := s.rdb.XRead(ctx, &goredis.XReadArgs{
		Streams: []string{StreamKey(encounterID), after}, Count: streamPage, Block: max(wait, time.Millisecond),
	}).Result()
	switch {
	case errors.Is(err, goredis.Nil): // the wait passed
		return nil, nil
	case err != nil:
		return nil, err
	}
	var out []Entry
	for _, stream := range res {
		for _, m := range stream.Messages {
			e := Entry{ID: m.ID}
			if v, ok := m.Values[StreamField].(string); ok {
				e.Envelope = []byte(v)
			}
			out = append(out, e)
		}
	}
	return out, nil
}

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
