package videosession

import (
	"testing"
	"time"
)

// A track handed over while nobody reads the tracks never blocks LiveKit's callback nor the agent's leaving
// (Copilot on PR #36): past the buffer it is refused, and the tracks still close.
func TestListening_HandNeverBlocks(t *testing.T) {
	l := &listening{tracks: make(chan RemoteAudio, trackBuffer)}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range trackBuffer + 3 {
			l.hand(RemoteAudio{Identity: "p_x"})
		}
		l.end()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("hand or end blocked with nobody reading the tracks")
	}
	if l.hand(RemoteAudio{}) {
		t.Error("hand() after end = true, want the track refused")
	}
}
