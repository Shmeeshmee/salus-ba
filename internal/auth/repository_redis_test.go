package auth_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// everyKeyExpires fails the test when a key in rdb has no TTL (CLAUDE.md: every Redis key has one).
func everyKeyExpires(t *testing.T, rdb *goredis.Client) {
	t.Helper()
	keys, err := rdb.Keys(t.Context(), "*").Result()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if ttl := rdb.PTTL(t.Context(), k).Val(); ttl <= 0 {
			t.Errorf("key %q has TTL %v, want one", k, ttl)
		}
	}
}

func TestRedisChallenges_KeepTheAccountUnderTheHashUntilTheTTL(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	store := auth.NewRedisChallengeStore(rdb)
	ctx := t.Context()
	hash := []byte("0123456789abcdef0123456789abcdef")
	key := "mfa:challenge:" + hex.EncodeToString(hash)

	if err := store.Put(ctx, hash, accountID, 5*time.Minute); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if got, err := store.Get(ctx, hash); err != nil || got != accountID {
		t.Errorf("Get() = %q, %v; want the account", got, err)
	}
	if ttl := rdb.PTTL(ctx, key).Val(); ttl <= 4*time.Minute || ttl > 5*time.Minute {
		t.Errorf("key %s has TTL %v, want 5 minutes", key, ttl)
	}
	if err := store.Delete(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, hash); !errors.Is(err, auth.ErrInvalidChallenge) {
		t.Errorf("Get() after Delete = %v, want ErrInvalidChallenge", err)
	}

	if err := store.Put(ctx, hash, accountID, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := store.Get(ctx, hash); !errors.Is(err, auth.ErrInvalidChallenge) {
		t.Errorf("Get() after the TTL = %v, want ErrInvalidChallenge", err)
	}
}

func TestRedisChallenges_EndAtTheLimitOfWrongCodes(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	store := auth.NewRedisChallengeStore(rdb)
	ctx := t.Context()
	hash := []byte("challenge hash")
	if err := store.Put(ctx, hash, accountID, 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	for i := range 4 {
		if err := store.Fail(ctx, hash, 5); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Get(ctx, hash); err != nil {
			t.Fatalf("Get() after %d wrong codes = %v, want the challenge", i+1, err)
		}
		everyKeyExpires(t, rdb)
	}
	if err := store.Fail(ctx, hash, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, hash); !errors.Is(err, auth.ErrInvalidChallenge) {
		t.Errorf("Get() after 5 wrong codes = %v, want ErrInvalidChallenge", err)
	}

	if err := store.Fail(ctx, []byte("no such challenge"), 5); err != nil {
		t.Fatal(err)
	}
	if n := rdb.DBSize(ctx).Val(); n != 0 {
		t.Errorf("%d keys after failing an expired challenge, want none (it would have no TTL)", n)
	}
}

func TestRedisCounters_CountInAFixedWindowThatExpires(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	counters := auth.NewRedisCounters(rdb)
	ctx := t.Context()

	if n, left, err := counters.Get(ctx, "login:ip:abc"); err != nil || n != 0 || left != 0 {
		t.Errorf("Get() of no counter = %d, %v, %v; want 0, 0", n, left, err)
	}
	for want := int64(1); want <= 3; want++ {
		n, left, err := counters.Incr(ctx, "login:ip:abc", 15*time.Minute)
		if err != nil || n != want || left <= 14*time.Minute || left > 15*time.Minute {
			t.Errorf("Incr() = %d, %v, %v; want %d with the window's time left", n, left, err, want)
		}
	}
	if n, left, err := counters.Get(ctx, "login:ip:abc"); err != nil || n != 3 || left <= 0 {
		t.Errorf("Get() = %d, %v, %v; want 3 with time left", n, left, err)
	}
	if ttl := rdb.PTTL(ctx, "ratelimit:login:ip:abc").Val(); ttl <= 0 {
		t.Errorf("ratelimit:login:ip:abc TTL %v, want one", ttl)
	}
	everyKeyExpires(t, rdb)

	if _, _, err := counters.Incr(ctx, "login:email:def", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	_, _, _ = counters.Incr(ctx, "login:email:def", 100*time.Millisecond) // does not extend the window
	time.Sleep(80 * time.Millisecond)
	if n, _, err := counters.Get(ctx, "login:email:def"); err != nil || n != 0 {
		t.Errorf("Get() after the window = %d, %v; want 0: the window is fixed, not sliding", n, err)
	}
}

func TestRedis_PassesFailuresOn(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	store, counters := auth.NewRedisChallengeStore(rdb), auth.NewRedisCounters(rdb)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // every call fails as it would with Redis unreachable
	failures := map[string]error{}
	failures["Put"] = store.Put(ctx, []byte("h"), accountID, time.Minute)
	_, failures["Get"] = store.Get(ctx, []byte("h"))
	failures["Fail"] = store.Fail(ctx, []byte("h"), 5)
	failures["Delete"] = store.Delete(ctx, []byte("h"))
	_, _, failures["Incr"] = counters.Incr(ctx, "k", time.Minute)
	_, _, failures["Get counter"] = counters.Get(ctx, "k")
	for name, err := range failures {
		if err == nil || errors.Is(err, auth.ErrInvalidChallenge) {
			t.Errorf("%s() failing = %v, want the failure", name, err)
		}
	}
}
