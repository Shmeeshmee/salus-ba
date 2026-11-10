package terminology

import (
	"context"
	"time"
)

// Repository reads the terms. They are reference data: no clinic or patient scope, and the app only reads.
type Repository interface {
	// Search returns at most s.Limit terms of s.Category (any when empty) with a label in any language that
	// matches s.Text, case and accents aside: an exact label first, then a label that starts with the text,
	// then a word within trigram distance (prefixes of later words and misspellings), each by word similarity
	// and then by the term's rank. Each term carries its labels in s.Languages only, by language in the enum's
	// order, the preferred one first, then the synonyms alphabetically.
	Search(ctx context.Context, s Search) ([]Term, error)
	// Glossary returns at most q.Limit terms with a label in q.From that equals one of q.Candidates, case and
	// accents aside, the most common (rank) first, each once: the label it was found by (the longest, the
	// preferred one of a length, when several) and its label in q.To, the preferred one, else the first alphabetically. A term without a label in
	// q.To is left out.
	Glossary(ctx context.Context, q GlossaryQuery) ([]GlossaryEntry, error)
	// Tag returns the candidates that equal a label in their language of a term with an explanation (in any
	// language), case and accents aside, each once with the most common (rank) such term.
	Tag(ctx context.Context, candidates []Candidate) ([]CandidateTerm, error)
	// Explanations returns per term with one its explanation in language, else its English one, leaving out an
	// explanation that needs changes; by term id.
	Explanations(ctx context.Context, language string) ([]Explanation, error)
}

// CurationRepository writes the terms: salusctl's, on the owner's connection (salus_app may only read them).
type CurationRepository interface {
	// CreateFromSeed inserts t and its labels unless a term with t's Wikidata id exists, and reports whether it
	// did.
	CreateFromSeed(ctx context.Context, t Term) (bool, error)
	// Term returns the term with id and its labels in every language, by language in the enum's order, the
	// preferred one first, then alphabetically; ErrTermNotFound when there is none.
	Term(ctx context.Context, id string) (Term, error)
	// UpdateTerm writes t's columns, its updated_at at, and replaces its labels; ErrTermNotFound when there is
	// no term t.ID, ErrDuplicateWikidata when another term has t's Wikidata id. A term has a translation in exactly
	// the languages it has labels in: one for a new language takes the term's status, one for a language left
	// without labels is deleted (CreateFromSeed's are unreviewed).
	UpdateTerm(ctx context.Context, t Term, at time.Time) error
	// SetTranslationsStatus sets every translation of the term to status with no reviewer, reviewed_at at unless
	// status is unreviewed (then each keeps its own), the next version and updated_at at.
	SetTranslationsStatus(ctx context.Context, termID, status string, at time.Time) error
	// Terms returns every term of review status status (any when empty) with its labels as Term has them, by
	// category in the enum's order, rank descending, then id.
	Terms(ctx context.Context, status string) ([]Term, error)
	// CountByCategory returns how many terms each category holds, in the enum's order; an empty one is absent.
	CountByCategory(ctx context.Context) ([]CategoryCount, error)
	// TermIDByWikidata returns the id of the term with the Wikidata id; ErrTermNotFound when there is none.
	TermIDByWikidata(ctx context.Context, wikidataID string) (string, error)
	// LockEntry and PutExplanation are ReviewRepository's (the review panel's writes, SAL-14), for the import of the
	// drafted explanations.
	LockEntry(ctx context.Context, termID, language string) (EntryState, error)
	PutExplanation(ctx context.Context, e Explanation) error
}

// ReviewRepository reads the review queue and writes a reviewer's edits and reviews (the review panel, SAL-14), on
// the app's connection, which may write only these.
type ReviewRepository interface {
	// Queue returns at most q.Limit entries of the terms in q.Language: each term with its labels in the language,
	// Latin and English (by language, the preferred one first, then alphabetically), its translation and its
	// explanation in the language. An entry's status is needs_changes when its translation or explanation needs
	// changes, else unreviewed when there is no translation or either is unreviewed, else approved; only those of
	// q.Status when set. Unreviewed first, then needs_changes, then approved, each by rank descending, then id;
	// after the entry q.AfterStatus, q.AfterRank, q.AfterID when q.AfterID is set.
	Queue(ctx context.Context, q QueueFilter) ([]QueueRow, error)
	// Progress returns per term language with any term how many terms there are and how many of their entries in
	// it are approved and need changes, statuses as in Queue; none without terms.
	Progress(ctx context.Context) ([]LanguageProgress, error)
	// LockEntry locks the term for the transaction in ctx and returns what a review write of it in language reads;
	// ErrTermNotFound when there is no such term.
	LockEntry(ctx context.Context, termID, language string) (EntryState, error)
	// ReplaceLabels replaces the term's labels in language with labels.
	ReplaceLabels(ctx context.Context, termID, language string, labels []Label) error
	// PutTranslation stores t, as it is, under its term and language.
	PutTranslation(ctx context.Context, t Translation) error
	// PutExplanation stores e, as it is, under its term and language.
	PutExplanation(ctx context.Context, e Explanation) error
	// SetTermReview sets the term's review status, reviewed_by and reviewed_at to r's (its curator's note stays)
	// and its updated_at to at.
	SetTermReview(ctx context.Context, termID string, r Review, at time.Time) error
}
