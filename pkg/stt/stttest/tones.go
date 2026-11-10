package stttest

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/stt"
)

// Line is a scripted line: what a stream in its language (a BCP 47 language, "de" for a stream in "de-DE") says
// while it hears a pure tone of Tone Hz. The tone stands for the line's speech, so a journey plays a conversation
// through the real audio path with generated tones instead of anyone's voice.
type Line struct {
	Lang string
	Text string
	Tone int
}

// The tones a line may have: on a 50 Hz grid, so each is a whole bin of a 20 ms block at 16 kHz, between
// MinTone and MaxTone, above the 440 Hz test tone and a browser's 400 Hz fake beep and below G.711's 4 kHz.
const (
	MinTone  = 500
	MaxTone  = 3500
	toneGrid = 50
)

// LoadScript reads the lines of every .json file in fsys: a scripted conversation each, whose "lines" hold a
// "tone", a "lang" and a "text" (other keys, read by llmtest.LoadScript and the journeys, are ignored here). A
// tone is used once per language.
func LoadScript(fsys fs.FS) ([]Line, error) {
	names, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, err
	}
	var lines []Line
	seen := map[[2]string]bool{}
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		var file struct{ Lines []Line }
		if err := json.Unmarshal(b, &file); err != nil {
			return nil, fmt.Errorf("stttest: %s: %w", name, err)
		}
		for _, l := range file.Lines {
			key := [2]string{l.Lang, fmt.Sprint(l.Tone)}
			switch {
			case strings.TrimSpace(l.Text) == "" || l.Lang == "":
				return nil, fmt.Errorf("stttest: %s: a line without its text or language", name)
			case l.Tone < MinTone || l.Tone > MaxTone || l.Tone%toneGrid != 0:
				return nil, fmt.Errorf("stttest: %s: tone %d is not a multiple of %d Hz from %d to %d", name, l.Tone, toneGrid, MinTone, MaxTone)
			case seen[key]:
				return nil, fmt.Errorf("stttest: %s: tone %d is used twice in %s", name, l.Tone, l.Lang)
			}
			seen[key] = true
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return nil, errors.New("stttest: the script has no line")
	}
	return lines, nil
}

// How a tone is heard: a 20 ms block is a line's tone when at least toneShare of its energy is in the tone's bin
// and it is louder than silence; a line starts after onBlocks such blocks in a row (a 20 ms beep is no line), and
// ends after offBlocks blocks without its tone.
const (
	blocksPerSecond = 50
	toneShare       = 0.5
	silence         = 100 * 100 // the mean square of a quiet block, of 16-bit samples
	onBlocks        = 3
	offBlocks       = 3
	resultsBuffer   = 64
)

// ear follows one stream's audio, block by block, for the tones of the lines in its language.
type ear struct {
	lines   []Line
	pending []byte // the bytes of a block not complete yet
	block   []float64
	rate    int
	n       int // blocks heard
	run     int // blocks in a row with the tone of candidate
	cand    int // the line whose tone the last block held, or -1
	current int // the line being said, or -1
	since   int // the block the current line started in
	last    int // the last block the current line's tone was heard in
}

// newEar listens for the lines in the stream's language, or is nil when there are none.
func newEar(lines []Line, cfg stt.Config) *ear {
	var mine []Line
	for _, l := range lines {
		if cfg.Lang == l.Lang || strings.HasPrefix(cfg.Lang, l.Lang+"-") {
			mine = append(mine, l)
		}
	}
	if len(mine) == 0 || cfg.SampleRate < 2*MaxTone {
		return nil
	}
	return &ear{lines: mine, rate: cfg.SampleRate, block: make([]float64, 0, cfg.SampleRate/blocksPerSecond), cand: -1, current: -1}
}

// hear takes audio (16-bit little-endian samples) and says the partial and final results of the lines it holds.
func (e *ear) hear(pcm []byte, say func(stt.Result)) {
	e.pending = append(e.pending, pcm...)
	size := cap(e.block)
	for len(e.pending) >= 2 {
		e.block = append(e.block, float64(int16(binary.LittleEndian.Uint16(e.pending))))
		e.pending = e.pending[2:]
		if len(e.block) == size {
			e.heard(e.tone(), say)
			e.block = e.block[:0]
		}
	}
	e.pending = append([]byte(nil), e.pending...) // keep the odd byte, not the caller's array
}

// heard moves on by one block that held the tone of line (or -1).
func (e *ear) heard(line int, say func(stt.Result)) {
	if e.current >= 0 {
		if line == e.current {
			e.last = e.n
		} else if e.n-e.last >= offBlocks {
			e.end(say)
		}
	}
	if line == e.cand {
		e.run++
	} else {
		e.cand, e.run = line, 1
	}
	if e.current < 0 && e.cand >= 0 && e.run >= onBlocks {
		e.current, e.since, e.last = e.cand, e.n-e.run+1, e.n
		say(stt.Result{Text: firstHalf(e.lines[e.current].Text), Start: e.at(e.since), End: e.at(e.n + 1)})
	}
	e.n++
}

// end says the current line's final result, if a line is being said.
func (e *ear) end(say func(stt.Result)) {
	if e.current < 0 {
		return
	}
	say(stt.Result{Text: e.lines[e.current].Text, Start: e.at(e.since), End: e.at(e.last + 1), Final: true})
	e.current = -1
}

// at is the offset of the start of block n.
func (e *ear) at(n int) time.Duration {
	return time.Duration(n) * time.Second / blocksPerSecond
}

// tone is the line whose tone the block holds, or -1: the strongest of their bins (Goertzel), when it holds at
// least toneShare of the block's energy.
func (e *ear) tone() int {
	var energy float64
	for _, v := range e.block {
		energy += v * v
	}
	n := float64(len(e.block))
	if energy/n < silence {
		return -1
	}
	best, share := -1, toneShare
	for i, l := range e.lines {
		// A pure tone in the bin has power (A·n/2)² and energy A²·n/2: their ratio, times 2/n, is 1.
		if s := 2 * goertzel(e.block, float64(l.Tone), float64(e.rate)) / (n * energy); s >= share {
			best, share = i, s
		}
	}
	return best
}

// goertzel is the power of x at freq Hz.
func goertzel(x []float64, freq, rate float64) float64 {
	c := 2 * math.Cos(2*math.Pi*freq/rate)
	var s1, s2 float64
	for _, v := range x {
		s1, s2 = v+c*s1-s2, s1
	}
	return s1*s1 + s2*s2 - c*s1*s2
}

// firstHalf is the first half of the words of text, at least one: what a recognizer has heard of a line early.
func firstHalf(text string) string {
	words := strings.Fields(text)
	return strings.Join(words[:(len(words)+1)/2], " ")
}
