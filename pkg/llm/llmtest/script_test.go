package llmtest_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
)

// conversation is a scripted conversation of two files, as test/testdata/consult holds them.
func conversation(t *testing.T) *llmtest.Script {
	t.Helper()
	s, err := llmtest.LoadScript(fstest.MapFS{
		"de.json": {Data: []byte(`{
			"about": "synthetic",
			"lines": [
				{"speaker": "patient", "tone": 650, "lang": "de", "text": "Platzhalter eins", "translations": {"bs": "Primjer jedan"}},
				{"speaker": "practitioner", "tone": 700, "lang": "bs", "text": "Primjer dva", "translations": {"de": "Platzhalter zwei"}}
			],
			"suggestions": [{"after": "Platzhalter eins", "type": "question", "text": "Primjer prijedloga", "confidence": 0.8}],
			"fields": [
				{"key": "complaint", "label": "Primjer: tegobe", "required": true, "from": "Platzhalter eins", "value": "Primjer vrijednosti"},
				{"key": "plan", "label": "Primjer: plan"}
			]
		}`)},
		"tr.json": {Data: []byte(`{
			"lines": [{"tone": 750, "lang": "tr", "text": "Yer tutucu", "translations": {"bs": "Primjer tri"}}],
			"fields": [{"key": "complaint", "from": "Yer tutucu", "value": "Primjer četiri"}]
		}`)},
		"notes.txt": {Data: []byte("not a script")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTranslator_GivesTheScriptsTranslationOfALineAndMarksAnyOtherText(t *testing.T) {
	t.Parallel()
	tr := &llmtest.Translator{Script: conversation(t)}
	for _, tc := range []struct{ text, from, to, want string }{
		{"Platzhalter eins", "de", "bs", "Primjer jedan"},
		{"Primjer dva", "bs", "de", "Platzhalter zwei"},
		{"Yer tutucu", "tr", "bs", "Primjer tri"},
		{"Platzhalter eins", "de", "hr", "[hr] Platzhalter eins"}, // no translation into hr
		{"Platzhalter eins", "tr", "bs", "[bs] Platzhalter eins"}, // not a line in tr
		{"Placeholder caption", "de", "bs", "[bs] Placeholder caption"},
	} {
		got, err := tr.Translate(t.Context(), llm.TranslateRequest{Text: tc.text, From: tc.from, To: tc.to})
		if err != nil || got != tc.want {
			t.Errorf("Translate(%q, %s to %s) = %q, %v; want %q", tc.text, tc.from, tc.to, got, err, tc.want)
		}
	}
}

func TestSuggester_SuggestsTheScriptsSuggestionsOnceTheirLineIsInTheTranscript(t *testing.T) {
	t.Parallel()
	s := &llmtest.Suggester{Script: conversation(t), Suggestions: []llm.Suggestion{{Kind: "term", Text: "fixed", SourceIDs: []string{"s0"}, Confidence: 0.5}}}
	before := []llm.Utterance{{ID: "s0", Speaker: "patient", Text: "Placeholder caption"}}
	got, err := s.Suggest(t.Context(), llm.SuggestRequest{Lang: "bs", Transcript: before})
	if err != nil || len(got) != 1 || got[0].Text != "fixed" {
		t.Errorf("Suggest() before the line = %+v, %v; want only Suggestions", got, err)
	}
	after := append(before, llm.Utterance{ID: "s1", Speaker: "patient", Text: "Platzhalter eins"}, llm.Utterance{ID: "s2", Speaker: "practitioner", Text: "Primjer dva"})
	got, err = s.Suggest(t.Context(), llm.SuggestRequest{Lang: "bs", Transcript: after})
	want := []llm.Suggestion{
		{Kind: "term", Text: "fixed", SourceIDs: []string{"s0"}, Confidence: 0.5},
		{Kind: "question", Text: "Primjer prijedloga", SourceIDs: []string{"s1"}, Confidence: 0.8},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Suggest() after the line = %+v, %v; want %+v", got, err, want)
	}
}

func TestDrafter_FillsAScriptsFieldFromItsLineAndLeavesTheRestMissing(t *testing.T) {
	t.Parallel()
	d := &llmtest.Drafter{Script: conversation(t)}
	fields := []llm.FieldSpec{{Key: "complaint", Required: true}, {Key: "plan"}, {Key: "other"}}
	got, err := d.Draft(t.Context(), llm.DraftRequest{Lang: "bs", Fields: fields, Transcript: []llm.Utterance{
		{ID: "s0", Speaker: "patient", Text: "Placeholder caption"}, {ID: "s1", Speaker: "patient", Text: "Platzhalter eins"},
	}})
	want := []llm.DraftField{
		{Key: "complaint", Value: "Primjer vrijednosti", SourceIDs: []string{"s1"}},
		{Key: "plan", Missing: true},
		{Key: "other", Missing: true},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Draft() = %+v, %v; want %+v", got, err, want)
	}
	got, _ = d.Draft(t.Context(), llm.DraftRequest{Lang: "bs", Fields: fields[:1]})
	if want := []llm.DraftField{{Key: "complaint", Missing: true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Draft() without the line = %+v; want %+v", got, want)
	}
	// Another conversation's field of the same key, from its own line.
	got, _ = d.Draft(t.Context(), llm.DraftRequest{Lang: "bs", Fields: fields[:1], Transcript: []llm.Utterance{{ID: "t1", Speaker: "patient", Text: "Yer tutucu"}}})
	if want := []llm.DraftField{{Key: "complaint", Value: "Primjer četiri", SourceIDs: []string{"t1"}}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Draft() of the other conversation = %+v; want %+v", got, want)
	}
}

func TestLoadScript_RefusesAScriptItCannotPlay(t *testing.T) {
	t.Parallel()
	for name, files := range map[string]fstest.MapFS{
		"no file":                 {},
		"not JSON":                {"de.json": {Data: []byte("{")}},
		"a suggestion of no line": {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins"}], "suggestions": [{"after": "Zwei", "type": "question", "text": "x", "confidence": 0.5}]}`)}},
		"a suggestion of no text": {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins"}], "suggestions": [{"after": "Eins", "type": "question", "confidence": 0.5}]}`)}},
		"a field from no line":    {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins"}], "fields": [{"key": "k", "from": "Zwei", "value": "x"}]}`)}},
		"a field without its key": {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins"}], "fields": [{"from": "Eins", "value": "x"}]}`)}},
		"a filled field no value": {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins"}], "fields": [{"key": "k", "from": "Eins"}]}`)}},
		"two translations of one": {"de.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins", "translations": {"bs": "a"}}]}`)}, "at.json": {Data: []byte(`{"lines": [{"lang": "de", "text": "Eins", "translations": {"bs": "b"}}]}`)}},
		"no line":                 {"de.json": {Data: []byte(`{"lines": []}`)}},
	} {
		if got, err := llmtest.LoadScript(files); err == nil {
			t.Errorf("%s: LoadScript() = %+v, want an error", name, got)
		}
	}
}

// The end-to-end journeys' conversations (make e2e's FAKE_ENGINE_SCRIPT) load, so a broken fixture fails here
// first.
func TestLoadScript_ReadsTheJourneysConversations(t *testing.T) {
	t.Parallel()
	if _, err := llmtest.LoadScript(os.DirFS(filepath.Join("..", "..", "..", "test", "testdata", "consult"))); err != nil {
		t.Errorf("LoadScript(test/testdata/consult): %v", err)
	}
}

// A command's fake model takes its script from FAKE_ENGINE_SCRIPT's directory, or has none when it is unset.
func TestLoadScriptDir_ReadsTheDirectoryOrNothing(t *testing.T) {
	t.Parallel()
	if s, err := llmtest.LoadScriptDir(""); s != nil || err != nil {
		t.Errorf(`LoadScriptDir("") = %v, %v; want no script`, s, err)
	}
	if s, err := llmtest.LoadScriptDir(filepath.Join("..", "..", "..", "test", "testdata", "consult")); s == nil || err != nil {
		t.Errorf("LoadScriptDir(test/testdata/consult) = %v, %v; want the script", s, err)
	}
	if _, err := llmtest.LoadScriptDir(t.TempDir()); err == nil {
		t.Error("LoadScriptDir of an empty directory: no error")
	}
}
