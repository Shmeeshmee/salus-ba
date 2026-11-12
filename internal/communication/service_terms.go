package communication

import (
	"cmp"
	"context"
	"slices"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Tagger marks the terms with a plain-language explanation in texts, each in its app language (terminology.Service,
// SAL-13). The texts are messages: never logged, stored or put into an error.
type Tagger interface {
	Tag(ctx context.Context, texts []terminology.Text) ([][]terminology.Span, error)
}

// Languages gives an account's UI language, an app language, or "" when it has none (auth.Service, through
// internal/api).
type Languages interface {
	Language(ctx context.Context, accountID string) (string, error)
}

// fallbackLanguage is the language the terms are also looked for in, and the only one for a caller without a
// language: medical terms are often written in English in any language's message.
const fallbackLanguage = "en"

// withTerms marks the explained terms in the bodies of msgs for the caller (SAL-13, as the transcript's lines are):
// in the caller's language, then in English where no term of theirs is; computed on read, never stored, the bodies
// unchanged. A failure leaves the messages without terms: they are read all the same, and the failure is not the
// caller's.
func (s *Service) withTerms(ctx context.Context, c Caller, msgs []Message) {
	if s.Terms == nil || len(msgs) == 0 {
		return
	}
	languages := []string{fallbackLanguage}
	if s.Languages != nil {
		if lang, err := s.Languages.Language(ctx, c.AccountID); err == nil && lang != "" && lang != fallbackLanguage {
			languages = []string{lang, fallbackLanguage}
		}
	}
	var texts []terminology.Text
	var at []int // per text, its message
	for i, m := range msgs {
		if m.Body == "" {
			continue
		}
		for _, lang := range languages {
			texts = append(texts, terminology.Text{Language: lang, Body: m.Body})
			at = append(at, i)
		}
	}
	if len(texts) == 0 {
		return
	}
	spans, err := s.Terms.Tag(ctx, texts)
	if err != nil {
		return
	}
	for k, sp := range spans {
		m := &msgs[at[k]]
		m.Terms = merge(m.Terms, sp)
	}
}

// withTermsOf marks the explained terms of the one message m for the caller.
func (s *Service) withTermsOf(ctx context.Context, c Caller, m *Message) {
	one := []Message{*m}
	s.withTerms(ctx, c, one)
	*m = one[0]
}

// merge adds to kept the spans of more that overlap none of kept, in order of their start.
func merge(kept, more []terminology.Span) []terminology.Span {
	for _, sp := range more {
		if !slices.ContainsFunc(kept, func(k terminology.Span) bool { return sp.Start < k.End && k.Start < sp.End }) {
			kept = append(kept, sp)
		}
	}
	slices.SortFunc(kept, func(a, b terminology.Span) int { return cmp.Compare(a.Start, b.Start) })
	return kept
}
