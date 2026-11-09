package ttstest_test

import (
	"errors"
	"io"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/tts"
	"github.com/Shmeeshmee/salus/pkg/tts/ttstest"
)

var _ tts.Synthesizer = (*ttstest.Synthesizer)(nil)

func TestSynthesizer_StreamsItsAudioAndRecordsTheCall(t *testing.T) {
	t.Parallel()
	s := &ttstest.Synthesizer{Audio: []byte{1, 2, 3}}
	req := tts.Request{Text: "placeholder", Lang: "bs", SampleRate: 24000}
	rc, err := s.Synthesize(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	if err != nil || string(got) != string([]byte{1, 2, 3}) {
		t.Errorf("audio = %v, %v; want [1 2 3]", got, err)
	}
	if err := rc.Close(); err != nil {
		t.Error(err)
	}
	if calls := s.Calls(); len(calls) != 1 || calls[0] != req {
		t.Errorf("Calls() = %+v, want [%+v]", calls, req)
	}
}

func TestSynthesizer_ReturnsItsError(t *testing.T) {
	t.Parallel()
	want := errors.New("engine down")
	if _, err := (&ttstest.Synthesizer{Err: want}).Synthesize(t.Context(), tts.Request{}); !errors.Is(err, want) {
		t.Errorf("Synthesize() = %v, want %v", err, want)
	}
}
