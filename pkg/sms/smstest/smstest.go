// Package smstest is the in-memory sms.Sender: the default of every binary until the owner chooses an SMS provider
// and adds its credentials (internal/config LoadSMS), and the fake of the tests. It sends nothing anywhere. Safe for
// concurrent use.
package smstest

import (
	"context"
	"strconv"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/sms"
)

// defaultKeep is how many messages a Sender keeps when Keep is zero.
const defaultKeep = 1000

// Sender records the messages it is sent, the newest Keep of them (defaultKeep when zero), so a long-running process
// that runs on it does not grow. It checks each message as a provider would (sms.Check), answers sms.ErrRejected for a
// number marked Unreachable, and Err, when set, for any other. Each message gets the id "fake-<n>"; a message with the
// Reference of one it already took gets that one's id and is not recorded again, as a provider that deduplicates on
// the client reference does.
type Sender struct {
	Err         error
	unreachable map[string]bool
	byReference map[string]string
	sent        []sms.Message
	Keep        int
	n           int
	mu          sync.Mutex
}

// Send records m and returns its id, or fails as told.
func (s *Sender) Send(_ context.Context, m sms.Message) (string, error) {
	if err := sms.Check(m); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.unreachable[m.To]:
		return "", sms.ErrRejected
	case s.Err != nil:
		return "", s.Err
	}
	if id, ok := s.byReference[m.Reference]; ok && m.Reference != "" {
		return id, nil
	}
	s.n++
	id := "fake-" + strconv.Itoa(s.n)
	if m.Reference != "" {
		if s.byReference == nil {
			s.byReference = map[string]string{}
		}
		s.byReference[m.Reference] = id
	}
	keep := s.Keep
	if keep <= 0 {
		keep = defaultKeep
	}
	s.sent = append(s.sent, m)
	if len(s.sent) > keep {
		s.sent = append([]sms.Message(nil), s.sent[len(s.sent)-keep:]...)
	}
	return id, nil
}

// Unreachable marks the number as one the provider rejects.
func (s *Sender) Unreachable(to string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unreachable == nil {
		s.unreachable = map[string]bool{}
	}
	s.unreachable[to] = true
}

// Sent returns a copy of the messages kept, oldest first.
func (s *Sender) Sent() []sms.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sms.Message(nil), s.sent...)
}
