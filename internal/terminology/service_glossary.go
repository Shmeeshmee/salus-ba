package terminology

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The bounds of a line's glossary (SAL-5, docs/plans/consult-intelligence.md).
const (
	glossaryWords   = 200 // words of a line looked up
	glossaryPhrase  = 3   // words of a phrase, at most
	glossarySuffix  = 3   // letters a phrase's last word may have beyond the term
	glossaryStem    = 4   // letters a shortened word keeps, at least
	glossaryMinWord = 3   // letters of a one-word phrase, at least
	glossaryLimit   = 20  // terms of a line's glossary, at most
)

// Glossary returns the terms said in text, a line of a consultation in language from, each with its label in
// language to, for the line's translation (docs/plans/consult-intelligence.md, SAL-5). A term is found by a
// phrase of the line, one to three words, equal to one of its labels in from, case and accents aside; the
// phrase's last word may carry up to three more letters (a plural, a case ending, a Turkish suffix) as long as
// four are left, and an Arabic phrase a proclitic ("and", "with", "the") on its first word. Words of two letters or fewer are no phrase of their own, punctuation around a word is not part
// of it, and only the first 200 words are looked up. At most 20 terms, the most common first. Both languages
// must be the terms' app languages and differ (else ErrInvalidSearch). For the transcription agent: the terms are
// reference data, so nothing is audited; the line is patient speech and is never logged, stored or put into an
// error.
func (s *Service) Glossary(ctx context.Context, text, from, to string) ([]GlossaryEntry, error) {
	if from == to || from == "la" || to == "la" || !slices.Contains(languages, from) || !slices.Contains(languages, to) {
		return nil, ErrInvalidSearch
	}
	candidates := phrases(text, from == "ar")
	if len(candidates) == 0 {
		return nil, nil
	}
	entries, err := s.repo.Glossary(ctx, GlossaryQuery{From: from, To: to, Candidates: candidates, Limit: glossaryLimit})
	if err != nil {
		return nil, fmt.Errorf("terminology: glossary: %w", err)
	}
	return entries, nil
}

// phrases are the candidate phrases of text, each once, in the order they first occur; with arabic, each phrase
// also without a proclitic on its first word.
func phrases(text string, arabic bool) []string {
	var words []string
	for _, f := range strings.Fields(text) {
		if w := strings.TrimFunc(f, notWord); w != "" {
			words = append(words, w)
		}
		if len(words) == glossaryWords {
			break
		}
	}
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for i := range words {
		for n := 1; n <= glossaryPhrase && i+n <= len(words); n++ {
			last := words[i+n-1]
			if n == 1 && utf8.RuneCountInString(last) < glossaryMinWord {
				continue
			}
			heads := []string{strings.Join(words[i:i+n-1], " ")}
			if bare := withoutProclitic(words[i]); arabic && bare != "" && n > 1 {
				heads = append(heads, strings.Join(append([]string{bare}, words[i+1:i+n-1]...), " "))
			}
			lasts := forms(last)
			if bare := withoutProclitic(last); arabic && bare != "" && n == 1 {
				lasts = append(lasts, forms(bare)...)
			}
			for _, head := range heads {
				for _, w := range lasts {
					add(strings.TrimSpace(head + " " + w))
				}
			}
		}
	}
	return out
}

// forms are w and w without its last one to three letters, keeping four at least.
func forms(w string) []string {
	r := []rune(w)
	out := []string{w}
	for cut := 1; cut <= glossarySuffix && len(r)-cut >= glossaryStem; cut++ {
		out = append(out, string(r[:len(r)-cut]))
	}
	return out
}

// arabicProclitics are the prefixes an Arabic word may carry: "and", "so", "with", "for", "like", "the", and
// their joins with "the"; the longest first.
var arabicProclitics = []string{"وال", "فال", "بال", "كال", "لل", "ال", "و", "ف", "ب", "ل", "ك"}

// withoutProclitic is w without its Arabic proclitic, when three letters at least are left; empty otherwise.
func withoutProclitic(w string) string {
	for _, p := range arabicProclitics {
		if rest, ok := strings.CutPrefix(w, p); ok && utf8.RuneCountInString(rest) >= glossaryMinWord {
			return rest
		}
	}
	return ""
}

// notWord is punctuation or a symbol around a word.
func notWord(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }
