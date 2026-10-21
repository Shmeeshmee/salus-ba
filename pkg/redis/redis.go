// Package redis connects to Redis, which holds temporary state only: every key has a TTL and nothing in
// Redis is the only copy of anything (ADR 0005).
package redis

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Open connects to url (redis://host:port/db) and checks the connection.
func Open(ctx context.Context, url string) (*goredis.Client, error) {
	if url == "" {
		return nil, errors.New("redis: REDIS_URL is required")
	}
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	c := goredis.NewClient(opts)
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return c, nil
}
