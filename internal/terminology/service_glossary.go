package terminology

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
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
	seen := map[string]bool{}
	var out []string
	for _, p := range phraseSpans(words(text, glossaryWords), arabic) {
		if !seen[p.text] {
			seen[p.text] = true
			out = append(out, p.text)
		}
	}
	return out
}

// word is a word of a text, punctuation around it left out, with its place in the text as UTF-16 offsets.
type word struct {
	text       string
	start, end int
}

// phrase is a candidate phrase and the place in the text of the words it was made from: the whole of its last
// word, also when the phrase has a shortened form of it.
type phrase struct {
	text       string
	start, end int
}

// words are the first max words of text: the runs of characters between white space, without the punctuation and
// symbols around them; a run of only those is no word.
func words(text string, max int) []word {
	var out []word
	var field []rune
	start, pos := 0, 0
	flush := func() {
		lo, hi := 0, len(field)
		for lo < hi && notWord(field[lo]) {
			lo++
		}
		for hi > lo && notWord(field[hi-1]) {
			hi--
		}
		if lo < hi {
			s := start + utf16Len(field[:lo])
			out = append(out, word{text: string(field[lo:hi]), start: s, end: s + utf16Len(field[lo:hi])})
		}
		field = field[:0]
	}
	for _, r := range text {
		switch {
		case !unicode.IsSpace(r):
			if len(field) == 0 {
				start = pos
			}
			field = append(field, r)
		case len(field) > 0:
			if flush(); len(out) == max {
				return out
			}
		}
		pos += utf16.RuneLen(r)
	}
	if len(field) > 0 && len(out) < max {
		flush()
	}
	return out
}

// utf16Len is the number of UTF-16 code units of rs.
func utf16Len(rs []rune) int {
	n := 0
	for _, r := range rs {
		n += utf16.RuneLen(r)
	}
	return n
}

// phraseSpans are the candidate phrases of ws, one to three words each, in the order they occur, the whole words
// before their shortened forms; repeats are kept.
func phraseSpans(ws []word, arabic bool) []phrase {
	var out []phrase
	for i := range ws {
		for n := 1; n <= glossaryPhrase && i+n <= len(ws); n++ {
			last := ws[i+n-1].text
			if n == 1 && utf8.RuneCountInString(last) < glossaryMinWord {
				continue
			}
			heads := []string{joinWords(ws[i : i+n-1])}
			if bare := withoutProclitic(ws[i].text); arabic && bare != "" && n > 1 {
				heads = append(heads, strings.TrimSpace(bare+" "+joinWords(ws[i+1:i+n-1])))
			}
			lasts := forms(last)
			if bare := withoutProclitic(last); arabic && bare != "" && n == 1 {
				lasts = append(lasts, forms(bare)...)
			}
			for _, head := range heads {
				for _, w := range lasts {
					out = append(out, phrase{text: strings.TrimSpace(head + " " + w), start: ws[i].start, end: ws[i+n-1].end})
				}
			}
		}
	}
	return out
}

// joinWords is the words' texts joined with spaces.
func joinWords(ws []word) string {
	texts := make([]string, len(ws))
	for i, w := range ws {
		texts[i] = w.text
	}
	return strings.Join(texts, " ")
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
