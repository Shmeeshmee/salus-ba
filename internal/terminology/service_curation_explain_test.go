package terminology_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// The drafted explanations (SAL-13) are imported as unreviewed, without a reviewer, for the terms with their
// Wikidata ids; an explanation already there (perhaps the reviewer's) is kept as it is, so a second import changes
// nothing. A row naming no term, a language that is no app language (Latin among them) or a text outside 1 to 500
// characters is refused whole, the reason naming the column, never the text; the other rows are applied.
func TestImportExplanations_AddsTheMissingOnesAsUnreviewedAndKeepsTheOthers(t *testing.T) {
	t.Parallel()
	f := newFakeCuration(terminology.Term{ID: "t-liver", WikidataID: "Q9368"}, terminology.Term{ID: "t-ulcer", WikidataID: "Q1065994"})
	f.explanations = map[string]terminology.Explanation{
		"t-liver bs": {TermID: "t-liver", Language: "bs", Text: "Reviewed placeholder.", Review: terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: "a-reviewer"}, VersionID: 3},
	}
	report, err := newCurator(f).ImportExplanations(t.Context(), []terminology.ExplanationRow{
		{Line: 1, WikidataID: "Q9368", Texts: map[string]string{"en": "  Placeholder: an organ.  ", "bs": "Placeholder: organ."}},
		{Line: 2, WikidataID: "Q1", Texts: map[string]string{"en": "Placeholder."}},
		{Line: 3, WikidataID: "Q1065994", Texts: map[string]string{"la": "Placeholder."}},
		{Line: 4, WikidataID: "Q1065994", Texts: map[string]string{"en": strings.Repeat("x", 501)}},
		{Line: 5, WikidataID: "Q1065994", Texts: map[string]string{"de": "Placeholder: ein Geschwür."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Created != 2 || report.Unchanged != 1 || report.Updated != 0 {
		t.Errorf("report %+v, want created 2 (liver en, ulcer de), unchanged 1 (liver bs)", report)
	}
	want := map[int]string{2: "wikidata: no such term", 3: "explanations: la", 4: "explanations: en"}
	if got := reasons(report); !reflect.DeepEqual(got, want) {
		t.Errorf("refused %v, want %v", got, want)
	}
	en := f.explanations["t-liver en"]
	wantEN := terminology.Explanation{TermID: "t-liver", Language: "en", Text: "Placeholder: an organ.", UpdatedAt: now, VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
	if !reflect.DeepEqual(en, wantEN) {
		t.Errorf("the liver's English explanation = %+v, want %+v", en, wantEN)
	}
	if bs := f.explanations["t-liver bs"]; bs.Text != "Reviewed placeholder." || bs.VersionID != 3 {
		t.Errorf("the reviewed explanation was changed: %+v", bs)
	}
	again, err := newCurator(f).ImportExplanations(t.Context(), []terminology.ExplanationRow{{Line: 1, WikidataID: "Q9368", Texts: map[string]string{"en": "Other placeholder."}}})
	if err != nil || again.Created != 0 || again.Unchanged != 1 || f.explanations["t-liver en"].Text != "Placeholder: an organ." {
		t.Errorf("a second import = %+v, %v; want nothing changed", again, err)
	}
}

// The explanations file is one term per line: its Wikidata id and its texts by language. A line that is not one is
// refused; blank lines are skipped.
func TestDecodeExplanations_ReadsOneTermPerLine(t *testing.T) {
	t.Parallel()
	rows, refused, err := terminology.DecodeExplanations(strings.NewReader(
		`{"wikidata":"Q9368","explanations":{"en":"Placeholder: an organ.","bs":"Placeholder: organ."}}` + "\n\n" +
			`{"wikidata":"Q1","labels":{}}` + "\n" + `not json` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []terminology.ExplanationRow{{Line: 1, WikidataID: "Q9368", Texts: map[string]string{"en": "Placeholder: an organ.", "bs": "Placeholder: organ."}}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows %+v, want %+v", rows, want)
	}
	if len(refused) != 2 || refused[0].Line != 3 || refused[1].Line != 4 || refused[0].Reason != "not an explanations row" {
		t.Errorf("refused %+v, want lines 3 and 4", refused)
	}
}

// The committed drafts (assets/terminology/explanations.draft.jsonl) import whole against the committed seed: every
// line names a seeded term and has a valid text in each of the ten app languages.
func TestImportExplanations_TheCommittedDraftsImportWhole(t *testing.T) {
	t.Parallel()
	seedFile, err := os.Open("../../assets/terminology/seed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = seedFile.Close() }()
	seeded, _, err := terminology.DecodeSeed(seedFile)
	if err != nil {
		t.Fatal(err)
	}
	f := newFakeCuration()
	for i, r := range seeded {
		r.Term.ID = fmt.Sprintf("t-%d", i)
		f.terms[r.Term.ID] = r.Term
	}
	drafts, err := os.Open("../../assets/terminology/explanations.draft.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = drafts.Close() }()
	rows, refused, err := terminology.DecodeExplanations(drafts)
	if err != nil || len(refused) > 0 {
		t.Fatalf("decoding the drafts: %v, refused %+v", err, refused)
	}
	report, err := newCurator(f).ImportExplanations(t.Context(), rows)
	if err != nil || len(report.Refused) > 0 || report.Created != 10*len(rows) {
		t.Errorf("importing the drafts = %+v, %v; want every row's ten explanations created", report, err)
	}
}
