package terminology

import (
	"cmp"
	"context"
	"fmt"
	"slices"
)

// tagWords is how many words of a text tagging looks at: a transcript line has far fewer, a document's line about
// as many at most.
const tagWords = 1000

// Tag marks the terms with a plain-language explanation in each of texts (SAL-13): the transcript's lines and a
// signed document's lines, for the tooltip that explains them. A term is found as Glossary finds it, by a phrase
// of one to three words equal to one of its labels in the text's language, case and accents aside, the last word
// with up to three more letters. Overlapping terms leave the one that starts first, of those the longest. The
// result has the spans of texts[i] at i, nil for none. Every text's language must be an app language (else
// ErrInvalidSearch). The texts are patient data: never logged, stored or put into an error; the spans are an
// overlay, the texts are not changed.
func (s *Service) Tag(ctx context.Context, texts []Text) ([][]Span, error) {
	all := make([][]phrase, len(texts))
	seen := map[Candidate]bool{}
	var candidates []Candidate
	for i, t := range texts {
		if !appLanguage(t.Language) {
			return nil, ErrInvalidSearch
		}
		all[i] = phraseSpans(words(t.Body, tagWords), t.Language == "ar")
		for _, p := range all[i] {
			if c := (Candidate{Language: t.Language, Phrase: p.text}); !seen[c] {
				seen[c] = true
				candidates = append(candidates, c)
			}
		}
	}
	out := make([][]Span, len(texts))
	if len(candidates) == 0 {
		return out, nil
	}
	found, err := s.repo.Tag(ctx, candidates)
	if err != nil {
		return nil, fmt.Errorf("terminology: tag: %w", err)
	}
	terms := make(map[Candidate]string, len(found))
	for _, f := range found {
		terms[Candidate{Language: f.Language, Phrase: f.Phrase}] = f.TermID
	}
	for i, t := range texts {
		out[i] = spans(all[i], t.Language, terms)
	}
	return out, nil
}

// spans are the phrases of ps that name a term in terms, without overlaps: by start, the longest first, a phrase
// that overlaps one already kept left out.
func spans(ps []phrase, language string, terms map[Candidate]string) []Span {
	var hits []Span
	for _, p := range ps {
		if id, ok := terms[Candidate{Language: language, Phrase: p.text}]; ok {
			hits = append(hits, Span{TermID: id, Start: p.start, End: p.end})
		}
	}
	slices.SortStableFunc(hits, func(a, b Span) int { return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(b.End, a.End)) })
	var out []Span
	for _, h := range hits {
		if len(out) == 0 || h.Start >= out[len(out)-1].End {
			out = append(out, h)
		}
	}
	return out
}

// Explanations are the plain-language explanations a viewer reads in language, one per term: the term's in language,
// else its English one; one that needs changes is not shown (SAL-13). Reference data, for any signed-in caller: not
// audited. ErrInvalidSearch for a language that is not an app language.
func (s *Service) Explanations(ctx context.Context, language string) ([]Explanation, error) {
	if !appLanguage(language) {
		return nil, ErrInvalidSearch
	}
	out, err := s.repo.Explanations(ctx, language)
	if err != nil {
		return nil, fmt.Errorf("terminology: explanations: %w", err)
	}
	return out, nil
}

// appLanguage reports whether l is one of the ten app languages (a term language other than Latin).
func appLanguage(l string) bool { return l != "la" && slices.Contains(languages, l) }
