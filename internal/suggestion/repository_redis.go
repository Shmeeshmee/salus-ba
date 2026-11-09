package suggestion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/transcript"
)

type redisPublisher struct {
	rdb *goredis.Client
}

// NewRedisPublisher returns the Publisher on Redis, as every producer of the stream writes it (transcript):
// XADD under transcript.StreamField with MAXLEN ~ transcript.StreamMaxLen, then the stream's TTL set to
// transcript.StreamTTL, in one transaction.
func NewRedisPublisher(rdb *goredis.Client) Publisher {
	return &redisPublisher{rdb: rdb}
}

func (p *redisPublisher) Publish(ctx context.Context, encounterID string, envelope []byte) error {
	key := transcript.StreamKey(encounterID)
	pipe := p.rdb.TxPipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{Stream: key, MaxLen: transcript.StreamMaxLen, Approx: true, Values: []any{transcript.StreamField, envelope}})
	pipe.Expire(ctx, key, transcript.StreamTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("suggestion: publish: %w", err)
	}
	return nil
}

type redisLive struct {
	rdb *goredis.Client
}

// NewRedisLive returns Live on Redis: the streams found by SCAN TYPE stream, read with XRANGE, and a lease per
// encounter (suggest:<encounter_id>:lease), whose value is its owner and which expires when not renewed.
func NewRedisLive(rdb *goredis.Client) Live {
	return &redisLive{rdb: rdb}
}

// streamPrefix and streamSuffix frame an encounter id in its stream's key (transcript.StreamKey).
var streamPrefix, streamSuffix, _ = strings.Cut(transcript.StreamKey("\x00"), "\x00")

func (l *redisLive) Encounters(ctx context.Context) ([]string, error) {
	var out []string
	var cursor uint64
	for {
		keys, next, err := l.rdb.ScanType(ctx, cursor, transcript.StreamKey("*"), 1000, "stream").Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			out = append(out, strings.TrimSuffix(strings.TrimPrefix(k, streamPrefix), streamSuffix))
		}
		if cursor = next; cursor == 0 {
			return out, nil
		}
	}
}

// claimScript takes the lease when it is free, or renews it when the owner holds it: 1 when the owner holds it
// after the call, 0 when another does.
var claimScript = goredis.NewScript(`
if redis.call('SET', KEYS[1], ARGV[1], 'NX', 'PX', ARGV[2]) then return 1 end
if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('PEXPIRE', KEYS[1], ARGV[2]) return 1 end
return 0`)

func (l *redisLive) Claim(ctx context.Context, encounterID, owner string, ttl time.Duration) (bool, error) {
	n, err := claimScript.Run(ctx, l.rdb, []string{"suggest:" + encounterID + ":lease"}, owner, ttl.Milliseconds()).Int()
	if err != nil {
		return false, fmt.Errorf("suggestion: claim: %w", err)
	}
	return n == 1, nil
}

func (l *redisLive) Entries(ctx context.Context, encounterID, after string, count int) ([]transcript.Entry, error) {
	start := "-"
	if after != "" {
		start = "(" + after
	}
	msgs, err := l.rdb.XRangeN(ctx, transcript.StreamKey(encounterID), start, "+", int64(count)).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("suggestion: entries: %w", err)
	}
	out := make([]transcript.Entry, len(msgs))
	for i, m := range msgs {
		out[i].ID = m.ID
		if v, ok := m.Values[transcript.StreamField].(string); ok {
			out[i].Envelope = []byte(v)
		}
	}
	return out, nil
}
