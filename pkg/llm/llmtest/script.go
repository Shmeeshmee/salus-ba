package llmtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/Shmeeshmee/salus/pkg/llm"
)

// Script is what the fakes answer in scripted conversations (LoadScript): the translation of each line, the
// suggestions given once a line is in the transcript, and the fields filled from a line. The end-to-end journeys
// play such a conversation through the whole pipeline (SAL-10); stttest.LoadScript reads the same files for the
// lines' tones. Anything not in the script gets the fakes' plain answers.
type Script struct {
	translations map[[3]string]string // by the line's language, its text and the target language
	suggestions  []scriptSuggestion
	fields       []scriptField
}

// scriptSuggestion is suggested once the line whose text is After is in the transcript, citing it.
type scriptSuggestion struct {
	After      string
	Type       string
	Text       string
	Confidence float32
}

// scriptField is a document field: filled with Value, citing the line whose text is From, once that line is in the
// transcript. A field without From is never filled. The journeys also read a field's label and required, for
// their template.
type scriptField struct {
	Key   string
	From  string
	Value string
}

// LoadScript reads every .json file in fsys, a scripted conversation each:
//
//	{"lines": [{"lang": "de", "text": "...", "translations": {"bs": "..."}, "tone": 650, "speaker": "patient"}],
//	 "suggestions": [{"after": "<a line's text>", "type": "question", "text": "...", "confidence": 0.8}],
//	 "fields": [{"key": "complaint", "from": "<a line's text>", "value": "..."}]}
//
// A suggestion or a field names a line of its own file; a line has one translation per language.
func LoadScript(fsys fs.FS) (*Script, error) {
	names, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, err
	}
	s := &Script{translations: map[[3]string]string{}}
	lines := 0
	for _, name := range names {
		n, err := s.load(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("llmtest: %s: %w", name, err)
		}
		lines += n
	}
	if lines == 0 {
		return nil, errors.New("llmtest: the script has no line")
	}
	return s, nil
}

// LoadScriptDir is LoadScript of the directory dir (a command's FAKE_ENGINE_SCRIPT), or no script (nil) when dir is
// empty.
func LoadScriptDir(dir string) (*Script, error) {
	if dir == "" {
		return nil, nil
	}
	return LoadScript(os.DirFS(dir))
}

// load adds one conversation to the script and returns its number of lines.
func (s *Script) load(fsys fs.FS, name string) (int, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return 0, err
	}
	var file struct {
		Lines []struct {
			Translations map[string]string
			Lang         string
			Text         string
		}
		Suggestions []scriptSuggestion
		Fields      []scriptField
	}
	if err := json.Unmarshal(b, &file); err != nil {
		return 0, err
	}
	said := map[string]bool{}
	for _, l := range file.Lines {
		said[l.Text] = true
		for to, text := range l.Translations {
			key := [3]string{l.Lang, l.Text, to}
			if _, ok := s.translations[key]; ok {
				return 0, fmt.Errorf("a second translation into %s of a line in %s", to, l.Lang)
			}
			s.translations[key] = text
		}
	}
	for _, sg := range file.Suggestions {
		if !said[sg.After] || strings.TrimSpace(sg.Text) == "" {
			return 0, errors.New("a suggestion without its text or after no line of the conversation")
		}
	}
	for _, f := range file.Fields {
		if f.Key == "" || (f.From != "" && (!said[f.From] || strings.TrimSpace(f.Value) == "")) {
			return 0, errors.New("a field without its key, or filled without a value or from no line of the conversation")
		}
	}
	s.suggestions = append(s.suggestions, file.Suggestions...)
	s.fields = append(s.fields, file.Fields...)
	return len(file.Lines), nil
}

// translation is the script's translation of a line, if it has one.
func (s *Script) translation(req llm.TranslateRequest) (string, bool) {
	if s == nil {
		return "", false
	}
	out, ok := s.translations[[3]string{req.From, req.Text, req.To}]
	return out, ok
}

// suggest is the script's suggestions whose line is in the transcript, each citing its latest utterance.
func (s *Script) suggest(transcript []llm.Utterance) []llm.Suggestion {
	if s == nil {
		return nil
	}
	var out []llm.Suggestion
	for _, sg := range s.suggestions {
		if id, ok := latest(transcript, sg.After); ok {
			out = append(out, llm.Suggestion{Kind: sg.Type, Text: sg.Text, SourceIDs: []string{id}, Confidence: sg.Confidence})
		}
	}
	return out
}

// field is the script's value of the field key, from its line in the transcript, if it has one.
func (s *Script) field(key string, transcript []llm.Utterance) (llm.DraftField, bool) {
	if s == nil {
		return llm.DraftField{}, false
	}
	for _, f := range s.fields {
		if f.Key != key || f.From == "" {
			continue
		}
		if id, ok := latest(transcript, f.From); ok {
			return llm.DraftField{Value: f.Value, SourceIDs: []string{id}}, true
		}
	}
	return llm.DraftField{}, false
}

// latest is the id of the last utterance whose text is text.
func latest(transcript []llm.Utterance, text string) (string, bool) {
	for i := len(transcript) - 1; i >= 0; i-- {
		if transcript[i].Text == text {
			return transcript[i].ID, true
		}
	}
	return "", false
}
