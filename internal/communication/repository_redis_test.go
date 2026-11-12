package communication_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

const (
	partA = "d0000001-0000-4000-8000-0000000000a1"
	partB = "d0000002-0000-4000-8000-0000000000b2"
)

// Typing is allowed once per TypingEvery per participant and conversation, by a key with that TTL: the next one is
// refused until it expires, another participant's is not.
func TestRedisPresence_AllowsTypingOnceASecondPerParticipant(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	p := communication.NewRedisPresence(rdb)
	conv := randomConversation()
	for _, tt := range []struct {
		part string
		want bool
	}{{partA, true}, {partA, false}, {partB, true}} {
		if got, err := p.AllowTyping(t.Context(), conv, tt.part); err != nil || got != tt.want {
			t.Errorf("AllowTyping(%s) = %v, %v; want %v", tt.part, got, err, tt.want)
		}
	}
	ttl, err := rdb.PTTL(t.Context(), communication.TypingKey(conv, partA)).Result()
	if err != nil || ttl <= 0 || ttl > communication.TypingEvery {
		t.Errorf("the typing key's TTL %v, %v; want at most %v", ttl, err, communication.TypingEvery)
	}
	time.Sleep(communication.TypingEvery + 50*time.Millisecond)
	if got, err := p.AllowTyping(t.Context(), conv, partA); err != nil || !got {
		t.Errorf("AllowTyping() after a second = %v, %v; want true", got, err)
	}
}

// A receipt is kept per participant, never lower than before, in one hash of the conversation with ReceiptTTL.
func TestRedisPresence_KeepsEachParticipantsHighestReceiptWithItsTTL(t *testing.T) {
	t.Parallel()
	rdb := redistest.NewClient(t)
	p := communication.NewRedisPresence(rdb)
	conv := randomConversation()
	if got, err := p.Receipts(t.Context(), conv); err != nil || len(got) != 0 {
		t.Fatalf("Receipts() of none = %v, %v", got, err)
	}
	for _, r := range []struct {
		part string
		seq  int64
	}{{partA, 3}, {partA, 2}, {partB, 7}, {partA, 5}} {
		if err := p.SetReceipt(t.Context(), conv, r.part, r.seq); err != nil {
			t.Fatal(err)
		}
	}
	got, err := p.Receipts(t.Context(), conv)
	if err != nil || len(got) != 2 || got[partA] != 5 || got[partB] != 7 {
		t.Errorf("Receipts() = %v, %v; want A at 5, B at 7", got, err)
	}
	ttl, err := rdb.TTL(t.Context(), communication.ReceiptsKey(conv)).Result()
	if err != nil || ttl <= communication.ReceiptTTL-time.Minute || ttl > communication.ReceiptTTL {
		t.Errorf("the receipts' TTL %v, %v; want %v", ttl, err, communication.ReceiptTTL)
	}
}

// randomConversation is a fresh synthetic id: the compose Redis is shared by every test run.
func randomConversation() string {
	return fmt.Sprintf("c0000000-0000-4000-8000-%012x", rand.Int64N(1<<48))
}
