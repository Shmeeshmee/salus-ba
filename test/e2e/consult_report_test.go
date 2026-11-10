//go:build e2e

package e2e_test

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The latency of the consultation pipeline's journeys against the plan's targets (docs/plans/consult-intelligence.md,
// SAL-10: caption under 1 s, translation under 2 s, draft under 15 s), reported once per run: printed, and written
// to E2E_LATENCY_REPORT (make e2e prints it; CI puts it in the job summary and keeps it as an artifact).

// The hops measured, and their budgets.
const (
	hopCaption     = "caption"
	hopTranslation = "translation"
	hopDraft       = "draft"
)

var budgets = map[string]time.Duration{hopCaption: time.Second, hopTranslation: 2 * time.Second, hopDraft: 15 * time.Second}

// What each hop measures, for the report.
var hopMeaning = map[string]string{
	hopCaption: "the first frame of a line's tone sent by the speaker's LiveKit participant, to its partial " +
		"transcript event received on the practitioner's event stream (WebSocket)",
	hopTranslation: "the last frame of a line's tone, to its translated final event on the practitioner's event stream",
	hopDraft: "the practitioner's Generate (a click in the web app, or POST /encounters/{id}/drafts) to the drafted " +
		"fields shown, or the answer",
}

// latencies is the run's samples, by journey and hop.
var latencies = &latencyLog{samples: map[[2]string][]time.Duration{}}

type latencyLog struct {
	samples map[[2]string][]time.Duration
	order   [][2]string
	mu      sync.Mutex
}

func (l *latencyLog) add(journey, hop string, d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := [2]string{journey, hop}
	if _, ok := l.samples[key]; !ok {
		l.order = append(l.order, key)
	}
	l.samples[key] = append(l.samples[key], d)
}

// check fails the journey when the median of a hop is over its budget: a clear breach, not one slow sample on a
// busy runner.
func (l *latencyLog) check(t *testing.T, journey string) {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range l.order {
		if key[0] != journey {
			continue
		}
		if p50 := pct(l.samples[key], 50); p50 > budgets[key[1]] {
			t.Errorf("%s: the %s's median latency %v is over its budget of %v", journey, key[1], p50, budgets[key[1]])
		}
	}
}

// report is the run's latency as Markdown.
func (l *latencyLog) report() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var b strings.Builder
	b.WriteString("## Consultation pipeline latency (make e2e)\n\n")
	b.WriteString("On the fake speech engine and the fake model: these numbers measure Salus's own pipeline (LiveKit, " +
		"the transcription agent, the Redis stream, the api's WebSocket; for the draft the api, Postgres and the web app), " +
		"not a speech engine or a language model, whose time comes on top once a real one is enabled. The fake engine " +
		"starts a line after 60 ms of its tone and ends it after 60 ms without it; both are in the numbers.\n\n")
	if len(l.order) == 0 {
		b.WriteString("No consultation pipeline journey ran.\n")
		return b.String()
	}
	b.WriteString("| Journey | Hop | n | p50 | max | Budget | Median |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, key := range l.order {
		d := l.samples[key]
		verdict := "within"
		if pct(d, 50) > budgets[key[1]] {
			verdict = "over"
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %v | %v | %v | %s |\n", key[0], key[1], len(d),
			pct(d, 50).Round(time.Millisecond), pct(d, 100).Round(time.Millisecond), budgets[key[1]], verdict)
	}
	b.WriteString("\n")
	for _, hop := range []string{hopCaption, hopTranslation, hopDraft} {
		fmt.Fprintf(&b, "- **%s**: %s.\n", hop, hopMeaning[hop])
	}
	return b.String()
}

// TestMain runs the journeys, then prints the latency report and writes it to E2E_LATENCY_REPORT when set.
func TestMain(m *testing.M) {
	code := m.Run()
	report := latencies.report()
	fmt.Print("\n" + report)
	if path := os.Getenv("E2E_LATENCY_REPORT"); path != "" {
		if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "write the latency report:", err)
			code = max(code, 1)
		}
	}
	os.Exit(code)
}

// pct is the p-th percentile of d (nearest rank), rounded to 10 µs.
func pct(d []time.Duration, p int) time.Duration {
	s := slices.Clone(d)
	slices.Sort(s)
	i := max(0, (len(s)*p+99)/100-1)
	return s[i].Round(10 * time.Microsecond)
}
