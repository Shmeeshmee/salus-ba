package transcript

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/llm"
)

// defaultTranslateBudget is how long a final segment's translation may take, its glossary and a retry included,
// before the segment is published untranslated and flagged: the translated line is to be on screen under 2 s
// after its final result (SAL-5), and the rest of the path takes milliseconds (PROGRESS, SAL-4 and SAL-5).
const defaultTranslateBudget = 1500 * time.Millisecond

// translateAttempts is how often a segment is sent to the translator within its budget: once, and once more
// after a failure.
const translateAttempts = 2

// errEmptyTranslation is a translator's answer without words.
var errEmptyTranslation = errors.New("transcript: the translation is empty")

// Glossary gives the terms said in a line, each with its label in the target language (terminology.Service).
// Both languages are tags of the enum language.
type Glossary interface {
	Glossary(ctx context.Context, text, from, to string) ([]terminology.GlossaryEntry, error)
}

// listener is who reads what speaker says: the other participant.
func listener(speaker Speaker) Speaker {
	if speaker == SpeakerPatient {
		return SpeakerPractitioner
	}
	return SpeakerPatient
}

// translated reports whether what speaker says is translated: when the listener reads another language. A
// patient's line goes into the practitioner's language (the clinic's), a practitioner's into the patient's.
func (c call) translated(speaker Speaker) bool {
	return c.languages[speaker] != c.languages[listener(speaker)]
}

// translate translates the final segment e into its listener's language, with the terms the terminology finds in
// it, within TranslateBudget: once, and once more after a failure. When both fail or the budget is spent, e keeps
// its original alone and is flagged TranslationFailed: a line is never dropped for its translation. Without its
// glossary (the terminology failed) the line is translated without one. Failures are logged by the session, the
// speaker's role and the error, never the line or its translation.
func (s *Live) translate(ctx context.Context, c call, e *Event) {
	ctx, cancel := context.WithTimeout(ctx, s.TranslateBudget)
	defer cancel()
	req := llm.TranslateRequest{Text: e.Original, From: c.languages[e.Speaker], To: c.languages[listener(e.Speaker)]}
	terms, err := s.Glossary.Glossary(ctx, req.Text, req.From, req.To)
	if err != nil {
		s.Log.WarnContext(ctx, "glossary failed", "session_id", c.sessionID, "speaker", string(e.Speaker), "error", err.Error())
	}
	for _, t := range terms {
		req.Glossary = append(req.Glossary, llm.Term{Source: t.Source, Target: t.Target})
	}
	for range translateAttempts {
		var out string
		out, err = s.Translator.Translate(ctx, req)
		if err == nil && strings.TrimSpace(out) != "" {
			e.Translation = out
			return
		}
		if err == nil {
			err = errEmptyTranslation
		}
		if ctx.Err() != nil {
			break
		}
	}
	e.TranslationFailed = true
	s.Log.WarnContext(ctx, "translation failed", "session_id", c.sessionID, "speaker", string(e.Speaker), "error", err.Error())
}
