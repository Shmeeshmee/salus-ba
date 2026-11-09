package redis_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/redis"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// seen collects what a handler was given, and fails the entries it is told to, once each.
type seen struct {
	failOnce map[string]bool // values that fail the first time
	failed   map[string]bool
	got      []string // stream key + " " + value, in the order handled
	mu       sync.Mutex
}

func (s *seen) handle(_ context.Context, stream, _ string, values map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, _ := values["v"].(string)
	if s.failOnce[v] && !s.failed[v] {
		s.failed[v] = true
		return errors.New("a passing failure")
	}
	s.got = append(s.got, stream+" "+v)
	return nil
}

func (s *seen) values() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.got)
}

// waitFor waits until cond holds, for at most five seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := wallNow().Add(5 * time.Second)
	for !cond() {
		if wallNow().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func xadd(t *testing.T, rdb *goredis.Client, key, v string) {
	t.Helper()
	if err := rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: key, Values: []any{"v", v}}).Err(); err != nil {
		t.Fatal(err)
	}
}

func group(rdb *goredis.Client, consumer string) *redis.Group {
	return &redis.Group{
		Client: rdb, Clock: clock.System{}, Pattern: "consult:*:events", Name: "store", Consumer: consumer,
		Block: 50 * time.Millisecond, MinIdle: 200 * time.Millisecond,
	}
}

// run runs g until the test ends.
func run(t *testing.T, g *redis.Group, h redis.Handler) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		g.Run(ctx, h)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func pending(t *testing.T, rdb *goredis.Client, key string) int64 {
	t.Helper()
	p, err := rdb.XPending(t.Context(), key, "store").Result()
	if err != nil {
		t.Fatal(err)
	}
	return p.Count
}

// Every entry of every stream matching the pattern is handled once, in order per stream, and acknowledged:
// the streams there at the start and one that appears later. Keys that do not match, or are no stream, are
// left alone.
func TestGroup_HandlesEveryEntryOfEveryMatchingStreamOnce(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	xadd(t, rdb, "consult:a:events", "a1")
	xadd(t, rdb, "consult:a:events", "a2")
	xadd(t, rdb, "other:a:events", "x")
	if err := rdb.Set(t.Context(), "consult:s:events", "a string", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	s := &seen{}
	run(t, group(rdb, "c1"), s.handle)
	waitFor(t, "a1 and a2", func() bool { return len(s.values()) == 2 })
	xadd(t, rdb, "consult:b:events", "b1")
	xadd(t, rdb, "consult:a:events", "a3")
	waitFor(t, "b1 and a3", func() bool { return len(s.values()) == 4 })

	got := s.values()
	if !slices.Equal(got[:2], []string{"consult:a:events a1", "consult:a:events a2"}) ||
		!slices.Contains(got, "consult:b:events b1") || !slices.Contains(got, "consult:a:events a3") {
		t.Errorf("handled %v", got)
	}
	time.Sleep(100 * time.Millisecond) // two more reads: nothing more is handled
	if n := len(s.values()); n != 4 {
		t.Errorf("handled %d entries, want 4", n)
	}
	for _, key := range []string{"consult:a:events", "consult:b:events"} {
		if n := pending(t, rdb, key); n != 0 {
			t.Errorf("%s: %d entries pending, want none", key, n)
		}
	}
}

// An entry whose handler fails stays pending and is handled again once it has been idle MinIdle; the entries
// after it are not held up.
func TestGroup_HandlesAFailedEntryAgainAfterMinIdle(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	s := &seen{failOnce: map[string]bool{"a1": true}, failed: map[string]bool{}}
	xadd(t, rdb, "consult:a:events", "a1")
	xadd(t, rdb, "consult:a:events", "a2")
	began := wallNow()
	run(t, group(rdb, "c1"), s.handle)
	waitFor(t, "a1 handled again", func() bool { return slices.Contains(s.values(), "consult:a:events a1") })
	if took := wallNow().Sub(began); took < 200*time.Millisecond {
		t.Errorf("handled again after %v, want after MinIdle", took)
	}
	if got := s.values(); !slices.Equal(got, []string{"consult:a:events a2", "consult:a:events a1"}) {
		t.Errorf("handled %v, want a2 then a1 again", got)
	}
	waitFor(t, "nothing pending", func() bool { return pending(t, rdb, "consult:a:events") == 0 })
}

// Two members of the group share the entries: each entry is handled by one of them, once. A member that starts
// once they have stopped (a restart) handles what came after, and nothing that was acknowledged.
func TestGroup_MembersShareTheEntriesAndARestartResumes(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	s := &seen{}
	for i := range 40 {
		xadd(t, rdb, "consult:"+strconv.Itoa(i%4)+":events", strconv.Itoa(i))
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, c := range []string{"c1", "c2"} {
		wg.Go(func() { group(rdb, c).Run(ctx, s.handle) })
	}
	waitFor(t, "all 40", func() bool { return len(s.values()) == 40 })
	cancel()
	wg.Wait()

	xadd(t, rdb, "consult:0:events", "40")
	run(t, group(rdb, "c3"), s.handle)
	waitFor(t, "the 41st", func() bool { return len(s.values()) == 41 })
	time.Sleep(100 * time.Millisecond) // more reads: nothing more is handled
	got := s.values()
	slices.Sort(got)
	if len(got) != 41 || len(slices.Compact(got)) != 41 {
		t.Errorf("handled %d entries, %d distinct; want 41 each once", len(s.values()), len(slices.Compact(got)))
	}
}

// A Redis that fails is reported and retried, never ends the loop: Run returns only when its context ends.
func TestGroup_ReportsRedisErrorsAndKeepsGoing(t *testing.T) {
	t.Parallel()
	rdb := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 50 * time.Millisecond})
	t.Cleanup(func() { _ = rdb.Close() })
	var mu sync.Mutex
	var errs int
	g := group(rdb, "c1")
	g.OnError = func(error) {
		mu.Lock()
		errs++
		mu.Unlock()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	g.Run(ctx, func(context.Context, string, string, map[string]any) error { return nil })
	mu.Lock()
	defer mu.Unlock()
	if errs < 2 || errs > 40 {
		t.Errorf("reported %d errors in 2s with a 50ms back-off, want a few", errs)
	}
}

// wallNow is the real time, for the waits of a test against real servers.
func wallNow() time.Time { return clock.System{}.Now() }
