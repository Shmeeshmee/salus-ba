// Package llm holds the ports of the language-model tasks: translating a segment, suggesting from a
// transcript, filling a document's fields from it, and summarising a conversation. It knows nothing of consultations; a caller maps its
// own types in and out. llmtest has in-memory fakes. An implementation that sends text to a hosted model
// needs its DPA and an entry in docs/privacy/processors.md first (CLAUDE.md "GDPR and health data").
package llm

import "context"

// TranslateRequest is a text and its language, and the language to translate it into (BCP 47 tags). Glossary
// is the terms found in the text, each with the term to use for it in the target language; it may be empty.
type TranslateRequest struct {
	Text     string
	From     string
	To       string
	Glossary []Term
}

// Term is a glossary entry: a term as it is written in the source language, and in the target language.
type Term struct {
	Source string
	Target string
}

// Translator translates one text.
type Translator interface {
	Translate(ctx context.Context, req TranslateRequest) (string, error)
}

// Utterance is one line of a transcript, with the id a result cites it by.
type Utterance struct {
	ID      string
	Speaker string
	Text    string
}

// SuggestRequest is a transcript, the language to write the suggestions in, and the texts of the suggestions
// already shown from it (dismissed and pinned ones included): none of them, nor one saying the same, is to be
// suggested again.
type SuggestRequest struct {
	Lang       string
	Transcript []Utterance
	Shown      []string
}

// Suggestion is one suggestion: its kind (a name the caller defines), its text, how confident the model is
// (0 to 1), and the utterances it came from.
type Suggestion struct {
	Kind       string
	Text       string
	SourceIDs  []string
	Confidence float32
}

// Suggester suggests from a transcript.
type Suggester interface {
	Suggest(ctx context.Context, req SuggestRequest) ([]Suggestion, error)
}

// FieldType is what kind of value a field holds, and so how the model writes it.
type FieldType string

// The field types. A list's items are one per line; a date is YYYY-MM-DD; a coded field's value is one code of
// the code system the instruction names (ICD-10 for a diagnosis).
const (
	FieldText  FieldType = "text"
	FieldList  FieldType = "list"
	FieldDate  FieldType = "date"
	FieldCoded FieldType = "coded"
)

// FieldSpec is one field of a document to fill: its key, the label and the instruction the model reads, its
// type, and whether the document needs it. A required field the transcript does not cover is still Missing:
// the model never guesses.
type FieldSpec struct {
	Key         string
	Label       string
	Instruction string
	Type        FieldType
	Required    bool
}

// DraftRequest is the fields to fill from a transcript, and the language to write them in.
type DraftRequest struct {
	Lang       string
	Fields     []FieldSpec
	Transcript []Utterance
}

// DraftField is one filled field, with the utterances its value came from, or Missing when the transcript
// does not say.
type DraftField struct {
	Key       string
	Value     string
	SourceIDs []string
	Missing   bool
}

// Drafter fills a document's fields from a transcript, one result per requested field, in order.
type Drafter interface {
	Draft(ctx context.Context, req DraftRequest) ([]DraftField, error)
}

// SummarizeRequest is a conversation's messages, oldest first, each with its sender's role as Speaker, and the
// language to write the summary in (a BCP 47 tag).
type SummarizeRequest struct {
	Lang     string
	Messages []Utterance
}

// Summarizer summarises a conversation for the practitioner who reads it. The summary stands beside the messages,
// never in place of them, and states what was written, not a diagnosis.
type Summarizer interface {
	Summarize(ctx context.Context, req SummarizeRequest) (string, error)
}
