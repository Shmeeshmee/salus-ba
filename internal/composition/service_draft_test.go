package composition_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/pkg/llm"
)

func generate(t *testing.T, f *fixture, c appointment.Caller, templateID, lang string) (composition.Draft, error) {
	t.Helper()
	return f.svc.Generate(t.Context(), c, encA, composition.Generate{TemplateID: templateID, Language: lang})
}

// The drafter fills the template's fields from the encounter's stored transcript; a field it fills cites the
// segments it came from, the others are missing. The draft keeps the template's texts, is stored as the open
// draft of the template, audited, and pushed on the encounter's stream as a draft event.
func TestGenerate_FillsTheFieldsFromTheTranscriptCitingTheirSegments(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	f.drafter.Fields = filled()

	d, err := generate(t, f, pract, tpl.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != composition.StatusPreliminary || d.EncounterID != encA || d.TemplateID != tpl.ID || d.Language != "bs" ||
		d.Title != tpl.Title || d.TypeCode != "11488-4" || d.VersionID != 1 || len(d.Fields) != 4 {
		t.Fatalf("Generate() = %+v", d)
	}
	want := []composition.Field{
		{Key: "findings", Value: "Placeholder findings", SourceSegmentIDs: []string{seg1}},
		{Key: "tests", Value: "Placeholder test one\nPlaceholder test two", SourceSegmentIDs: []string{seg2}},
		{Key: "control", Missing: true},
		{Key: "diagnosis", Missing: true},
	}
	for i, w := range want {
		got := d.Fields[i]
		if got.Key != w.Key || got.Value != w.Value || got.Missing != w.Missing || !slices.Equal(got.SourceSegmentIDs, w.SourceSegmentIDs) ||
			got.Label != tpl.Fields[i].Label || got.Type != tpl.Fields[i].Type || got.Required != tpl.Fields[i].Required {
			t.Errorf("field %d = %+v, want %+v with the template's texts", i, got, w)
		}
	}
	calls := f.drafter.Calls()
	if len(calls) != 1 || calls[0].Lang != "bs" || len(calls[0].Transcript) != 2 ||
		calls[0].Transcript[0] != (llm.Utterance{ID: seg1, Speaker: "patient", Text: "Placeholder complaint"}) ||
		calls[0].Fields[0] != (llm.FieldSpec{Key: "findings", Label: "Placeholder findings", Instruction: "Placeholder instruction", Type: llm.FieldText, Required: true}) ||
		calls[0].Fields[3].Type != llm.FieldCoded {
		t.Errorf("drafter calls = %+v", calls)
	}
	if got := f.codes(); !slices.Equal(got, []string{"create 0 Composition/" + d.ID}) {
		t.Errorf("audit = %q", got)
	}
	if len(f.publisher.envelopes) != 1 || f.publisher.streams[0] != encA || f.publisher.envelopes[0].Type != "draft" || f.publisher.envelopes[0].Version != 1 {
		t.Fatalf("published %+v on %v", f.publisher.envelopes, f.publisher.streams)
	}
	var e composition.Event
	if err := json.Unmarshal(f.publisher.envelopes[0].Payload, &e); err != nil || e.Validate() != nil || e.DraftID != d.ID || e.VersionID != 1 || len(e.Fields) != 4 {
		t.Errorf("published event = %+v, %v", e, err)
	}
}

// The model's output is checked, never trusted: a value without sources, one citing a line the transcript
// does not have, a date that is no date, a code that is no ICD-10 code, or a blank list, is missing. A list
// keeps its non-blank lines, a code is upper-cased, and a source cited twice is kept once.
func TestGenerate_ADraftedValueWithoutValidSourcesIsMissing(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		field llm.DraftField
		key   string
		want  composition.Field // Key is filled in
	}{
		"no sources":       {llm.DraftField{Value: "Placeholder"}, "findings", composition.Field{Missing: true}},
		"an unknown line":  {llm.DraftField{Value: "Placeholder", SourceIDs: []string{seg1, unknownID}}, "findings", composition.Field{Missing: true}},
		"not a UUID":       {llm.DraftField{Value: "Placeholder", SourceIDs: []string{"s1"}}, "findings", composition.Field{Missing: true}},
		"a blank value":    {llm.DraftField{Value: " ", SourceIDs: []string{seg1}}, "findings", composition.Field{Missing: true}},
		"marked missing":   {llm.DraftField{Value: "Placeholder", SourceIDs: []string{seg1}, Missing: true}, "findings", composition.Field{Missing: true}},
		"a long value":     {llm.DraftField{Value: strings.Repeat("x", 10001), SourceIDs: []string{seg1}}, "findings", composition.Field{Missing: true}},
		"no date":          {llm.DraftField{Value: "next week", SourceIDs: []string{seg1}}, "control", composition.Field{Missing: true}},
		"a date":           {llm.DraftField{Value: " 2030-02-01 ", SourceIDs: []string{seg1}}, "control", composition.Field{Value: "2030-02-01", SourceSegmentIDs: []string{seg1}}},
		"no code":          {llm.DraftField{Value: "a cold", SourceIDs: []string{seg1}}, "diagnosis", composition.Field{Missing: true}},
		"a code":           {llm.DraftField{Value: "j06.9", SourceIDs: []string{seg1, seg2, seg1}}, "diagnosis", composition.Field{Value: "J06.9", SourceSegmentIDs: []string{seg1, seg2}}},
		"a blank list":     {llm.DraftField{Value: "\n \n", SourceIDs: []string{seg2}}, "tests", composition.Field{Missing: true}},
		"a list with gaps": {llm.DraftField{Value: " one \n\n two", SourceIDs: []string{seg2}}, "tests", composition.Field{Value: "one\ntwo", SourceSegmentIDs: []string{seg2}}},
	} {
		f := newFixture(t)
		tpl := f.template(t)
		f.drafter.Fields = map[string]llm.DraftField{tt.key: tt.field}
		d, err := generate(t, f, pract, tpl.ID, "")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		i := slices.IndexFunc(d.Fields, func(x composition.DraftField) bool { return x.Key == tt.key })
		got, want := d.Fields[i].Field, tt.want
		want.Key = tt.key
		if got.Value != want.Value || got.Missing != want.Missing || !slices.Equal(got.SourceSegmentIDs, want.SourceSegmentIDs) {
			t.Errorf("%s: field = %+v, want %+v", name, got, want)
		}
	}
}

// A drafter that answers fields out of order, twice, or not at all is mapped by key: each template field
// once, in the template's order, missing when the drafter has none.
func TestGenerate_MapsTheDraftersFieldsByKey(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	d, err := f.svc.Generate(t.Context(), pract, encA, composition.Generate{TemplateID: tpl.ID})
	if err != nil {
		t.Fatal(err)
	}
	for i, field := range d.Fields {
		if field.Key != tpl.Fields[i].Key || !field.Missing {
			t.Errorf("field %d = %+v, want %s missing", i, field, tpl.Fields[i].Key)
		}
	}
}

// A regeneration, during or after the call, replaces the open draft of the template: the same draft, its next
// version, audited as an update.
func TestGenerate_RegeneratingReplacesTheOpenDraft(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	first, err := generate(t, f, pract, tpl.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	f.drafter.Fields = filled()
	second, err := generate(t, f, pract, tpl.ID, "")
	if err != nil || second.ID != first.ID || second.VersionID != 2 || second.Fields[0].Missing {
		t.Fatalf("Generate() again = %+v, %v; want the same draft, version 2, filled", second, err)
	}
	if got := f.codes(); !slices.Equal(got, []string{"create 0 Composition/" + first.ID, "update 0 Composition/" + first.ID}) {
		t.Errorf("audit = %q", got)
	}
}

// The draft is written in the language asked for, the template's original or one of its translations: the
// model reads that language's labels, and the draft keeps them. Another language is refused, audited.
func TestGenerate_WritesInALanguageTheTemplateHas(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	d, err := generate(t, f, pract, tpl.ID, "en")
	if err != nil || d.Language != "en" || d.Title != "Placeholder report" || f.drafter.Calls()[0].Lang != "en" {
		t.Fatalf("Generate(en) = %+v, %v", d, err)
	}
	if _, err := generate(t, f, pract, tpl.ID, "de"); !errors.Is(err, composition.ErrLanguageUnavailable) {
		t.Errorf("Generate(de) = %v, want ErrLanguageUnavailable", err)
	}
	if _, err := generate(t, f, pract, unknownID, ""); !errors.Is(err, composition.ErrTemplateNotFound) {
		t.Errorf("Generate(unknown template) = %v, want ErrTemplateNotFound", err)
	}
	if got := f.codes(); len(got) != 3 || got[1] != "create 4 Encounter/"+encA || got[2] != "create 4 Encounter/"+encA {
		t.Errorf("audit = %q", got)
	}
}

// Only the encounter's practitioner, with the patient's active consent, drafts: the clinic's admin, another
// practitioner and the patient are refused, an encounter they may not see is not found, and after a
// revocation nothing is drafted. Every refusal is audited and nothing reaches the model or the stream.
func TestGenerate_OnlyTheEncountersPractitionerWithConsentDrafts(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		want error
		id   string
		ref  string
		c    appointment.Caller
	}{
		"admin":                {composition.ErrForbidden, encA, "Encounter/" + encA, admin},
		"another practitioner": {composition.ErrForbidden, encA, "Encounter/" + encA, pract2},
		"the patient":          {composition.ErrForbidden, encA, "Encounter/" + encA, me},
		"an unknown encounter": {composition.ErrNotFound, unknownID, "", pract},
	} {
		f := newFixture(t)
		tpl := f.template(t)
		_, err := f.svc.Generate(t.Context(), tt.c, tt.id, composition.Generate{TemplateID: tpl.ID})
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Generate() = %v, want %v", name, err, tt.want)
		}
		if got := f.codes(); !slices.Equal(got, []string{"create 4 " + tt.ref}) || len(f.drafter.Calls()) != 0 || len(f.publisher.envelopes) != 0 {
			t.Errorf("%s: audit %q, %d drafter calls, %d published", name, got, len(f.drafter.Calls()), len(f.publisher.envelopes))
		}
	}
	f := newFixture(t)
	tpl := f.template(t)
	f.encounters.attendances[practAcc] = []appointment.Attendance{attendance(true, false, false)}
	if _, err := generate(t, f, pract, tpl.ID, ""); !errors.Is(err, composition.ErrConsentRequired) {
		t.Errorf("Generate(revoked) = %v, want ErrConsentRequired", err)
	}
}

// A revocation while the model drafts is seen before anything is stored: the access is checked again.
func TestGenerate_ARevocationDuringTheDraftingStoresNothing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	f.encounters.attendances[practAcc] = []appointment.Attendance{attendance(true, false, true), attendance(true, false, false)}
	if _, err := generate(t, f, pract, tpl.ID, ""); !errors.Is(err, composition.ErrConsentRequired) {
		t.Fatalf("Generate() = %v, want ErrConsentRequired", err)
	}
	if len(f.repo.drafts) != 0 || len(f.publisher.envelopes) != 0 || !slices.Equal(f.codes(), []string{"create 4 Encounter/" + encA}) {
		t.Errorf("drafts %d, published %d, audit %q", len(f.repo.drafts), len(f.publisher.envelopes), f.codes())
	}
}

// Without a stored transcript there is nothing to draft from: every field is missing, the model is not asked.
func TestGenerate_WithoutATranscriptEveryFieldIsMissing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	f.transcripts.segments = nil
	f.drafter.Fields = filled()
	d, err := generate(t, f, pract, tpl.ID, "")
	if err != nil || len(f.drafter.Calls()) != 0 {
		t.Fatalf("Generate() = %v, %d drafter calls; want none", err, len(f.drafter.Calls()))
	}
	for _, field := range d.Fields {
		if !field.Missing {
			t.Errorf("field %+v, want missing", field)
		}
	}
}

// A failing model stores nothing and says drafting is unavailable; a failing store is a failure; a failing
// stream loses only the live push (logged without a value): the draft is stored and returned.
func TestGenerate_Failures(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tpl := f.template(t)
	f.drafter.Err = errBackend
	if _, err := generate(t, f, pract, tpl.ID, ""); !errors.Is(err, composition.ErrDraftingUnavailable) || len(f.repo.drafts) != 0 {
		t.Errorf("Generate(model down) = %v, %d drafts; want ErrDraftingUnavailable, none", err, len(f.repo.drafts))
	}
	f.drafter.Err, f.transcripts.err = nil, errBackend
	if _, err := generate(t, f, pract, tpl.ID, ""); !errors.Is(err, errBackend) {
		t.Errorf("Generate(transcript down) = %v, want the failure", err)
	}
	f.transcripts.err, f.drafter.Fields = nil, filled()
	f.publisher.err = errBackend
	d, err := generate(t, f, pract, tpl.ID, "")
	if err != nil || len(f.repo.drafts) != 1 {
		t.Fatalf("Generate(stream down) = %v, %d drafts; want the draft", err, len(f.repo.drafts))
	}
	if log := f.logs.String(); !strings.Contains(log, d.ID) || strings.Contains(log, "Placeholder") {
		t.Errorf("log = %q, want the draft's id and no value", log)
	}
	f.encounters.err = errBackend
	if _, err := generate(t, f, pract, tpl.ID, ""); !errors.Is(err, errBackend) {
		t.Errorf("Generate(encounters down) = %v, want the failure", err)
	}
}
