package terminology

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// ExplanationRow is one line of the drafted explanations' file (assets/terminology/explanations.draft.jsonl): the
// term by its Wikidata id and its explanation per app language, and the line it is on.
type ExplanationRow struct {
	Texts      map[string]string
	WikidataID string
	Line       int
}

// ImportExplanations adds the drafted plain-language explanations (SAL-13) a term does not have yet in a language,
// as unreviewed and without a reviewer, so they land in the review queue; an explanation already there is kept as it
// is, whatever its text (the reviewer's work is never overwritten), so a second import changes nothing. The report
// counts explanations: created and kept (unchanged). A row naming no term, with a language that is no app language
// or with a text that is not 1 to 500 characters once trimmed is refused whole, the rest applied, each row in its own
// transaction; a database error stops the import, as in ImportSeed. Written through the review panel's repository
// methods (LockEntry, PutExplanation). Not audited, as the curator's file: no reviewer is named.
func (c *Curator) ImportExplanations(ctx context.Context, rows []ExplanationRow) (ImportReport, error) {
	var report ImportReport
	for _, r := range rows {
		langs, err := explanationTexts(r.Texts)
		if err != nil {
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: err.Error()})
			continue
		}
		var created, kept int
		err = c.tx.WithTx(ctx, func(ctx context.Context) error {
			created, kept = 0, 0
			id, err := c.repo.TermIDByWikidata(ctx, r.WikidataID)
			if err != nil {
				return err
			}
			for _, lang := range langs {
				st, err := c.repo.LockEntry(ctx, id, lang)
				switch {
				case err != nil:
					return err
				case st.Explanation != nil:
					kept++
					continue
				}
				e := Explanation{TermID: id, Language: lang, Text: strings.TrimSpace(r.Texts[lang]), UpdatedAt: c.clock.Now(), VersionID: 1, Review: Review{Status: ReviewUnreviewed}}
				if err := c.repo.PutExplanation(ctx, e); err != nil {
					return err
				}
				created++
			}
			return nil
		})
		switch {
		case errors.Is(err, ErrTermNotFound):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "wikidata: no such term"})
		case err != nil:
			return report, fmt.Errorf("terminology: explanations import, line %d: %w", r.Line, err)
		default:
			report.Created += created
			report.Unchanged += kept
		}
	}
	return report, nil
}

// explanationTexts are the languages of texts in the enum's order, or a refusal naming the first language that is
// no app language or whose text is not 1 to 500 characters once trimmed.
func explanationTexts(texts map[string]string) ([]string, error) {
	var bad []string
	for lang, text := range texts {
		text = strings.TrimSpace(text)
		if n := utf8.RuneCountInString(text); !explanationLanguage(lang) || n == 0 || n > maxExplanation || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
			bad = append(bad, lang)
		}
	}
	if len(bad) > 0 {
		slices.Sort(bad)
		return nil, refusal("explanations: " + bad[0])
	}
	var out []string
	for _, l := range languages {
		if _, ok := texts[l]; ok {
			out = append(out, l)
		}
	}
	return out, nil
}
