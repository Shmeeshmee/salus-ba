package composition_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// fakeTagger finds the spans it holds for a text's body, recording the texts it was asked.
type fakeTagger struct {
	err   error
	spans map[string][]terminology.Span
	asked []terminology.Text
}

func (f *fakeTagger) Tag(_ context.Context, texts []terminology.Text) ([][]terminology.Span, error) {
	f.asked = append(f.asked, texts...)
	out := make([][]terminology.Span, len(texts))
	for i, t := range texts {
		out[i] = f.spans[t.Body]
	}
	return out, f.err
}

// The explained terms of a signed document (SAL-13), for the tooltip on the patient's and the practitioner's
// screens: per field in the document's order, per line of its value as the Composition's narrative has them (none
// for a missing field), tagged in the document's language. Read under the Composition's rule and audited as its
// read; the document is not changed.
func TestTerms_GivesTheExplainedTermsOfEachFieldLineByLine(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tagger := &fakeTagger{spans: map[string][]terminology.Span{"Placeholder findings": {{TermID: "t-1", Start: 0, End: 11}}}}
	f.svc.Tagger = tagger
	vals := values()
	vals[1].Value = "Placeholder findings\nPlaceholder second line"
	signed, err := f.svc.Sign(t.Context(), pract, f.drafted(t).ID, vals, 1)
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil

	got, err := f.svc.Terms(t.Context(), me, signed.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []composition.SectionTerms{
		{Key: "findings", Lines: [][]terminology.Span{{{TermID: "t-1", Start: 0, End: 11}}, nil}},
		{Key: "tests", Lines: [][]terminology.Span{nil}},
		{Key: "control", Lines: [][]terminology.Span{nil}},
		{Key: "diagnosis", Lines: [][]terminology.Span{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Terms() = %+v,\nwant %+v", got, want)
	}
	for _, text := range tagger.asked {
		if text.Language != signed.Language {
			t.Errorf("tagged %+v, want the document's language %q", text, signed.Language)
		}
	}
	if got, want := f.codes(), []string{"read 0 Composition/" + signed.ID}; !slices.Equal(got, want) {
		t.Errorf("audit = %q, want %q", got, want)
	}
}

// A draft is never the patient's, so neither are its terms; a failed tagging is the read's failure.
func TestTerms_FollowsTheCompositionsRule(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.svc.Tagger = &fakeTagger{}
	d := f.drafted(t)
	if _, err := f.svc.Terms(t.Context(), me, d.ID); !errors.Is(err, composition.ErrNotFound) {
		t.Errorf("a draft's terms for the patient = %v, want ErrNotFound", err)
	}
	f.svc.Tagger = &fakeTagger{err: errors.New("the database is down")}
	if _, err := f.svc.Terms(t.Context(), pract, d.ID); err == nil {
		t.Errorf("Terms() with tagging down = nil, want the failure")
	}
}
