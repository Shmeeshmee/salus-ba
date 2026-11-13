package smstest_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/sms"
	"github.com/Shmeeshmee/salus/pkg/sms/smstest"
)

var _ sms.Sender = (*smstest.Sender)(nil)

func TestSender_RecordsChecksAndDeduplicatesByReference(t *testing.T) {
	t.Parallel()
	s := &smstest.Sender{}
	m := sms.Message{To: "+38760000001", Body: "Reminder", Reference: "d1"}
	id, err := s.Send(t.Context(), m)
	if err != nil || id != "fake-1" {
		t.Fatalf("Send() = %q, %v", id, err)
	}
	if again, err := s.Send(t.Context(), m); err != nil || again != id || len(s.Sent()) != 1 {
		t.Errorf("a repeat = %q, %v with %d sent; want the same id and one message", again, err, len(s.Sent()))
	}
	if id2, _ := s.Send(t.Context(), sms.Message{To: m.To, Body: "Other"}); id2 != "fake-2" {
		t.Errorf("a new message got %q", id2)
	}
	if _, err := s.Send(t.Context(), sms.Message{To: "060", Body: "x"}); !errors.Is(err, sms.ErrInvalidMessage) {
		t.Errorf("an invalid number = %v", err)
	}
	s.Unreachable("+38760000002")
	if _, err := s.Send(t.Context(), sms.Message{To: "+38760000002", Body: "x"}); !errors.Is(err, sms.ErrRejected) {
		t.Errorf("an unreachable number = %v", err)
	}
	s.Err = errors.New("down")
	if _, err := s.Send(t.Context(), sms.Message{To: m.To, Body: "x"}); !errors.Is(err, s.Err) {
		t.Errorf("with Err set = %v", err)
	}
}

func TestSender_KeepsTheNewestAndIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	s := &smstest.Sender{Keep: 5}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := s.Send(t.Context(), sms.Message{To: "+38760000001", Body: "x"}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := len(s.Sent()); n != 5 {
		t.Errorf("kept %d, want 5", n)
	}
}
