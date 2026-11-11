package terminology_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// flagFake holds term t1 (Q1) with never-reviewed translations in hr and tr, and term t2 (Q2) with an hr translation
// a reviewer approved.
func flagFake() *fakeCuration {
	f := newFakeCuration(
		terminology.Term{ID: "t1", WikidataID: "Q1", Labels: labels("hr", "placeholder hr", "tr", "placeholder tr", "tr", "wrong tr")},
		terminology.Term{ID: "t2", WikidataID: "Q2", Labels: labels("hr", "placeholder two")})
	approved := terminology.Review{Status: terminology.ReviewApproved, ReviewedBy: "acc-1", ReviewedAt: now.Add(-time.Hour)}
	f.stored = map[string]terminology.Translation{
		"t1 hr": {TermID: "t1", Language: "hr", VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}},
		"t1 tr": {TermID: "t1", Language: "tr", VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}},
		"t2 hr": {TermID: "t2", Language: "hr", VersionID: 3, Review: approved},
	}
	f.termReviews = map[string]terminology.Review{"t1": {Status: terminology.ReviewUnreviewed}, "t2": approved}
	return f
}

// A flag marks a translation never reviewed as needing changes, with its note, no reviewer and the import's time, at
// the next version; in a language the term's status follows (Latin and the staff languages), the term needs changes
// too, without a reviewer; in a patient-only language the term's status stays.
func TestImportFlags_MarksANeverReviewedTranslationAsNeedingChanges(t *testing.T) {
	f := flagFake()
	report, err := newCurator(f).ImportFlags(t.Context(), []terminology.FlagRow{
		{Line: 1, WikidataID: "Q1", Language: "tr", Label: "wrong tr", Note: " Placeholder: a wrong synonym. "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := (terminology.ImportReport{Updated: 1}); !reflect.DeepEqual(report, want) {
		t.Errorf("report = %+v, want %+v", report, want)
	}
	want := terminology.Translation{TermID: "t1", Language: "tr", VersionID: 2, UpdatedAt: now,
		Review: terminology.Review{Status: terminology.ReviewNeedsChanges, ReviewedAt: now, Note: "Placeholder: a wrong synonym."}}
	if got := f.stored["t1 tr"]; !reflect.DeepEqual(got, want) {
		t.Errorf("translation = %+v, want %+v", got, want)
	}
	if got := f.termReviews["t1"].Status; got != terminology.ReviewUnreviewed {
		t.Errorf("term status after a tr flag = %q, want unreviewed (tr does not move it)", got)
	}

	if _, err := newCurator(f).ImportFlags(t.Context(), []terminology.FlagRow{{Line: 1, WikidataID: "Q1", Language: "hr", Label: "placeholder hr", Note: "Placeholder."}}); err != nil {
		t.Fatal(err)
	}
	if want := (terminology.Review{Status: terminology.ReviewNeedsChanges, ReviewedAt: now}); f.termReviews["t1"] != want {
		t.Errorf("term review after an hr flag = %+v, want %+v", f.termReviews["t1"], want)
	}
}

// A translation that has been reviewed (by a reviewer, the curator's file or an earlier flag) is kept as it is, so
// a second import changes nothing and a reviewer's work is never overwritten.
func TestImportFlags_KeepsAReviewedTranslation(t *testing.T) {
	f := flagFake()
	rows := []terminology.FlagRow{
		{Line: 1, WikidataID: "Q2", Language: "hr", Label: "placeholder two", Note: "Placeholder."},
		{Line: 2, WikidataID: "Q1", Language: "hr", Label: "placeholder hr", Note: "Placeholder."},
	}
	if _, err := newCurator(f).ImportFlags(t.Context(), rows); err != nil {
		t.Fatal(err)
	}
	if f.stored["t2 hr"].Review.Status != terminology.ReviewApproved || f.stored["t2 hr"].VersionID != 3 {
		t.Errorf("approved translation = %+v, want it kept", f.stored["t2 hr"])
	}
	again, err := newCurator(f).ImportFlags(t.Context(), rows)
	if err != nil {
		t.Fatal(err)
	}
	if want := (terminology.ImportReport{Unchanged: 2}); !reflect.DeepEqual(again, want) {
		t.Errorf("second import = %+v, want %+v", again, want)
	}
}

// A row naming no term, a language that is no term language or that the term has no labels in, a label the term
// does not have in the language (the file names the wrong label, so a flag never lands on another term or language),
// or a note that is empty or over 500 characters is refused; the other rows are applied.
func TestImportFlags_RefusesBadRows(t *testing.T) {
	f := flagFake()
	report, err := newCurator(f).ImportFlags(t.Context(), []terminology.FlagRow{
		{Line: 1, WikidataID: "Q9", Language: "hr", Label: "placeholder hr", Note: "Placeholder."},
		{Line: 2, WikidataID: "Q1", Language: "de", Label: "placeholder hr", Note: "Placeholder."},
		{Line: 3, WikidataID: "Q1", Language: "xx", Label: "placeholder hr", Note: "Placeholder."},
		{Line: 4, WikidataID: "Q1", Language: "hr", Label: "placeholder hr", Note: "  "},
		{Line: 5, WikidataID: "Q1", Language: "hr", Label: "placeholder hr", Note: strings.Repeat("a", 501)},
		{Line: 6, WikidataID: "Q1", Language: "hr", Label: "wrong tr", Note: "Placeholder."},
		{Line: 7, WikidataID: "Q1", Language: "tr", Label: "wrong tr", Note: "Placeholder."},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{
		1: "wikidata: no such term", 2: "language: the term has no labels in it", 3: "language", 4: "note", 5: "note",
		6: "label: not one of the term's labels in the language",
	}
	if got := reasons(report); !reflect.DeepEqual(got, want) || report.Updated != 1 {
		t.Errorf("refused = %v, updated %d; want %v, updated 1", got, report.Updated, want)
	}
}

// The flags file is one flag per line: the term's Wikidata id, its English name (for the reader), the language, the
// wrong label and the note. A line that is not one is refused; blank lines are skipped.
func TestDecodeFlags_ReadsOneFlagPerLine(t *testing.T) {
	t.Parallel()
	rows, refused, err := terminology.DecodeFlags(strings.NewReader(
		`{"wikidata":"Q1","term":"placeholder","language":"tr","label":"wrong tr","note":"Placeholder."}` + "\n\n" +
			`{"wikidata":"Q1","language":"tr","label":"wrong tr"}` + "\n" + `{"wikidata":"Q1","language":"tr","label":"x","note":"n","extra":1}` + "\n" + `not json` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []terminology.FlagRow{{Line: 1, WikidataID: "Q1", Language: "tr", Label: "wrong tr", Note: "Placeholder."}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows %+v, want %+v", rows, want)
	}
	if len(refused) != 3 || refused[0].Line != 3 || refused[2].Line != 5 || refused[0].Reason != "not a flag row" {
		t.Errorf("refused %+v, want lines 3 to 5", refused)
	}
}

// The committed flags (assets/terminology/translation-flags.jsonl) import whole against the committed seed: every
// line names a seeded term, a language it has labels in and one of those labels.
func TestImportFlags_TheCommittedFlagsImportWhole(t *testing.T) {
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
	f.stored, f.termReviews = map[string]terminology.Translation{}, map[string]terminology.Review{}
	for i, r := range seeded {
		r.Term.ID = fmt.Sprintf("t-%d", i)
		f.terms[r.Term.ID] = r.Term
		for _, l := range r.Term.Labels {
			f.stored[r.Term.ID+" "+l.Language] = terminology.Translation{TermID: r.Term.ID, Language: l.Language, VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
		}
	}
	file, err := os.Open("../../assets/terminology/translation-flags.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	rows, refused, err := terminology.DecodeFlags(file)
	if err != nil || len(refused) > 0 || len(rows) == 0 {
		t.Fatalf("decoding the flags: %d rows, %v, refused %+v", len(rows), err, refused)
	}
	report, err := newCurator(f).ImportFlags(t.Context(), rows)
	if err != nil || len(report.Refused) > 0 || report.Updated != len(rows) {
		t.Errorf("importing the flags = %+v, %v; want every row flagged", report, err)
	}
}
