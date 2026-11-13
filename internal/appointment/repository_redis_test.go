package appointment_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestRedisLinkLimiter_AllowsUpToTheLimitInAFixedWindowWithATTL(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	limiter := appointment.NewRedisLinkLimiter(rdb)
	for i := 1; i <= 4; i++ {
		ok, err := limiter.Allow(t.Context(), "link:abc", 3, time.Minute)
		if err != nil || ok != (i <= 3) {
			t.Errorf("Allow() #%d = %v, %v; want %v", i, ok, err, i <= 3)
		}
	}
	if ttl := rdb.PTTL(t.Context(), "ratelimit:link:abc").Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("ratelimit:link:abc TTL %v, want the window's", ttl)
	}
	if ok, err := limiter.Allow(t.Context(), "link:other", 3, 50*time.Millisecond); err != nil || !ok {
		t.Fatalf("Allow() of another key = %v, %v", ok, err)
	}
	time.Sleep(80 * time.Millisecond)
	if n := rdb.Exists(t.Context(), "ratelimit:link:other").Val(); n != 0 {
		t.Error("the counter outlived its window")
	}
	_ = rdb.Close()
	if _, err := limiter.Allow(t.Context(), "link:abc", 3, time.Minute); err == nil {
		t.Error("Allow() on a closed client gave no error")
	}
}
