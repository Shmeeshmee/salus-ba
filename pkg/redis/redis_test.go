package redis_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/redis"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestOpen_ConnectsAndPings(t *testing.T) {
	t.Parallel()

	c, err := redis.Open(t.Context(), redistest.URL(t))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer c.Close()
	if err := c.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestOpen_FailsWhenServerIsUnreachable(t *testing.T) {
	t.Parallel()

	if _, err := redis.Open(t.Context(), "redis://127.0.0.1:1/0?dial_timeout=1s&max_retries=-1"); err == nil {
		t.Fatal("Open() error = nil, want a connection error")
	}
}

func TestOpen_RejectsAnEmptyURL(t *testing.T) {
	t.Parallel()

	if _, err := redis.Open(t.Context(), ""); err == nil {
		t.Fatal("Open(\"\") error = nil, want an error")
	}
}
