package terminology

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// FlagRow is one line of the translation flags' file (assets/terminology/translation-flags.jsonl): a term by its
// Wikidata id, a language, the label that is wrong there and the note saying what is wrong, and the line it is on.
type FlagRow struct {
	WikidataID string
	Language   string
	Label      string
	Note       string
	Line       int
}

// ImportFlags marks the flagged translations as needing changes, so they come up under the review panel's "Needs
// changes" filter, the most common term first: the note is the flag's, trimmed; no reviewer is named (the file names no one,
// as the curator's CSV); reviewed_at is now; the term's status follows (followedStatus), without a reviewer. Only a
// translation never reviewed is flagged: one with a review (a reviewer's, the curator's file's or an earlier flag's)
// is kept as it is, so a second import changes nothing and a reviewer's work is never overwritten. The report counts
// translations: updated (flagged) and unchanged (kept). A row naming no term, a language that is no term language or
// that the term has no labels in, a label the term does not have in the language (so a flag never lands on another
// term or language, and one whose label a reviewer already changed is no longer applied), or a note that is not 1 to 500 characters once trimmed is refused, the rest
// applied, each row in its own transaction; a database error stops the import. Not audited, as the curator's file.
func (c *Curator) ImportFlags(ctx context.Context, rows []FlagRow) (ImportReport, error) {
	var report ImportReport
	for _, r := range rows {
		note := strings.TrimSpace(r.Note)
		switch {
		case !slices.Contains(languages, r.Language):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "language"})
			continue
		case note == "" || utf8.RuneCountInString(note) > maxNote || !utf8.ValidString(note) || strings.ContainsRune(note, 0):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "note"})
			continue
		}
		var flagged bool
		err := c.tx.WithTx(ctx, func(ctx context.Context) error {
			var err error
			flagged, err = c.flag(ctx, r, note)
			return err
		})
		switch {
		case errors.Is(err, ErrTermNotFound):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "wikidata: no such term"})
		case errors.Is(err, ErrNoTranslation):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "language: the term has no labels in it"})
		case errors.Is(err, errNoSuchLabel):
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: "label: not one of the term's labels in the language"})
		case err != nil:
			return report, fmt.Errorf("terminology: flags import, line %d: %w", r.Line, err)
		case flagged:
			report.Updated++
		default:
			report.Unchanged++
		}
	}
	return report, nil
}

// errNoSuchLabel is a flag's label the term does not have in the flag's language.
var errNoSuchLabel = errors.New("terminology: no such label")

// flag marks r's translation as needing changes with note, in the transaction in ctx, and reports whether it did:
// false for a translation already reviewed.
func (c *Curator) flag(ctx context.Context, r FlagRow, note string) (bool, error) {
	id, err := c.repo.TermIDByWikidata(ctx, r.WikidataID)
	if err != nil {
		return false, err
	}
	st, err := c.repo.LockEntry(ctx, id, r.Language)
	if err != nil {
		return false, err
	}
	tr, found := translationOf(st, id, r.Language)
	switch {
	case !found:
		return false, ErrNoTranslation
	case !slices.ContainsFunc(st.Labels, func(l Label) bool { return l.Text == r.Label }):
		return false, errNoSuchLabel
	case !tr.Review.ReviewedAt.IsZero():
		return false, nil
	}
	now := c.clock.Now()
	tr.Review = Review{Status: ReviewNeedsChanges, ReviewedAt: now, Note: note}
	tr.VersionID, tr.UpdatedAt = tr.VersionID+1, now
	if err := c.repo.PutTranslation(ctx, tr); err != nil {
		return false, err
	}
	if status := followedStatus(st, tr); status != st.TermReview.Status {
		r := st.TermReview
		r.Status, r.ReviewedBy, r.ReviewedAt = status, "", now
		if err := c.repo.SetTermReview(ctx, id, r, now); err != nil {
			return false, err
		}
	}
	return true, nil
}
