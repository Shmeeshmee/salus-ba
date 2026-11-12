// Package pushtest is the in-memory push.Sender: the default of every binary until the platforms' credentials are
// set (internal/config LoadPush), and the fake of the tests. It sends nothing anywhere. Safe for concurrent use.
package pushtest

import (
	"context"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/push"
)

// defaultKeep is how many messages a Sender keeps when Keep is zero.
const defaultKeep = 1000

// Sender records the messages it is sent, the newest Keep of them (defaultKeep when zero), so a long-running
// process that runs on it does not grow. It answers push.ErrTokenGone for a token marked Gone, and Err, when set,
// for any other.
type Sender struct {
	Err  error
	gone map[string]bool
	sent []push.Message
	Keep int
	mu   sync.Mutex
}

// Send records m, or fails as told.
func (s *Sender) Send(_ context.Context, m push.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gone[m.Token] {
		return push.ErrTokenGone
	}
	if s.Err != nil {
		return s.Err
	}
	keep := s.Keep
	if keep <= 0 {
		keep = defaultKeep
	}
	s.sent = append(s.sent, m)
	if len(s.sent) > keep {
		s.sent = append([]push.Message(nil), s.sent[len(s.sent)-keep:]...)
	}
	return nil
}

// Gone marks token as no longer reaching a device.
func (s *Sender) Gone(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gone == nil {
		s.gone = map[string]bool{}
	}
	s.gone[token] = true
}

// Sent returns a copy of the messages kept, oldest first.
func (s *Sender) Sent() []push.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]push.Message(nil), s.sent...)
}
