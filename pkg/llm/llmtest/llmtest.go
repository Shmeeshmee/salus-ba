// Package llmtest holds in-memory fakes of the llm ports for tests. Each records its calls, returns a copy of
// its script, and fails with Err when set. All are safe for concurrent use.
package llmtest

import (
	"context"
	"slices"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/llm"
)

// Translator returns the text marked with the target language: "[de] text".
type Translator struct {
	Err   error
	calls []llm.TranslateRequest
	mu    sync.Mutex
}

// Translate records the request and marks its text.
func (f *Translator) Translate(_ context.Context, req llm.TranslateRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	if f.Err != nil {
		return "", f.Err
	}
	return "[" + req.To + "] " + req.Text, nil
}

// Calls returns the requests so far, oldest first.
func (f *Translator) Calls() []llm.TranslateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// Suggester returns Suggestions on every call.
type Suggester struct {
	Err         error
	Suggestions []llm.Suggestion
	calls       []llm.SuggestRequest
	mu          sync.Mutex
}

// Suggest records the request and returns a copy of Suggestions.
func (f *Suggester) Suggest(_ context.Context, req llm.SuggestRequest) ([]llm.Suggestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]llm.Suggestion, len(f.Suggestions))
	for i, s := range f.Suggestions {
		s.SourceIDs = slices.Clone(s.SourceIDs)
		out[i] = s
	}
	return out, nil
}

// Calls returns the requests so far, oldest first.
func (f *Suggester) Calls() []llm.SuggestRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// Drafter fills each requested field from Fields by its key, and marks a field it has no entry for missing.
type Drafter struct {
	Err    error
	Fields map[string]llm.DraftField
	calls  []llm.DraftRequest
	mu     sync.Mutex
}

// Draft records the request and returns one field per requested field, in order.
func (f *Drafter) Draft(_ context.Context, req llm.DraftRequest) ([]llm.DraftField, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	req.Fields, req.Transcript = slices.Clone(req.Fields), slices.Clone(req.Transcript)
	f.calls = append(f.calls, req)
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]llm.DraftField, len(req.Fields))
	for i, spec := range req.Fields {
		field, ok := f.Fields[spec.Key]
		if !ok {
			field = llm.DraftField{Missing: true}
		}
		field.Key = spec.Key
		field.SourceIDs = slices.Clone(field.SourceIDs)
		out[i] = field
	}
	return out, nil
}

// Calls returns the requests so far, oldest first.
func (f *Drafter) Calls() []llm.DraftRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}
