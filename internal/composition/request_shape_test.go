//go:build loadtest

package composition_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// The shape of what a drafting sends the model (SAL-8, docs/plans/consult-intelligence.PROGRESS.md): a
// 30-minute consultation of 360 segments of about 130 characters and the seeded findings report's seven fields,
// as the DraftRequest the Drafter receives, measured as JSON. Not part of make test.
func TestDraftRequestShape(t *testing.T) {
	f := newFixture(t)
	f.transcripts.segments = nil
	for i := range 360 {
		f.transcripts.segments = append(f.transcripts.segments, transcript.Event{
			EncounterID: encA, SegmentID: fmt.Sprintf("51000000-0000-4000-8000-%012d", i), Speaker: transcript.SpeakerPatient,
			Lang: "bs", Original: strings.Repeat("Placeholder words of a sentence ", 4), IsFinal: true, TsStart: now.Add(time.Duration(i) * 5 * time.Second),
		})
	}
	c := content()
	c.Translations = nil
	for _, k := range []string{"history", "therapy", "reason"} {
		c.Fields = append(c.Fields, composition.TemplateField{Key: k, Type: composition.FieldText, Label: "Placeholder label", Instruction: strings.Repeat("Placeholder instruction ", 3)})
	}
	tpl, err := f.svc.CreateTemplate(t.Context(), pract, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := generate(t, f, pract, tpl.ID, ""); err != nil {
		t.Fatal(err)
	}
	req := f.drafter.Calls()[0]
	b, _ := json.Marshal(req)
	text := 0
	for _, u := range req.Transcript {
		text += len(u.Text)
	}
	t.Logf("DraftRequest: %d utterances, %d fields, %d bytes as JSON, %d characters of transcript text", len(req.Transcript), len(req.Fields), len(b), text)
}
