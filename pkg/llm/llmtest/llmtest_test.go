package llmtest_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
)

var (
	_ llm.Translator = (*llmtest.Translator)(nil)
	_ llm.Suggester  = (*llmtest.Suggester)(nil)
	_ llm.Drafter    = (*llmtest.Drafter)(nil)
)

var errDown = errors.New("model down")

func transcript() []llm.Utterance {
	return []llm.Utterance{
		{ID: "s1", Speaker: "patient", Text: "placeholder complaint"},
		{ID: "s2", Speaker: "practitioner", Text: "placeholder question"},
	}
}

func TestTranslator_MarksTheTextWithTheTargetLanguageAndRecordsTheCall(t *testing.T) {
	t.Parallel()
	tr := &llmtest.Translator{}
	req := llm.TranslateRequest{Text: "placeholder", From: "bs", To: "de"}
	got, err := tr.Translate(t.Context(), req)
	if err != nil || got != "[de] placeholder" {
		t.Errorf("Translate() = %q, %v; want %q", got, err, "[de] placeholder")
	}
	if calls := tr.Calls(); !slices.Equal(calls, []llm.TranslateRequest{req}) {
		t.Errorf("Calls() = %+v", calls)
	}
}

func TestTranslator_ReturnsItsError(t *testing.T) {
	t.Parallel()
	if _, err := (&llmtest.Translator{Err: errDown}).Translate(t.Context(), llm.TranslateRequest{}); !errors.Is(err, errDown) {
		t.Errorf("Translate() = %v, want %v", err, errDown)
	}
}

func TestTranslator_IsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	tr := &llmtest.Translator{}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _, _ = tr.Translate(t.Context(), llm.TranslateRequest{Text: "x", To: "bs"}) })
	}
	wg.Wait()
	if n := len(tr.Calls()); n != 8 {
		t.Errorf("%d calls, want 8", n)
	}
}

func TestSuggester_ReturnsACopyOfItsSuggestionsAndRecordsTheCall(t *testing.T) {
	t.Parallel()
	want := []llm.Suggestion{{Kind: "question", Text: "placeholder question", Confidence: 0.5, SourceIDs: []string{"s1"}}}
	s := &llmtest.Suggester{Suggestions: want}
	got, err := s.Suggest(t.Context(), llm.SuggestRequest{Lang: "bs", Transcript: transcript()})
	if err != nil || len(got) != 1 || got[0].Text != want[0].Text {
		t.Fatalf("Suggest() = %+v, %v; want %+v", got, err, want)
	}
	got[0].SourceIDs[0] = "changed"
	if s.Suggestions[0].SourceIDs[0] != "s1" {
		t.Error("the caller's change reached the fake's script")
	}
	if calls := s.Calls(); len(calls) != 1 || calls[0].Lang != "bs" || len(calls[0].Transcript) != 2 {
		t.Errorf("Calls() = %+v", calls)
	}
}

func TestSuggester_ReturnsItsError(t *testing.T) {
	t.Parallel()
	if _, err := (&llmtest.Suggester{Err: errDown}).Suggest(t.Context(), llm.SuggestRequest{}); !errors.Is(err, errDown) {
		t.Errorf("Suggest() = %v, want %v", err, errDown)
	}
}

func TestDrafter_FillsTheFieldsItKnowsAndMarksTheRestMissing(t *testing.T) {
	t.Parallel()
	d := &llmtest.Drafter{Fields: map[string]llm.DraftField{
		"complaint": {Value: "placeholder complaint", SourceIDs: []string{"s1"}},
	}}
	got, err := d.Draft(t.Context(), llm.DraftRequest{
		Lang:       "bs",
		Fields:     []llm.FieldSpec{{Key: "complaint", Label: "Placeholder label"}, {Key: "plan", Label: "Placeholder label"}},
		Transcript: transcript(),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []llm.DraftField{
		{Key: "complaint", Value: "placeholder complaint", SourceIDs: []string{"s1"}},
		{Key: "plan", Missing: true},
	}
	if !slices.EqualFunc(got, want, func(a, b llm.DraftField) bool {
		return a.Key == b.Key && a.Value == b.Value && a.Missing == b.Missing && slices.Equal(a.SourceIDs, b.SourceIDs)
	}) {
		t.Errorf("Draft() = %+v, want %+v", got, want)
	}
	got[0].SourceIDs[0] = "changed"
	if d.Fields["complaint"].SourceIDs[0] != "s1" {
		t.Error("the caller's change reached the fake's script")
	}
	if calls := d.Calls(); len(calls) != 1 || len(calls[0].Fields) != 2 {
		t.Errorf("Calls() = %+v", calls)
	}
}

func TestDrafter_ReturnsItsError(t *testing.T) {
	t.Parallel()
	if _, err := (&llmtest.Drafter{Err: errDown}).Draft(t.Context(), llm.DraftRequest{}); !errors.Is(err, errDown) {
		t.Errorf("Draft() = %v, want %v", err, errDown)
	}
}

// A field's type, instruction and required flag reach the fake as the caller sent them, and the recorded
// request is the caller's own copy: a later change to the caller's slice does not alter it.
func TestDrafter_RecordsEachFieldsTypeInstructionAndRequiredFlag(t *testing.T) {
	t.Parallel()
	d := &llmtest.Drafter{}
	fields := []llm.FieldSpec{
		{Key: "findings", Label: "Placeholder label", Type: llm.FieldText, Instruction: "Placeholder instruction", Required: true},
		{Key: "next", Label: "Placeholder label", Type: llm.FieldDate},
	}
	if _, err := d.Draft(t.Context(), llm.DraftRequest{Lang: "bs", Fields: fields, Transcript: transcript()}); err != nil {
		t.Fatal(err)
	}
	want := slices.Clone(fields)
	fields[0].Instruction = "changed"
	if calls := d.Calls(); len(calls) != 1 || !slices.Equal(calls[0].Fields, want) {
		t.Errorf("Calls() = %+v, want the fields %+v", calls, want)
	}
}

// The field types are the four a template knows.
func TestFieldTypes_AreTextListDateAndCoded(t *testing.T) {
	t.Parallel()
	got := []llm.FieldType{llm.FieldText, llm.FieldList, llm.FieldDate, llm.FieldCoded}
	if !slices.Equal(got, []llm.FieldType{"text", "list", "date", "coded"}) {
		t.Errorf("field types = %q", got)
	}
}
