package composition

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// DraftTimeout bounds the model's drafting, within the API's 30 s request timeout; the target is a draft in
// under 15 s (docs/plans/consult-intelligence.md, "E (SAL-8)").
const DraftTimeout = 25 * time.Second

// maxValue bounds a field's value, drafted or typed.
const maxValue = 10000

var icd10Pattern = regexp.MustCompile(`^[A-Z][0-9]{2}(\.[0-9A-Z]{1,4})?$`)

// Generate is what a drafting asks for: the template, and the language to write in ("" for the template's
// original language).
type Generate struct {
	TemplateID string
	Language   string
}

// Generate drafts the template in.TemplateID for the encounter from its stored transcript so far, for its
// practitioner with the patient's active consent, during or after the call: the model fills the fields, each
// filled field citing the segments it came from; a value it gives without sources, citing a segment the
// transcript does not have, or not of its field's type, is not accepted and the field stays missing, as do the
// fields it leaves out. The draft is stored as the encounter's open draft of the template (a regeneration
// replaces its fields as its next version), audited, and pushed on the encounter's stream. The access is
// checked again after the model's answer, so a revocation meanwhile stores nothing.
//
// It needs the patient's transcription consent too (SAL-12). ErrNotFound, ErrForbidden, ErrConsentRequired,
// ErrTranscriptionConsentRequired, ErrTemplateNotFound and ErrLanguageUnavailable are refusals, audited; ErrDraftingUnavailable when the model fails or takes longer than DraftTimeout.
func (s *Service) Generate(ctx context.Context, c appointment.Caller, encounterID string, in Generate) (Draft, error) {
	d, segments, err := s.prepare(ctx, c, encounterID, in)
	if err != nil {
		return Draft{}, err
	}
	if len(segments) > 0 {
		if err := s.fill(ctx, &d, segments); err != nil {
			return Draft{}, err
		}
	}
	var refused error
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := entry{c: c, interaction: auditevent.CodeCreate}
		enc, err := s.allowedDrafting(ctx, c, encounterID)
		if refused = refusal(err); refused != nil {
			e.reference, e.outcome = encounterRef(enc.ID), auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		if err != nil {
			return err
		}
		saved, created, err := s.Repo.SaveDraft(ctx, d, s.Clock.Now())
		if err != nil {
			return err
		}
		if !created {
			e.interaction = auditevent.CodeUpdate
		}
		d, e.reference, e.org = saved, "Composition/"+saved.ID, enc.OrganizationID
		return s.record(ctx, e)
	})
	if err := wrapErr("generate", err, refused); err != nil {
		return Draft{}, err
	}
	s.publish(ctx, d)
	return d, nil
}

// prepare checks the caller may draft, in one transaction: the encounter, the template and the language, and
// reads the transcript (audited by transcript). It returns the draft with every field missing, and the
// transcript's final segments.
func (s *Service) prepare(ctx context.Context, c appointment.Caller, encounterID string, in Generate) (Draft, []utterance, error) {
	var d Draft
	var segments []utterance
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		enc, err := s.allowedDrafting(ctx, c, encounterID)
		if refused = refusal(err); refused == nil && err == nil {
			refused, err = s.drafted(ctx, &d, enc, in)
		}
		if refused != nil {
			return s.record(ctx, entry{c: c, interaction: auditevent.CodeCreate, reference: encounterRef(enc.ID), outcome: auditevent.OutcomeMinorFailure})
		}
		if err != nil {
			return err
		}
		events, err := s.Transcripts.Export(ctx, c, enc.ID)
		for _, e := range events {
			segments = append(segments, utterance{id: e.SegmentID, speaker: string(e.Speaker), text: e.Original})
		}
		return err
	})
	return d, segments, wrapErr("generate", err, refused)
}

// utterance is a stored segment as the model reads it: the original, as it was said.
type utterance struct{ id, speaker, text string }

// drafted sets d to the empty draft of the template for the encounter in the language asked for, or returns
// the refusal.
func (s *Service) drafted(ctx context.Context, d *Draft, enc encounter.Encounter, in Generate) (refused, failed error) {
	t, err := s.Repo.Template(ctx, in.TemplateID, false)
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		return ErrTemplateNotFound, nil
	case err != nil:
		return nil, err
	}
	lang := in.Language
	if lang == "" {
		lang = t.Language
	}
	title, fields, ok := t.In(lang)
	if !ok {
		return ErrLanguageUnavailable, nil
	}
	*d = Draft{EncounterID: enc.ID, TemplateID: t.ID, Language: lang, Title: title, TypeCode: t.TypeCode, Fields: make([]DraftField, len(fields))}
	for i, f := range fields {
		d.Fields[i] = DraftField{
			Field: Field{Key: f.Key, Missing: true}, Type: f.Type, Label: f.Label, Instruction: f.Instruction, Required: f.Required,
		}
	}
	return nil, nil
}

// fill asks the model to fill d's fields from the segments, outside any transaction, and keeps each value it
// accepts.
func (s *Service) fill(ctx context.Context, d *Draft, segments []utterance) error {
	req := llm.DraftRequest{Lang: d.Language, Fields: make([]llm.FieldSpec, len(d.Fields)), Transcript: make([]llm.Utterance, len(segments))}
	known := make(map[string]bool, len(segments))
	for i, u := range segments {
		req.Transcript[i] = llm.Utterance{ID: u.id, Speaker: u.speaker, Text: u.text}
		known[u.id] = true
	}
	for i, f := range d.Fields {
		req.Fields[i] = llm.FieldSpec{Key: f.Key, Label: f.Label, Instruction: f.Instruction, Type: llm.FieldType(f.Type), Required: f.Required}
	}
	ctx, cancel := context.WithTimeout(ctx, DraftTimeout)
	defer cancel()
	out, err := s.Drafter.Draft(ctx, req)
	if err != nil {
		s.Log.WarnContext(ctx, "drafting failed", "encounter_id", d.EncounterID, "template_id", d.TemplateID, "error", err.Error())
		return ErrDraftingUnavailable
	}
	for i, f := range d.Fields {
		j := slices.IndexFunc(out, func(o llm.DraftField) bool { return o.Key == f.Key })
		if j < 0 || out[j].Missing {
			continue
		}
		value, ok := normalise(f.Type, out[j].Value)
		sources, cited := cite(out[j].SourceIDs, known)
		if ok && cited {
			d.Fields[i].Field = Field{Key: f.Key, Value: value, SourceSegmentIDs: sources}
		}
	}
	return nil
}

// cite returns the sources once each, in order, and whether there is at least one and every one is a segment
// of the transcript.
func cite(ids []string, known map[string]bool) ([]string, bool) {
	var out []string
	for _, id := range ids {
		if !fhir.IsUUID(id) || !known[id] {
			return nil, false
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, len(out) > 0
}

// normalise returns the value as its field's type stores it, and false when it is not one: text trimmed; a
// list's non-blank lines, trimmed; a date as YYYY-MM-DD; an ICD-10 code upper-cased. Blank or longer than
// maxValue, or with a letter the PDF cannot set, is never a value.
func normalise(t FieldType, v string) (string, bool) {
	v = strings.TrimSpace(v)
	switch t {
	case FieldList:
		var items []string
		for line := range strings.SplitSeq(v, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				items = append(items, line)
			}
		}
		v = strings.Join(items, "\n")
	case FieldDate:
		if _, err := time.Parse(time.DateOnly, v); err != nil {
			return "", false
		}
	case FieldCoded:
		if v = strings.ToUpper(v); !icd10Pattern.MatchString(v) {
			return "", false
		}
	}
	return v, v != "" && chars(v) <= maxValue && pdfdoc.Supported(v)
}

// publish pushes the draft's event on its encounter's stream; a failure loses only the live push (the page
// reads the drafts again) and is logged by id.
func (s *Service) publish(ctx context.Context, d Draft) {
	b, err := wire.Encode(MessageType, MessageVersion, d.Event())
	if err == nil {
		err = s.Publisher.Publish(ctx, d.EncounterID, b)
	}
	if err != nil {
		s.Log.WarnContext(ctx, "draft publish failed", "draft_id", d.ID, "encounter_id", d.EncounterID, "error", err.Error())
	}
}
