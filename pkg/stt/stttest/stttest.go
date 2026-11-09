// Package stttest is an in-memory stt.Recognizer for tests: every stream plays the same scripted results and
// records the audio it was sent.
package stttest

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/stt"
)

// Recognizer gives every stream the results of Results, in order, or fails Stream with Err when set. With
// DiscardAudio its streams keep none of the audio they are sent (for a long-running fake, as in cmd/agent).
type Recognizer struct {
	Err          error
	Results      []stt.Result
	streams      []*Stream
	mu           sync.Mutex
	DiscardAudio bool
}

// Stream opens a stream that plays the script.
func (r *Recognizer) Stream(ctx context.Context, cfg stt.Config) (stt.Stream, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	results := make(chan stt.Result, len(r.Results))
	for _, res := range r.Results {
		results <- res
	}
	s := &Stream{Config: cfg, ctx: ctx, results: results, closed: make(chan struct{}), discard: r.DiscardAudio}
	r.mu.Lock()
	r.streams = append(r.streams, s)
	r.mu.Unlock()
	return s, nil
}

// Streams returns the streams opened so far, oldest first.
func (r *Recognizer) Streams() []*Stream {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.streams)
}

// Stream is one fake recognition. Config is what it was opened with.
type Stream struct {
	ctx     context.Context // Recv ends with the context the stream was opened with, as a real stream does
	results chan stt.Result
	closed  chan struct{}
	Config  stt.Config
	audio   []byte
	mu      sync.Mutex
	isDone  bool
	discard bool
}

// Send records the audio.
func (s *Stream) Send(pcm []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isDone {
		return stt.ErrClosed
	}
	if !s.discard {
		s.audio = append(s.audio, pcm...)
	}
	return nil
}

// CloseSend ends the audio; Recv returns io.EOF once the script is played.
func (s *Stream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isDone {
		return stt.ErrClosed
	}
	s.isDone = true
	close(s.closed)
	return nil
}

// Recv returns the next scripted result; after the script, io.EOF once the audio is closed, or the context's
// error once it is done.
func (s *Stream) Recv() (stt.Result, error) {
	select {
	case res := <-s.results:
		return res, nil
	default:
	}
	select {
	case <-s.closed:
		if err := s.ctx.Err(); err != nil { // both ready: select picks either, the context wins
			return stt.Result{}, err
		}
		return stt.Result{}, io.EOF
	case <-s.ctx.Done():
		return stt.Result{}, s.ctx.Err()
	}
}

// Audio returns a copy of the audio sent so far.
func (s *Stream) Audio() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.audio)
}
