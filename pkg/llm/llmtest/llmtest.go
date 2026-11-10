// Package llmtest holds in-memory fakes of the llm ports for tests. Each records its calls, returns a copy of
// its script, and fails with Err when set. All are safe for concurrent use.
package llmtest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/pkg/llm"
)

// ErrScripted is the error of a Translator's scripted Failures.
var ErrScripted = errors.New("llmtest: scripted failure")

// Translator returns the text marked with the target language: "[de] text", or the Script's translation of a
// scripted line. It fails its first Failures calls
// with ErrScripted, and holds every call for Delay (or until its context ends) before it answers. With
// DiscardCalls it keeps none of its calls (for a long-running fake, as in cmd/agent).
type Translator struct {
	Err          error
	Script       *Script
	calls        []llm.TranslateRequest
	Failures     int
	Delay        time.Duration
	mu           sync.Mutex
	DiscardCalls bool
}

// Translate records the request and marks its text.
func (f *Translator) Translate(ctx context.Context, req llm.TranslateRequest) (string, error) {
	f.mu.Lock()
	if !f.DiscardCalls {
		f.calls = append(f.calls, llm.TranslateRequest{Text: req.Text, From: req.From, To: req.To, Glossary: slices.Clone(req.Glossary)})
	}
	fail := f.Failures > 0
	if fail {
		f.Failures--
	}
	err, delay := f.Err, f.Delay
	f.mu.Unlock()
	if delay > 0 {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	switch {
	case err != nil:
		return "", err
	case fail:
		return "", ErrScripted
	}
	if out, ok := f.Script.translation(req); ok {
		return out, nil
	}
	return "[" + req.To + "] " + req.Text, nil
}

// Calls returns the requests so far, oldest first.
func (f *Translator) Calls() []llm.TranslateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// Suggester returns Suggestions on every call, then the Script's suggestions whose line is in the transcript.
// With DiscardCalls it keeps none of its calls (for a long-running
// fake engine).
type Suggester struct {
	Err          error
	Script       *Script
	Suggestions  []llm.Suggestion
	calls        []llm.SuggestRequest
	mu           sync.Mutex
	DiscardCalls bool
}

// Suggest records the request and returns a copy of Suggestions.
func (f *Suggester) Suggest(_ context.Context, req llm.SuggestRequest) ([]llm.Suggestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.DiscardCalls {
		req.Transcript, req.Shown = slices.Clone(req.Transcript), slices.Clone(req.Shown)
		f.calls = append(f.calls, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]llm.Suggestion, len(f.Suggestions))
	for i, s := range f.Suggestions {
		s.SourceIDs = slices.Clone(s.SourceIDs)
		out[i] = s
	}
	return append(out, f.Script.suggest(req.Transcript)...), nil
}

// Calls returns the requests so far, oldest first.
func (f *Suggester) Calls() []llm.SuggestRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// Drafter fills each requested field from Fields by its key, else from the Script's line in the transcript, and
// marks a field it has neither for missing.
// With DiscardCalls it keeps none of its calls (for a long-running fake engine).
type Drafter struct {
	Err          error
	Script       *Script
	Fields       map[string]llm.DraftField
	calls        []llm.DraftRequest
	mu           sync.Mutex
	DiscardCalls bool
}

// Draft records the request and returns one field per requested field, in order.
func (f *Drafter) Draft(_ context.Context, req llm.DraftRequest) ([]llm.DraftField, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.DiscardCalls {
		req.Fields, req.Transcript = slices.Clone(req.Fields), slices.Clone(req.Transcript)
		f.calls = append(f.calls, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]llm.DraftField, len(req.Fields))
	for i, spec := range req.Fields {
		field, ok := f.Fields[spec.Key]
		if !ok {
			field, ok = f.Script.field(spec.Key, req.Transcript)
		}
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
