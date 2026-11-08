// Package stt is the port of a streaming speech-to-text engine: audio in, partial and final text out. It knows
// nothing of consultations; the caller maps results to its own events. stttest has an in-memory fake.
package stt

import (
	"context"
	"errors"
	"time"
)

// ErrClosed is returned by Send or CloseSend on a stream whose audio is already closed.
var ErrClosed = errors.New("stt: stream closed")

// Config is what one stream recognises: the spoken language (a BCP 47 tag) and the sample rate of the audio,
// 16-bit little-endian PCM, mono.
type Config struct {
	Lang       string
	SampleRate int
}

// Result is a hypothesis for the utterance being spoken. Partial results (Final false) revise one another;
// a final result closes the utterance, and the next result starts a new one. Start and End are offsets from
// the start of the stream's audio.
type Result struct {
	Text  string
	Lang  string
	Start time.Duration
	End   time.Duration
	Final bool
}

// Recognizer opens recognition streams; one per audio track.
type Recognizer interface {
	Stream(ctx context.Context, cfg Config) (Stream, error)
}

// Stream is one recognition. Send and Recv may run in different goroutines. After CloseSend, Recv returns
// the remaining results, then io.EOF; it returns the context's error once the context is done.
type Stream interface {
	Send(pcm []byte) error
	CloseSend() error
	Recv() (Result, error)
}
