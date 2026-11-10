package transcript

import (
	"context"
	"strings"
	"unicode"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Tagger marks the terms with a plain-language explanation in texts, each in its app language (terminology.Service,
// SAL-13). The texts are what was said: never logged, stored or put into an error.
type Tagger interface {
	Tag(ctx context.Context, texts []terminology.Text) ([][]terminology.Span, error)
}

// tag marks the explained terms of the final line e, its original in the speaker's language and its translation in
// the listener's. A failure is logged by the session, the speaker's role and the error, never the line, and e is
// published without terms: the tooltip is an overlay a line never waits for.
func (s *Live) tag(ctx context.Context, c call, e *Event) {
	if s.Terms == nil {
		return
	}
	texts := []terminology.Text{{Language: c.languages[e.Speaker], Body: e.Original}}
	if e.Translation != "" {
		texts = append(texts, terminology.Text{Language: c.languages[listener(e.Speaker)], Body: e.Translation})
	}
	spans, err := s.Terms.Tag(ctx, texts)
	if err != nil {
		s.Log.WarnContext(ctx, "term tagging failed", "session_id", c.sessionID, "speaker", string(e.Speaker), "error", err.Error())
		return
	}
	e.Terms = terms(spans)
}

// terms are the spans of a line's original (first) and translation (second, when there is one); nil for none.
func terms(spans [][]terminology.Span) *Terms {
	var t Terms
	if len(spans) > 0 {
		t.Original = spans[0]
	}
	if len(spans) > 1 {
		t.Translation = spans[1]
	}
	if t.Original == nil && t.Translation == nil {
		return nil
	}
	return &t
}

// tagStored marks the explained terms of the stored lines in place, in one call: each original in its speaker's
// language, each translation in the other speaker's, both from the stored tags (textLanguage); a translation whose
// language is not known (the other speaker said nothing) or a text in no app language is not tagged. A failure leaves
// the lines without terms: the transcript is read all the same, and the failure is not the caller's.
func (s *Service) tagStored(ctx context.Context, lines []Event) {
	if s.Terms == nil || len(lines) == 0 {
		return
	}
	tags := map[Speaker]string{}
	for _, l := range lines {
		if _, ok := tags[l.Speaker]; !ok {
			tags[l.Speaker] = l.Lang
		}
	}
	var texts []terminology.Text
	var at [][2]int // per text, the line and 0 for its original, 1 for its translation
	add := func(i, which int, tag, body string) {
		if lang := textLanguage(tag, body); lang != "" && body != "" {
			texts = append(texts, terminology.Text{Language: lang, Body: body})
			at = append(at, [2]int{i, which})
		}
	}
	for i, l := range lines {
		add(i, 0, l.Lang, l.Original)
		if other, ok := tags[listener(l.Speaker)]; ok {
			add(i, 1, other, l.Translation)
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
		if len(sp) == 0 {
			continue
		}
		l := &lines[at[k][0]]
		if l.Terms == nil {
			l.Terms = &Terms{}
		}
		if at[k][1] == 0 {
			l.Terms.Original = sp
		} else {
			l.Terms.Translation = sp
		}
	}
}

// textLanguage is the app language of body, written in the language of the BCP 47 tag (an app language, or the
// speech locale a line was recognised in): the tag itself when it is an app language, else its primary language
// when the app has it, Serbian with the script of body's letters; "" for a language the app does not have.
func textLanguage(tag, body string) string {
	if _, ok := speechLocales[tag]; ok {
		return tag
	}
	primary, _, _ := strings.Cut(tag, "-")
	if primary == "sr" {
		for _, r := range body {
			if unicode.Is(unicode.Cyrillic, r) {
				return "sr-Cyrl"
			}
		}
		return "sr-Latn"
	}
	if _, ok := speechLocales[primary]; ok {
		return primary
	}
	return ""
}
