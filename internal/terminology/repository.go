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
	// no term t.ID, ErrDuplicateWikidata when another term has t's Wikidata id.
	UpdateTerm(ctx context.Context, t Term, at time.Time) error
	// Terms returns every term of review status status (any when empty) with its labels as Term has them, by
	// category in the enum's order, rank descending, then id.
	Terms(ctx context.Context, status string) ([]Term, error)
	// CountByCategory returns how many terms each category holds, in the enum's order; an empty one is absent.
	CountByCategory(ctx context.Context) ([]CategoryCount, error)
}
