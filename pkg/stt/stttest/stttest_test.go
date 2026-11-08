package stttest_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
)

var _ stt.Recognizer = (*stttest.Recognizer)(nil)

func script() []stt.Result {
	return []stt.Result{
		{Text: "placeholder", Lang: "bs"},
		{Text: "placeholder words", Lang: "bs", Final: true, Start: 0, End: 1200 * time.Millisecond},
	}
}

func TestStream_RecvGivesTheScriptThenEOFOnceTheAudioIsClosed(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Results: script()}
	s, err := r.Stream(t.Context(), stt.Config{Lang: "bs", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range script() {
		got, err := s.Recv()
		if err != nil || got != want {
			t.Fatalf("Recv() #%d = %+v, %v; want %+v", i, got, err, want)
		}
	}
	if err := s.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recv(); !errors.Is(err, io.EOF) {
		t.Errorf("Recv() after the script and CloseSend: %v, want io.EOF", err)
	}
}

func TestStream_RecvWaitsForTheAudioToCloseAfterTheScript(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{}
	s, err := r.Stream(t.Context(), stt.Config{Lang: "bs", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.Recv()
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("Recv() returned %v before CloseSend", err)
	case <-time.After(20 * time.Millisecond):
	}
	if err := s.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Errorf("Recv() = %v, want io.EOF", err)
	}
}

func TestStream_RecvEndsWithTheContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	s, err := (&stttest.Recognizer{}).Stream(ctx, stt.Config{Lang: "bs", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := s.Recv(); !errors.Is(err, context.Canceled) {
		t.Errorf("Recv() = %v, want context.Canceled", err)
	}
}

func TestStream_RecordsTheAudioAndTheConfigAndRefusesAudioAfterClose(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{}
	cfg := stt.Config{Lang: "hr", SampleRate: 48000}
	s, err := r.Stream(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, frame := range [][]byte{{1, 2}, {3, 4}} {
		if err := s.Send(frame); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if err := s.Send([]byte{5}); !errors.Is(err, stt.ErrClosed) {
		t.Errorf("Send() after CloseSend = %v, want stt.ErrClosed", err)
	}
	if err := s.CloseSend(); !errors.Is(err, stt.ErrClosed) {
		t.Errorf("CloseSend() twice = %v, want stt.ErrClosed", err)
	}

	streams := r.Streams()
	if len(streams) != 1 {
		t.Fatalf("%d streams, want 1", len(streams))
	}
	if streams[0].Config != cfg || string(streams[0].Audio()) != string([]byte{1, 2, 3, 4}) {
		t.Errorf("stream = config %+v, audio %v; want %+v and [1 2 3 4]", streams[0].Config, streams[0].Audio(), cfg)
	}
}

func TestRecognizer_EachStreamGetsTheWholeScript(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Results: script()}
	for i := range 2 {
		s, err := r.Stream(t.Context(), stt.Config{Lang: "bs", SampleRate: 16000})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := s.Recv(); err != nil || got != script()[0] {
			t.Errorf("stream %d: first Recv() = %+v, %v", i, got, err)
		}
	}
}

func TestRecognizer_ReturnsItsError(t *testing.T) {
	t.Parallel()
	want := errors.New("engine down")
	if _, err := (&stttest.Recognizer{Err: want}).Stream(t.Context(), stt.Config{}); !errors.Is(err, want) {
		t.Errorf("Stream() = %v, want %v", err, want)
	}
}
