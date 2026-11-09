// Package tts is the port of a text-to-speech engine, for dubbing a translated segment into a consultation
// (docs/plans/consult-intelligence.md, v2). Only the port and its fake (ttstest) exist until v2.
package tts

import (
	"context"
	"io"
)

// Request is a text, its language (a BCP 47 tag), and the sample rate of the audio to return.
type Request struct {
	Text       string
	Lang       string
	SampleRate int
}

// Synthesizer speaks a text. The audio is 16-bit little-endian PCM, mono, streamed as it is synthesised; the
// caller closes it.
type Synthesizer interface {
	Synthesize(ctx context.Context, req Request) (io.ReadCloser, error)
}
