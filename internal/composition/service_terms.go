package composition

import (
	"context"
	"strings"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Tagger marks the terms with a plain-language explanation in texts, each in its app language (terminology.Service,
// SAL-13). The texts are a document's: never logged, stored or put into an error.
type Tagger interface {
	Tag(ctx context.Context, texts []terminology.Text) ([][]terminology.Span, error)
}

// SectionTerms are the explained terms of one field of a document, by its key: per line of its value, as the
// Composition's narrative has them (a paragraph or list item per line; none for a missing field).
type SectionTerms struct {
	Key   string
	Lines [][]terminology.Span
}

// Terms returns the explained terms of the draft id, field by field in its order, line by line, tagged in the
// language it was written in (SAL-13): for the tooltip on the patient's and the practitioner's screens. Read under
// Composition's rule and audited as its read, with its errors. An overlay: the document is not changed.
func (s *Service) Terms(ctx context.Context, c appointment.Caller, id string) ([]SectionTerms, error) {
	doc, err := s.Composition(ctx, c, id)
	if err != nil {
		return nil, err
	}
	d := doc.Draft
	out := make([]SectionTerms, len(d.Fields))
	var texts []terminology.Text
	for i, f := range d.Fields {
		out[i] = SectionTerms{Key: f.Key, Lines: [][]terminology.Span{}}
		if f.Missing {
			continue
		}
		for _, line := range strings.Split(f.Value, "\n") {
			texts = append(texts, terminology.Text{Language: d.Language, Body: line})
		}
	}
	spans, err := s.Tagger.Tag(ctx, texts)
	if err != nil {
		return nil, wrapErr("terms", err, nil)
	}
	k := 0
	for i, f := range d.Fields {
		if f.Missing {
			continue
		}
		for range strings.Count(f.Value, "\n") + 1 {
			out[i].Lines = append(out[i].Lines, spans[k])
			k++
		}
	}
	return out, nil
}
