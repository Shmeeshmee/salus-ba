// Package ttstest is an in-memory tts.Synthesizer for tests.
package ttstest

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/tts"
)

// Synthesizer returns Audio for every request, or fails with Err when set. It records its calls and is safe
// for concurrent use.
type Synthesizer struct {
	Err   error
	Audio []byte
	calls []tts.Request
	mu    sync.Mutex
}

// Synthesize records the request and streams a copy of Audio.
func (f *Synthesizer) Synthesize(_ context.Context, req tts.Request) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	if f.Err != nil {
		return nil, f.Err
	}
	return io.NopCloser(bytes.NewReader(slices.Clone(f.Audio))), nil
}

// Calls returns the requests so far, oldest first.
func (f *Synthesizer) Calls() []tts.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}
