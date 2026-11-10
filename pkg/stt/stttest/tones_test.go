package stttest_test

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
)

const rate = 16000

// pcm is d of a sine of freq Hz (silence for 0) as 16-bit little-endian samples at rate.
func pcm(freq float64, d time.Duration) []byte {
	n := int(d.Seconds() * rate)
	out := make([]byte, 2*n)
	for i := range n {
		v := 0.0
		if freq > 0 {
			v = 8000 * math.Sin(2*math.Pi*freq*float64(i)/rate)
		}
		binary.LittleEndian.PutUint16(out[2*i:], uint16(int16(v)))
	}
	return out
}

// sendAll sends audio in chunks of size bytes, as a decoder hands over frames of its own size.
func sendAll(t *testing.T, s stt.Stream, audio []byte, size int) {
	t.Helper()
	for len(audio) > 0 {
		n := min(size, len(audio))
		if err := s.Send(audio[:n]); err != nil {
			t.Fatal(err)
		}
		audio = audio[n:]
	}
}

// results closes the audio and returns every result the stream gives before io.EOF.
func results(t *testing.T, s stt.Stream) []stt.Result {
	t.Helper()
	if err := s.CloseSend(); err != nil {
		t.Fatal(err)
	}
	var got []stt.Result
	for {
		r, err := s.Recv()
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
}

func lines() []stttest.Line {
	return []stttest.Line{
		{Tone: 650, Lang: "de", Text: "Platzhalter eins zwei drei"},
		{Tone: 1050, Lang: "de", Text: "Platzhalter vier"},
		{Tone: 1450, Lang: "bs", Text: "Primjer pet"},
	}
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func TestStream_SaysTheLineOfAToneInTheStreamsLanguage(t *testing.T) {
	t.Parallel()
	for _, size := range []int{640, 333, 4096} { // 20 ms frames, and chunks of any other size
		r := &stttest.Recognizer{Lines: lines(), DiscardAudio: true}
		s, err := r.Stream(t.Context(), stt.Config{Lang: "de-DE", SampleRate: rate})
		if err != nil {
			t.Fatal(err)
		}
		sendAll(t, s, cat(pcm(0, 200*time.Millisecond), pcm(650, 600*time.Millisecond), pcm(0, 200*time.Millisecond)), size)
		want := []stt.Result{
			{Text: "Platzhalter eins", Start: 200 * time.Millisecond, End: 260 * time.Millisecond},
			{Text: "Platzhalter eins zwei drei", Start: 200 * time.Millisecond, End: 800 * time.Millisecond, Final: true},
		}
		if got := results(t, s); !equal(got, want) {
			t.Errorf("chunks of %d bytes: results %+v, want %+v", size, got, want)
		}
	}
}

func TestStream_SaysOneLineAfterAnotherWithoutAPause(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Lines: lines()}
	s, _ := r.Stream(t.Context(), stt.Config{Lang: "de-DE", SampleRate: rate})
	sendAll(t, s, cat(pcm(650, 400*time.Millisecond), pcm(1050, 400*time.Millisecond), pcm(0, 100*time.Millisecond)), 640)
	want := []stt.Result{
		{Text: "Platzhalter eins", Start: 0, End: 60 * time.Millisecond},
		{Text: "Platzhalter eins zwei drei", Start: 0, End: 400 * time.Millisecond, Final: true},
		{Text: "Platzhalter", Start: 400 * time.Millisecond, End: 460 * time.Millisecond},
		{Text: "Platzhalter vier", Start: 400 * time.Millisecond, End: 800 * time.Millisecond, Final: true},
	}
	if got := results(t, s); !equal(got, want) {
		t.Errorf("results %+v, want %+v", got, want)
	}
}

func TestStream_EndsTheLineWhenTheAudioClosesDuringItsTone(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Lines: lines()}
	s, _ := r.Stream(t.Context(), stt.Config{Lang: "bs-BA", SampleRate: rate})
	sendAll(t, s, pcm(1450, 300*time.Millisecond), 640)
	want := []stt.Result{
		{Text: "Primjer", Start: 0, End: 60 * time.Millisecond},
		{Text: "Primjer pet", Start: 0, End: 300 * time.Millisecond, Final: true},
	}
	if got := results(t, s); !equal(got, want) {
		t.Errorf("results %+v, want %+v", got, want)
	}
}

func TestStream_HearsNoLineInSilenceOtherTonesShortBeepsOrAnotherLanguage(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		lang  string
		audio []byte
	}{
		"silence":                    {"de-DE", pcm(0, time.Second)},
		"the default test tone":      {"de-DE", pcm(440, time.Second)},
		"a browser's fake beep":      {"de-DE", cat(pcm(400, time.Second))},
		"a 40 ms burst of the tone":  {"de-DE", cat(pcm(0, 100*time.Millisecond), pcm(650, 40*time.Millisecond), pcm(0, 200*time.Millisecond))},
		"a line of another language": {"bs-BA", pcm(650, time.Second)},
	} {
		r := &stttest.Recognizer{Lines: lines()}
		s, _ := r.Stream(t.Context(), stt.Config{Lang: tc.lang, SampleRate: rate})
		sendAll(t, s, tc.audio, 640)
		if got := results(t, s); len(got) != 0 {
			t.Errorf("%s: results %+v, want none", name, got)
		}
	}
}

func TestStream_PlaysItsResultsBeforeTheLines(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Results: script(), Lines: lines()}
	s, _ := r.Stream(t.Context(), stt.Config{Lang: "bs-BA", SampleRate: rate})
	sendAll(t, s, pcm(1450, 200*time.Millisecond), 640)
	got := results(t, s)
	if len(got) != 4 || !equal(got[:2], script()) || got[3].Text != "Primjer pet" {
		t.Errorf("results %+v, want the script, then the line", got)
	}
}

func TestStream_RecvWaitsForALine(t *testing.T) {
	t.Parallel()
	r := &stttest.Recognizer{Lines: lines()}
	s, _ := r.Stream(t.Context(), stt.Config{Lang: "de-DE", SampleRate: rate})
	got := make(chan stt.Result, 1)
	go func() {
		res, _ := s.Recv()
		got <- res
	}()
	sendAll(t, s, pcm(650, 100*time.Millisecond), 640)
	select {
	case res := <-got:
		if res.Text != "Platzhalter eins" || res.Final {
			t.Errorf("Recv() = %+v, want the line's partial", res)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Recv() did not return the line's partial")
	}
}

func equal(a, b []stt.Result) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoadScript_ReadsTheLinesOfEveryFile(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"de.json":   {Data: []byte(`{"about": "synthetic", "lines": [{"speaker": "patient", "tone": 650, "lang": "de", "text": "Platzhalter", "translations": {"bs": "Primjer"}}]}`)},
		"tr.json":   {Data: []byte(`{"lines": [{"tone": 700, "lang": "tr", "text": "Yer tutucu"}], "suggestions": []}`)},
		"notes.txt": {Data: []byte("not a script")},
	}
	got, err := stttest.LoadScript(fsys)
	want := []stttest.Line{{Tone: 650, Lang: "de", Text: "Platzhalter"}, {Tone: 700, Lang: "tr", Text: "Yer tutucu"}}
	if err != nil || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("LoadScript() = %+v, %v; want %+v", got, err, want)
	}
}

func TestLoadScript_RefusesAScriptItCannotPlay(t *testing.T) {
	t.Parallel()
	for name, files := range map[string]fstest.MapFS{
		"no file":                   {},
		"not JSON":                  {"de.json": {Data: []byte("{")}},
		"no line":                   {"de.json": {Data: []byte(`{"lines": []}`)}},
		"no text":                   {"de.json": {Data: []byte(`{"lines": [{"tone": 650, "lang": "de", "text": " "}]}`)}},
		"no language":               {"de.json": {Data: []byte(`{"lines": [{"tone": 650, "text": "Platzhalter"}]}`)}},
		"a tone too low":            {"de.json": {Data: []byte(`{"lines": [{"tone": 440, "lang": "de", "text": "Platzhalter"}]}`)}},
		"a tone too high":           {"de.json": {Data: []byte(`{"lines": [{"tone": 3700, "lang": "de", "text": "Platzhalter"}]}`)}},
		"a tone off the 50 Hz grid": {"de.json": {Data: []byte(`{"lines": [{"tone": 655, "lang": "de", "text": "Platzhalter"}]}`)}},
		"one tone twice in a language": {
			"de.json": {Data: []byte(`{"lines": [{"tone": 650, "lang": "de", "text": "Eins"}]}`)},
			"at.json": {Data: []byte(`{"lines": [{"tone": 650, "lang": "de", "text": "Zwei"}]}`)},
		},
	} {
		if got, err := stttest.LoadScript(files); err == nil {
			t.Errorf("%s: LoadScript() = %+v, want an error", name, got)
		}
	}
}

// The end-to-end journeys' conversations (make e2e's FAKE_ENGINE_SCRIPT) load, so a broken fixture fails here
// first.
func TestLoadScript_ReadsTheJourneysConversations(t *testing.T) {
	t.Parallel()
	lines, err := stttest.LoadScript(os.DirFS(filepath.Join("..", "..", "..", "test", "testdata", "consult")))
	if err != nil || len(lines) == 0 {
		t.Errorf("LoadScript(test/testdata/consult) = %d lines, %v", len(lines), err)
	}
}
