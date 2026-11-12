package redis_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/redis"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// Pub/sub channels are shared by every database of the server, so each test names its own.
func channel(t *testing.T) string {
	t.Helper()
	return "hubtest:" + rand.Text()
}

func receive(t *testing.T, ch <-chan []byte) string {
	t.Helper()
	select {
	case b, ok := <-ch:
		if !ok {
			t.Fatal("the subscription closed")
		}
		return string(b)
	case <-time.After(5 * time.Second):
		t.Fatal("nothing received within 5 s")
	}
	return ""
}

func quiet(t *testing.T, ch <-chan []byte) {
	t.Helper()
	select {
	case b := <-ch:
		t.Fatalf("received %q, want nothing", b)
	case <-time.After(200 * time.Millisecond):
	}
}

func newHub(t *testing.T) *redis.Hub {
	t.Helper()
	rdb := redistest.NewClient(t)
	h := redis.NewHub(t.Context(), rdb, 4)
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func TestHub_HandsEachMessageToEverySubscriberOfItsChannelOnly(t *testing.T) {
	t.Parallel()
	h := newHub(t)
	a, b := channel(t), channel(t)
	a1, stopA1, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopA1()
	a2, stopA2, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopA2()
	b1, stopB1, err := h.Subscribe(t.Context(), b)
	if err != nil {
		t.Fatal(err)
	}
	defer stopB1()

	if err := h.Publish(t.Context(), a, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, a1); got != "one" {
		t.Errorf("first subscriber got %q", got)
	}
	if got := receive(t, a2); got != "one" {
		t.Errorf("second subscriber got %q", got)
	}
	quiet(t, b1)
}

func TestHub_AStoppedSubscriptionReceivesNothingAndTheOthersGoOn(t *testing.T) {
	t.Parallel()
	h := newHub(t)
	a := channel(t)
	first, stopFirst, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	second, stopSecond, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopSecond()
	stopFirst()
	stopFirst() // twice is harmless
	if _, ok := <-first; ok {
		t.Error("a stopped subscription is still open")
	}
	if err := h.Publish(t.Context(), a, []byte("two")); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, second); got != "two" {
		t.Errorf("got %q", got)
	}
}

func TestHub_ResubscribesAChannelAfterItsLastSubscriberLeft(t *testing.T) {
	t.Parallel()
	h := newHub(t)
	a := channel(t)
	_, stop, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	again, stopAgain, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopAgain()
	if err := h.Publish(t.Context(), a, []byte("three")); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, again); got != "three" {
		t.Errorf("got %q", got)
	}
}

// A subscriber that does not keep up loses messages instead of holding up the others: its buffer (4 here) fills and
// the rest is dropped. The subscriber finds the gap itself (Postgres is the record, ADR 0012).
func TestHub_DropsForASlowSubscriberWithoutHoldingUpTheOthers(t *testing.T) {
	t.Parallel()
	h := newHub(t)
	a := channel(t)
	slow, stopSlow, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopSlow()
	fast, stopFast, err := h.Subscribe(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	defer stopFast()
	for i := range 10 {
		if err := h.Publish(t.Context(), a, []byte{byte('0' + i)}); err != nil {
			t.Fatal(err)
		}
		if got := receive(t, fast); got != string(rune('0'+i)) {
			t.Fatalf("fast subscriber got %q at %d", got, i)
		}
	}
	n := 0
	for len(slow) > 0 {
		<-slow
		n++
	}
	if n != 4 {
		t.Errorf("slow subscriber kept %d, want its buffer of 4", n)
	}
}

func TestHub_SubscribeFailsOnceClosedOrWithAnEndedContext(t *testing.T) {
	t.Parallel()
	h := newHub(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := h.Subscribe(ctx, channel(t)); err == nil {
		t.Error("subscribed with an ended context")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.Subscribe(t.Context(), channel(t)); err == nil {
		t.Error("subscribed to a closed hub")
	}
}
