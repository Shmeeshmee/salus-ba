// Package api_test keeps the contracts in this folder in step with the Go types they describe.
package api_test

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// schema is the part of an OpenAPI 3.1 schema object these tests read.
type schema struct {
	Const      any               `yaml:"const"`
	Items      *schema           `yaml:"items"`
	Properties map[string]schema `yaml:"properties"`
	Ref        string            `yaml:"$ref"`
	Type       string            `yaml:"type"`
	Required   []string          `yaml:"required"`
	Enum       []string          `yaml:"enum"`
	OneOf      []schema          `yaml:"oneOf"`
}

func schemas(t *testing.T) map[string]schema {
	t.Helper()
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]schema `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Components.Schemas
}

// jsonFields returns the JSON names of a struct's fields, and those that are always encoded (no omitempty).
func jsonFields(v any) (all, required []string) {
	typ := reflect.TypeOf(v)
	for i := range typ.NumField() {
		name, opts, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		all = append(all, name)
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	slices.Sort(all)
	slices.Sort(required)
	return all, required
}

// The live events' schemas (generated into web/src/api/schema.d.ts by make gen-clients) have the Go types'
// fields, required exactly when Go always encodes them, and the Go types' enums.
func TestLiveEvents_MatchTheirGoTypes(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, tt := range map[string]struct {
		value any
		enums map[string][]string
	}{
		"TranscriptEvent": {transcript.Event{}, map[string][]string{
			"speaker": {string(transcript.SpeakerPractitioner), string(transcript.SpeakerPatient)},
		}},
		"SuggestionEvent": {suggestion.Event{}, map[string][]string{
			"type": {string(suggestion.TypeDiagnosis), string(suggestion.TypeQuestion), string(suggestion.TypeRedFlag),
				string(suggestion.TypeTerm)},
			"status": {string(suggestion.StatusShown), string(suggestion.StatusDismissed), string(suggestion.StatusPinned)},
		}},
		"DraftEvent": {composition.Event{}, map[string][]string{
			"status": {string(composition.StatusPreliminary), string(composition.StatusFinal), string(composition.StatusEnteredInError)},
		}},
		"DraftField": {composition.Field{}, nil},
	} {
		sch, ok := s[name]
		if !ok {
			t.Errorf("no schema %s", name)
			continue
		}
		all, required := jsonFields(tt.value)
		var props []string
		for p := range sch.Properties {
			props = append(props, p)
		}
		slices.Sort(props)
		req := slices.Sorted(slices.Values(sch.Required))
		if !slices.Equal(props, all) {
			t.Errorf("%s: properties %v, want the Go fields %v", name, props, all)
		}
		if !slices.Equal(req, required) {
			t.Errorf("%s: required %v, want %v", name, req, required)
		}
		for prop, want := range tt.enums {
			if got := sch.Properties[prop].Enum; !slices.Equal(got, want) {
				t.Errorf("%s.%s: enum %v, want %v", name, prop, got, want)
			}
		}
	}
	if items := s["DraftEvent"].Properties["fields"].Items; items == nil || items.Ref != "#/components/schemas/DraftField" {
		t.Errorf("DraftEvent.fields items = %+v, want a DraftField reference", items)
	}
}

// Each live message is a wire.Envelope of its event, with the type and version its Go package sends, and
// ConsultMessage is one of them.
func TestConsultMessages_AreTheEnvelopesTheGoPackagesSend(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, want := range map[string]struct {
		typ     string
		payload string
		version int
	}{
		"TranscriptMessage": {transcript.MessageType, "TranscriptEvent", transcript.MessageVersion},
		"SuggestionMessage": {suggestion.MessageType, "SuggestionEvent", suggestion.MessageVersion},
		"DraftMessage":      {composition.MessageType, "DraftEvent", composition.MessageVersion},
	} {
		m := s[name]
		if !slices.Equal(slices.Sorted(slices.Values(m.Required)), []string{"payload", "type", "version"}) {
			t.Errorf("%s: required %v, want payload, type and version", name, m.Required)
		}
		if got := m.Properties["type"].Const; got != want.typ {
			t.Errorf("%s: type const %v, want %q", name, got, want.typ)
		}
		if got := m.Properties["version"].Const; got != want.version {
			t.Errorf("%s: version const %v, want %d", name, got, want.version)
		}
		if got := m.Properties["payload"].Ref; got != "#/components/schemas/"+want.payload {
			t.Errorf("%s: payload %q, want a %s reference", name, got, want.payload)
		}
	}
	var refs []string
	for _, o := range s["ConsultMessage"].OneOf {
		refs = append(refs, o.Ref)
	}
	slices.Sort(refs)
	if want := []string{"#/components/schemas/DraftMessage", "#/components/schemas/SuggestionMessage",
		"#/components/schemas/TranscriptMessage"}; !slices.Equal(refs, want) {
		t.Errorf("ConsultMessage is one of %v, want %v", refs, want)
	}
}
