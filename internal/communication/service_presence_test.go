package communication_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
)

// receipts are the receipt frames published, as payloads.
func (f *fixture) receipts(t *testing.T) []communication.Receipt {
	t.Helper()
	var out []communication.Receipt
	for _, e := range f.pub.published() {
		if e.Type == communication.ReceiptType {
			var r communication.Receipt
			if err := json.Unmarshal(e.Payload, &r); err != nil {
				t.Fatal(err)
			}
			out = append(out, r)
		}
	}
	return out
}

// A read position that moves goes to the others as a receipt on the conversation's channel and is kept in Redis for
// a socket that opens later (SAL-30); one that does not move goes nowhere. Nothing of it is stored in Postgres but
// the reader's own position.
func TestMarkRead_TellsTheOthersAReceiptWhenThePositionMoves(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic one")
	f.send(t, pract, c.ID, "Synthetic two")
	part, _ := participantOf(c, me)
	for _, seq := range []int64{1, 1, 0, 2} {
		if err := f.svc.MarkRead(t.Context(), me, c.ID, seq); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := f.receipts(t), []communication.Receipt{{ParticipantID: part, Seq: 1}, {ParticipantID: part, Seq: 2}}; !slices.Equal(got, want) {
		t.Errorf("receipts %v, want %v", got, want)
	}
	if got, err := f.svc.Receipts(t.Context(), me, c.ID); err != nil || !slices.Contains(got, communication.Receipt{ParticipantID: part, Seq: 2}) {
		t.Errorf("Receipts() = %v, %v", got, err)
	}
	f.sal30.presence.mu.Lock()
	f.sal30.presence.err = errTag
	f.sal30.presence.mu.Unlock()
	f.send(t, pract, c.ID, "Synthetic three")
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 3); err != nil {
		t.Errorf("MarkRead() with Redis down = %v, want the position moved all the same", err)
	}
}

// Typing goes out at most once a TypingEvery per participant, naming them.
func TestTyping_GoesOutAtMostOnceASecond(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	part, _ := participantOf(c, me)
	for _, at := range []time.Duration{0, 300 * time.Millisecond, 999 * time.Millisecond, time.Second, 1500 * time.Millisecond} {
		f.clock.set(start.Add(at))
		if err := f.svc.Typing(t.Context(), c.ID, part); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, e := range f.pub.published() {
		if e.Type == communication.TypingType {
			var p communication.Typing
			_ = json.Unmarshal(e.Payload, &p)
			got = append(got, p.ParticipantID)
		}
	}
	if !slices.Equal(got, []string{part, part}) {
		t.Errorf("typing frames %v, want two (at 0 and 1 s)", got)
	}
}

// The receipts a socket gets when it opens are those of the conversation's current participants: a removed
// participant's receipt, still kept in Redis, is never sent (SAL-30 review, finding 4); nor anything to who may not
// read the conversation.
func TestReceipts_AreThoseOfTheCurrentParticipantsOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	part, _ := participantOf(c, me)
	const removed = "d0000009-0000-4000-8000-000000000009"
	for _, id := range []string{part, removed} {
		if err := f.sal30.presence.SetReceipt(t.Context(), c.ID, id, 4); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := f.svc.Receipts(t.Context(), pract, c.ID); err != nil || !slices.Equal(got, []communication.Receipt{{ParticipantID: part, Seq: 4}}) {
		t.Errorf("Receipts() = %v, %v; want the patient's only", got, err)
	}
	if got, err := f.svc.Receipts(t.Context(), other, c.ID); err == nil || len(got) != 0 {
		t.Errorf("another patient: Receipts() = %v, %v; want an error and none", got, err)
	}
}
