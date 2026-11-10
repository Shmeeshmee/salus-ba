package terminology

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// validStatus reports whether status is a review status (review_status).
func validStatus(status string) bool {
	return status == ReviewUnreviewed || status == ReviewApproved || status == ReviewNeedsChanges
}

// maxLabel is the longest label in characters (term_label's check).
const maxLabel = 300

// wikidataItem is the form of a Wikidata item id (term's check).
var wikidataItem = regexp.MustCompile(`^Q[1-9][0-9]*$`)

// Transactor runs fn in one transaction (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Curator is the curation round trip of docs/terminology/SPEC.md §3 "Curation": the seed and the curator's
// spreadsheet imported, the terms exported. salusctl's, on the owner's connection; the app never builds one.
// Not audited: the terms are reference data without personal data, and the review keeps only its date.
type Curator struct {
	repo  CurationRepository
	tx    Transactor
	clock clock.Clock
}

// NewCurator returns the Curator on repo.
func NewCurator(repo CurationRepository, tx Transactor, clk clock.Clock) *Curator {
	return &Curator{repo: repo, tx: tx, clock: clk}
}

// refusal is a row's reason to be refused; it names a column, never a value.
type refusal string

func (r refusal) Error() string { return string(r) }

// ImportSeed adds the seed's terms that are not there yet (by Wikidata id) as drafts and leaves the others as
// they are: a new fetch never overwrites the curator's work. Each row is its own transaction; an invalid row is
// refused and reported, the rest applied. A database error stops the import: the rows before it stay applied,
// and running it again is safe.
func (c *Curator) ImportSeed(ctx context.Context, rows []ImportRow) (ImportReport, error) {
	var report ImportReport
	for _, r := range rows {
		t, err := normalise(r.Term)
		if err == nil && t.WikidataID == "" {
			err = refusal("wikidata: missing")
		}
		if err != nil {
			report.Refused = append(report.Refused, RowError{Line: r.Line, Reason: err.Error()})
			continue
		}
		t.ReviewStatus, t.ReviewedAt, t.Note = ReviewUnreviewed, time.Time{}, ""
		var created bool
		err = c.tx.WithTx(ctx, func(ctx context.Context) error {
			var err error
			created, err = c.repo.CreateFromSeed(ctx, t)
			return err
		})
		if err != nil {
			return report, fmt.Errorf("terminology: seed import, line %d: %w", r.Line, err)
		}
		if created {
			report.Created++
		} else {
			report.Unchanged++
		}
	}
	return report, nil
}

// ImportCurated applies the curator's rows to the terms with their ids: columns written, labels replaced,
// reviewed_at set when a term turns approved or needs_changes (kept when its status stays or goes back to
// unreviewed), and reviewed_by cleared when its status changes (the file names no reviewer). A row equal to
// its term changes nothing, so a second import of the same file is a no-op. A row without an id, with an id
// that names no term or that an earlier row had, or with an invalid value is refused and reported, the rest
// applied, each row in its own transaction. A database error stops the import, as in ImportSeed.
func (c *Curator) ImportCurated(ctx context.Context, rows []ImportRow) (ImportReport, error) {
	var report ImportReport
	seen := map[string]bool{}
	for _, r := range rows {
		t, err := normalise(r.Term)
		switch {
		case err != nil:
		case t.ID == "":
			err = refusal("id: missing; new terms come from the seed")
		case !validStatus(t.ReviewStatus):
			err = refusal("review_status: not unreviewed, approved or needs_changes")
		case seen[t.ID]:
			err = refusal("id: repeated in the file")
		default:
			seen[t.ID] = true
			var updated bool
			updated, err = c.applyCurated(ctx, t)
			switch {
			case err == nil && updated:
				report.Updated++
			case err == nil:
				report.Unchanged++
			}
		}
		if refused := refusal(""); errors.As(err, &refused) {
			report.Refused = append(report.Refused, RowError{ID: r.Term.ID, Line: r.Line, Reason: refused.Error()})
		} else if err != nil {
			return report, fmt.Errorf("terminology: curated import, line %d: %w", r.Line, err)
		}
	}
	return report, nil
}

// applyCurated writes t over its stored term in one transaction unless they are equal, and reports whether it
// wrote.
func (c *Curator) applyCurated(ctx context.Context, t Term) (bool, error) {
	var updated bool
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		stored, err := c.repo.Term(ctx, t.ID)
		if errors.Is(err, ErrTermNotFound) {
			return refusal("id: no such term")
		}
		if err != nil {
			return err
		}
		t.ReviewedAt, t.ReviewedBy = stored.ReviewedAt, stored.ReviewedBy
		if t.ReviewStatus != stored.ReviewStatus {
			t.ReviewedBy = ""
			if t.ReviewStatus != ReviewUnreviewed {
				t.ReviewedAt = c.clock.Now()
			}
		}
		if same(stored, t) {
			return nil
		}
		err = c.repo.UpdateTerm(ctx, t, c.clock.Now())
		if errors.Is(err, ErrDuplicateWikidata) {
			return refusal("wikidata: another term has it")
		}
		updated = err == nil
		return err
	})
	return updated, err
}

// Export returns the terms of review status status (every term when empty) with all their labels, by category,
// rank descending, then id; ErrInvalidStatus for another status.
func (c *Curator) Export(ctx context.Context, status string) ([]Term, error) {
	if status != "" && !validStatus(status) {
		return nil, ErrInvalidStatus
	}
	terms, err := c.repo.Terms(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("terminology: export: %w", err)
	}
	return terms, nil
}

// Counts returns how many terms each category holds.
func (c *Curator) Counts(ctx context.Context) ([]CategoryCount, error) {
	counts, err := c.repo.CountByCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminology: counts: %w", err)
	}
	return counts, nil
}

// normalise trims t's values and its labels (white space collapsed, empty labels and repeats dropped, sorted as
// the repository returns them) and checks them against the tables' rules.
func normalise(t Term) (Term, error) {
	t.ID, t.Category, t.ReviewStatus = strings.TrimSpace(t.ID), strings.TrimSpace(t.Category), strings.TrimSpace(t.ReviewStatus)
	for _, code := range []*string{&t.WikidataID, &t.ICD10, &t.TA98, &t.ATC, &t.PCS, &t.Note} {
		*code = strings.TrimSpace(*code)
	}
	switch {
	case t.WikidataID != "" && !wikidataItem.MatchString(t.WikidataID):
		return t, refusal("wikidata: not a Wikidata item id")
	case !slices.Contains(categories, t.Category):
		return t, refusal("category: unknown")
	case t.Rank < 0:
		return t, refusal("rank: negative")
	}
	labels, err := normaliseLabels(t.Labels)
	t.Labels = labels
	return t, err
}

// normaliseLabels collapses the white space in each label, drops the empty ones and repeats, checks the
// languages, the length, that no label holds "|" and that there is one preferred label per language, and sorts them by language in the enum's order,
// the preferred one first, then by text.
func normaliseLabels(in []Label) ([]Label, error) {
	out := []Label{}
	for _, l := range in {
		l.Text = strings.Join(strings.Fields(l.Text), " ")
		switch {
		case !slices.Contains(languages, l.Language):
			return nil, refusal("labels: unknown language")
		case l.Text == "" || slices.ContainsFunc(out, func(o Label) bool { return o.Language == l.Language && o.Text == l.Text }):
			continue
		case utf8.RuneCountInString(l.Text) > maxLabel:
			return nil, refusal(l.Language + ": a label longer than 300 characters")
		case strings.Contains(l.Text, "|"): // the CSV's synonym separator
			return nil, refusal(l.Language + ": a label with |")
		}
		out = append(out, l)
	}
	sortLabels(out)
	for i, l := range out {
		first := i == 0 || out[i-1].Language != l.Language
		switch {
		case first && !l.Preferred:
			return nil, refusal(l.Language + ": synonyms without a preferred label")
		case !first && l.Preferred:
			return nil, refusal(l.Language + ": more than one preferred label")
		}
	}
	return out, nil
}

func sortLabels(labels []Label) {
	slices.SortStableFunc(labels, func(a, b Label) int {
		if c := cmp.Compare(slices.Index(languages, a.Language), slices.Index(languages, b.Language)); c != 0 {
			return c
		}
		if a.Preferred != b.Preferred {
			if a.Preferred {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Text, b.Text)
	})
}

// same reports whether the curated t holds what stored holds, its labels in any order.
func same(stored, t Term) bool {
	labels := slices.Clone(stored.Labels)
	sortLabels(labels)
	stored.Labels = labels
	return stored.ReviewedAt.Equal(t.ReviewedAt) && stored.Category == t.Category && stored.WikidataID == t.WikidataID &&
		stored.ICD10 == t.ICD10 && stored.TA98 == t.TA98 && stored.ATC == t.ATC && stored.PCS == t.PCS &&
		stored.Rank == t.Rank && stored.ReviewStatus == t.ReviewStatus && stored.Note == t.Note &&
		slices.Equal(stored.Labels, t.Labels)
}
