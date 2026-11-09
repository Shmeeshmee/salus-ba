// Package anthropic is the pkg/llm ports on Anthropic's Claude API: one Messages API request per call
// (POST https://api.anthropic.com/v1/messages), the answer constrained to a JSON schema (structured outputs) and
// validated before it is returned. Plain net/http with an injected client, no SDK: one endpoint and three JSON
// shapes, fewer lines than the SDK's configuration.
//
// It is a processor of patient text (docs/privacy/processors.md): merged off by default, not enabled with real
// patient data before its DPA, zero data retention (an arrangement Anthropic enables for the organisation, not a
// request flag) and the plan's EU region condition are met. It sends the model only what the call needs (the
// line or the transcript, the glossary, the field labels; no name, id of a person, or account), asks for no beta
// and no retention-based feature, and logs nothing: its errors name a status, an API error type or a stop
// reason, never a key, text sent or text received. The JSON schemas are fixed and hold no patient data (they are
// cached by the API apart from the prompts).
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// endpoint is the Messages API. There is no override: tests reach their server through the client's transport.
const endpoint = "https://api.anthropic.com/v1/messages"

// apiVersion is the anthropic-version header: the current API version, under which a request may not set
// temperature, top_p or top_k (it sets none).
const apiVersion = "2026-10-21"

// maxResponse bounds the answer read from the API.
const maxResponse = 1 << 20

var (
	// ErrConfig is returned by New for a missing key, model or HTTP client.
	ErrConfig = errors.New("anthropic: invalid configuration")
	// ErrService is returned when the API answers with an error status or a body that is not a message, or
	// cannot be reached.
	ErrService = errors.New("anthropic: the Claude API failed")
	// ErrRefused is returned when the model declined to answer (stop reason refusal).
	ErrRefused = errors.New("anthropic: the model refused")
	// ErrIncomplete is returned when the answer was cut off (stop reason max_tokens).
	ErrIncomplete = errors.New("anthropic: the answer was cut off")
	// ErrInvalidOutput is returned for an answer that is not its schema's JSON, or holds nothing usable.
	ErrInvalidOutput = errors.New("anthropic: invalid answer")
)

// Config is the API's settings: the key of the organisation's workspace and the model (a Claude model id).
type Config struct {
	Key   string
	Model string
}

// Client calls the Claude API. It is safe for concurrent use.
type Client struct {
	http  *http.Client
	key   string
	model string
}

// New returns a Client for cfg on hc, whose timeouts and transport the caller sets; each call is bounded by its
// context. It refuses a missing key, model or client; the error names the setting, never its value.
func New(cfg Config, hc *http.Client) (*Client, error) {
	switch {
	case cfg.Key == "":
		return nil, fmt.Errorf("%w: no key", ErrConfig)
	case cfg.Model == "":
		return nil, fmt.Errorf("%w: no model", ErrConfig)
	case hc == nil:
		return nil, fmt.Errorf("%w: no HTTP client", ErrConfig)
	}
	return &Client{http: hc, key: cfg.Key, model: cfg.Model}, nil
}

// task is one kind of call: how much the model may think and write, and the schema its answer follows.
type task struct {
	effort    string
	schema    json.RawMessage
	maxTokens int
}

type messageRequest struct {
	Model        string       `json:"model"`
	System       string       `json:"system"`
	Messages     []message    `json:"messages"`
	OutputConfig outputConfig `json:"output_config"`
	MaxTokens    int          `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type outputConfig struct {
	Effort string `json:"effort"`
	Format format `json:"format"`
}

type format struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

type messageResponse struct {
	Type       string `json:"type"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type errorResponse struct {
	Error struct {
		Type string `json:"type"`
	} `json:"error"`
}

// call sends one request (system prompt, one user message) for t and decodes the answer's text into out.
func (c *Client) call(ctx context.Context, t task, system, user string, out any) error {
	body, err := encode(messageRequest{
		Model: c.model, MaxTokens: t.maxTokens, System: system,
		Messages:     []message{{Role: "user", Content: user}},
		OutputConfig: outputConfig{Effort: t.effort, Format: format{Type: "json_schema", Schema: t.schema}},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: request: %w", ErrService, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.key)
	req.Header.Set("Anthropic-Version", apiVersion)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: unreachable", ErrService) // the transport's error may carry the URL; nothing else
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return fmt.Errorf("%w: reading the answer", ErrService)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d%s", ErrService, resp.StatusCode, errorType(raw))
	}
	return decodeAnswer(raw, out)
}

// decodeAnswer reads the message's text into out, by its stop reason.
func decodeAnswer(raw []byte, out any) error {
	var m messageResponse
	if err := json.Unmarshal(raw, &m); err != nil || m.Type != "message" {
		return fmt.Errorf("%w: the answer is not a message", ErrService)
	}
	switch m.StopReason {
	case "refusal":
		return ErrRefused
	case "max_tokens":
		return ErrIncomplete
	}
	var text strings.Builder
	for _, b := range m.Content {
		if b.Type == "text" {
			text.WriteString(b.Text)
		}
	}
	if text.Len() == 0 {
		return fmt.Errorf("%w: no text (stop reason %q)", ErrInvalidOutput, safeWord(m.StopReason))
	}
	if err := json.Unmarshal([]byte(text.String()), out); err != nil {
		return fmt.Errorf("%w: not its schema's JSON", ErrInvalidOutput)
	}
	return nil
}

// errorType is ", <the API error's type>" when the body is an API error with a plain type, else empty: the
// error's message is never kept.
func errorType(raw []byte) string {
	var e errorResponse
	if json.Unmarshal(raw, &e) != nil || safeWord(e.Error.Type) == "" {
		return ""
	}
	return ", " + e.Error.Type
}

// safeWord is s when it is a short lower-case identifier (an API enum value), else empty.
func safeWord(s string) string {
	if len(s) > 40 || strings.Trim(s, "abcdefghijklmnopqrstuvwxyz_") != "" {
		return ""
	}
	return s
}

// encode is v as JSON without HTML escaping, so the prompts' tags and the languages' letters go as written.
func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("anthropic: encoding the request: %w", err)
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}
