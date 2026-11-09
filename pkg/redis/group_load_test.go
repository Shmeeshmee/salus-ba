//go:build loadtest

package redis_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// The cost of finding the streams (SAL-6): one SCAN TYPE stream sweep over a keyspace of 100,000 other keys
// (sessions, rate limits) and 50 streams, as the consumer runs it once a second. Not part of make test.
func TestGroupDiscoveryCost(t *testing.T) {
	rdb := redistest.NewClient(t)
	pipe := rdb.Pipeline()
	for i := range 100_000 {
		pipe.Set(t.Context(), fmt.Sprintf("session:%d", i), "x", time.Hour)
	}
	if _, err := pipe.Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	for i := range 50 {
		xadd(t, rdb, fmt.Sprintf("consult:%d:events", i), "x")
	}
	for range 3 {
		began := wallNow()
		var cursor uint64
		var found int
		for {
			keys, next, err := rdb.ScanType(t.Context(), cursor, "consult:*:events", 1000, "stream").Result()
			if err != nil {
				t.Fatal(err)
			}
			found += len(keys)
			if cursor = next; cursor == 0 {
				break
			}
		}
		t.Logf("one sweep over 100,050 keys: %v, %d streams", wallNow().Sub(began), found)
	}
}
