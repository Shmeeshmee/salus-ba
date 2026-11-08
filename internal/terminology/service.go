package terminology

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// The bounds of a search (api/openapi.yaml, searchTerminology).
const (
	maxText  = 100 // characters
	maxLimit = 25
)

// languages are the enum term_language: Latin, then the ten app languages.
var languages = []string{"la", "bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

// categories are the enum term_category.
var categories = []string{"anatomy", "diagnosis", "symptom", "procedure", "drug"}

// Service is the terminology's business rules: who may search and what a search may ask. The search text is
// never logged, stored or put into an error.
type Service struct {
	repo Repository
	// language is an account's UI language: its own choice, else the one its client detected; empty for none.
	language func(ctx context.Context, accountID string) (string, error)
}

// NewService returns the Service on repo; language gives an account's UI language for the default languages.
func NewService(repo Repository, language func(ctx context.Context, accountID string) (string, error)) *Service {
	return &Service{repo: repo, language: language}
}

// Search returns at most q.Limit terms that match q.Text, best first, each with its labels in q.Languages, in
// that order (docs/terminology/SPEC.md §3 "API and UI"). Staff only: a patient gets ErrForbidden. A text that
// is blank or longer than 100 characters, a limit outside 1 to 25, or an unknown or repeated language or
// category gives ErrInvalidSearch. Without languages, the labels are Latin, the caller's UI language and
// English. Within a match type, labels in Latin and the UI language rank first, then English, then the others.
// Not audited: the terms are reference data, no patient data is read.
func (s *Service) Search(ctx context.Context, c Caller, q Search) ([]Match, error) {
	if !c.Staff {
		return nil, ErrForbidden
	}
	q.Text = strings.TrimSpace(q.Text)
	if !valid(q) {
		return nil, ErrInvalidSearch
	}
	if len(q.Languages) == 0 {
		ui, err := s.language(ctx, c.AccountID)
		if err != nil {
			return nil, fmt.Errorf("terminology: the caller's language: %w", err)
		}
		q.Languages = defaultLanguages(ui)
	}
	q.UILanguage = uiLanguage(q.Languages)
	terms, err := s.repo.Search(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("terminology: search: %w", err)
	}
	matches := make([]Match, len(terms))
	for i, t := range terms {
		matches[i] = match(t, q.Languages)
	}
	return matches, nil
}

// valid reports whether q, its text trimmed, is a search the API accepts.
func valid(q Search) bool {
	n := utf8.RuneCountInString(q.Text)
	if n == 0 || n > maxText || !utf8.ValidString(q.Text) || strings.ContainsRune(q.Text, 0) {
		return false
	}
	if q.Limit < 1 || q.Limit > maxLimit || (q.Category != "" && !slices.Contains(categories, q.Category)) {
		return false
	}
	for i, l := range q.Languages {
		if !slices.Contains(languages, l) || slices.Contains(q.Languages[:i], l) {
			return false
		}
	}
	return true
}

// defaultLanguages are Latin, the UI language ui (when it is one of the terms' languages) and English.
func defaultLanguages(ui string) []string {
	out := []string{"la"}
	if ui != "la" && ui != "en" && slices.Contains(languages, ui) {
		out = append(out, ui)
	}
	return append(out, "en")
}

// uiLanguage is the caller's UI language for the ranking: the first of langs other than Latin, as the defaults
// and the web both put it right after Latin (English when it is the UI language); empty when there is none. It
// saves reading the account on every keystroke.
func uiLanguage(langs []string) string {
	for _, l := range langs {
		if l != "la" {
			return l
		}
	}
	return ""
}

// match groups t's labels per language of langs, in that order, leaving out a language without a label. The
// repository gives each language's labels preferred first, then alphabetically: the first is the preferred
// one.
func match(t Term, langs []string) Match {
	m := Match{Labels: []Names{}}
	for _, lang := range langs {
		var names *Names
		for _, l := range t.Labels {
			switch {
			case l.Language != lang:
			case names == nil:
				names = &Names{Language: lang, Preferred: l.Text, Synonyms: []string{}}
			default:
				names.Synonyms = append(names.Synonyms, l.Text)
			}
		}
		if names != nil {
			m.Labels = append(m.Labels, *names)
		}
	}
	t.Labels = nil
	m.Term = t
	return m
}
