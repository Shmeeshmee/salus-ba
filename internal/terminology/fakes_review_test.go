package terminology_test

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// txKey marks a context inside the fake transaction.
type txKey struct{}

// fakeReviewTx runs fn with a marked context, counting the transactions.
type fakeReviewTx struct{ txs int }

func (f *fakeReviewTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.txs++
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeAudit keeps the recorded events and whether each was recorded inside a transaction.
type fakeAudit struct {
	err     error
	events  []auditevent.AuditEvent
	outside int
}

func (f *fakeAudit) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		f.outside++
	}
	f.events = append(f.events, e)
	return f.err
}

// fakeTerm is one term as the fake review repository keeps it.
type fakeTerm struct {
	termAt       time.Time // the term's updated_at, set by SetTermReview
	labels       map[string][]terminology.Label
	translations map[string]terminology.Translation
	explanations map[string]terminology.Explanation
	review       terminology.Review
}

// fakeReviews keeps the terms in memory by id and records the queue filter and the writes; every read and write
// must run inside a transaction, except the queue and the progress.
type fakeReviews struct {
	err      error // returned by every method when set
	terms    map[string]*fakeTerm
	filter   *terminology.QueueFilter
	rows     []terminology.QueueRow
	progress []terminology.LanguageProgress
	writes   []string // the write methods called, in order
	outside  []string // the calls made outside a transaction
}

func newFakeReviews() *fakeReviews { return &fakeReviews{terms: map[string]*fakeTerm{}} }

// add stores a term with labels in each of langs (a preferred "<lang> label") and an unreviewed translation for each.
func (f *fakeReviews) add(id string, langs ...string) *fakeTerm {
	t := &fakeTerm{
		labels: map[string][]terminology.Label{}, translations: map[string]terminology.Translation{},
		explanations: map[string]terminology.Explanation{}, review: terminology.Review{Status: terminology.ReviewUnreviewed},
	}
	for _, l := range langs {
		t.labels[l] = []terminology.Label{{Language: l, Text: l + " label", Preferred: true}}
		t.translations[l] = terminology.Translation{TermID: id, Language: l, VersionID: 1, Review: terminology.Review{Status: terminology.ReviewUnreviewed}}
	}
	f.terms[id] = t
	return t
}

func (f *fakeReviews) inTx(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		f.outside = append(f.outside, call)
	}
}

func (f *fakeReviews) Queue(_ context.Context, q terminology.QueueFilter) ([]terminology.QueueRow, error) {
	f.filter = &q
	return f.rows, f.err
}

func (f *fakeReviews) Progress(context.Context) ([]terminology.LanguageProgress, error) {
	return f.progress, f.err
}

func (f *fakeReviews) LockEntry(ctx context.Context, termID, language string) (terminology.EntryState, error) {
	f.inTx(ctx, "LockEntry")
	if f.err != nil {
		return terminology.EntryState{}, f.err
	}
	t, ok := f.terms[termID]
	if !ok {
		return terminology.EntryState{}, terminology.ErrTermNotFound
	}
	s := terminology.EntryState{TermReview: t.review, Labels: slices.Clone(t.labels[language])}
	for _, tr := range t.translations {
		s.Translations = append(s.Translations, tr)
	}
	if e, ok := t.explanations[language]; ok {
		s.Explanation = &e
	}
	return s, nil
}

func (f *fakeReviews) ReplaceLabels(ctx context.Context, termID, language string, labels []terminology.Label) error {
	f.inTx(ctx, "ReplaceLabels")
	f.writes = append(f.writes, "ReplaceLabels")
	f.terms[termID].labels[language] = labels
	return nil
}

func (f *fakeReviews) PutTranslation(ctx context.Context, tr terminology.Translation) error {
	f.inTx(ctx, "PutTranslation")
	f.writes = append(f.writes, "PutTranslation")
	f.terms[tr.TermID].translations[tr.Language] = tr
	return nil
}

func (f *fakeReviews) PutExplanation(ctx context.Context, e terminology.Explanation) error {
	f.inTx(ctx, "PutExplanation")
	f.writes = append(f.writes, "PutExplanation")
	f.terms[e.TermID].explanations[e.Language] = e
	return nil
}

func (f *fakeReviews) SetTermReview(ctx context.Context, termID string, r terminology.Review, at time.Time) error {
	f.inTx(ctx, "SetTermReview")
	f.writes = append(f.writes, "SetTermReview")
	if at.IsZero() {
		return errors.New("no updated_at")
	}
	t := f.terms[termID]
	t.review, t.termAt = r, at
	return nil
}
