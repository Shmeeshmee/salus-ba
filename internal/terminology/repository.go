package terminology

import "context"

// Repository reads the terms. They are reference data: no clinic or patient scope, and the app only reads.
type Repository interface {
	// Search returns at most s.Limit terms of s.Category (any when empty) with a label in any language that
	// matches s.Text, case and accents aside: an exact label first, then a label that starts with the text,
	// then a word within trigram distance (prefixes of later words and misspellings), each by word similarity
	// and then by the term's rank. Each term carries its labels in s.Languages only, by language in the enum's
	// order, the preferred one first, then the synonyms alphabetically.
	Search(ctx context.Context, s Search) ([]Term, error)
}
