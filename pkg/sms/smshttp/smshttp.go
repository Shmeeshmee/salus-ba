// Package smshttp is a generic HTTPS adapter of sms.Sender: one JSON request per message to the provider's endpoint,
// with a bearer token. It is merged off and wired nowhere (docs/plans/appointments.md, Q2): the provider is not chosen,
// and when it is, this adapter is checked against (and adapted to) its API, its DPA and sub-processors are recorded in
// docs/privacy/processors.md, and the owner adds the credentials (internal/config LoadSMS). Nothing here logs; errors
// never carry the number, the body, the token or the endpoint's answer.
package smshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Shmeeshmee/salus/pkg/sms"
)

// maxAnswer bounds the provider's answer read.
const maxAnswer = 64 << 10

// Sender posts each message to Endpoint as {"to", "text", "reference"} with "Authorization: Bearer <token>" and reads
// {"id"} from a 2xx answer; a 4xx answer is sms.ErrRejected (a retry fails the same way), anything else an error to
// retry.
type Sender struct {
	client   *http.Client
	endpoint string
	token    string
}

// New returns the Sender on client (with its timeout) for the https endpoint and the token.
func New(client *http.Client, endpoint, token string) (*Sender, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, errors.New("smshttp: the endpoint is not an https URL")
	}
	if client == nil || token == "" {
		return nil, errors.New("smshttp: no client or token")
	}
	return &Sender{client: client, endpoint: endpoint, token: token}, nil
}

type request struct {
	To        string `json:"to"`
	Text      string `json:"text"`
	Reference string `json:"reference,omitempty"`
}

// Send delivers m and returns the provider's id.
func (s *Sender) Send(ctx context.Context, m sms.Message) (string, error) {
	if err := sms.Check(m); err != nil {
		return "", err
	}
	body, err := json.Marshal(request{To: m.To, Text: m.Body, Reference: m.Reference})
	if err != nil {
		return "", errors.New("smshttp: encode the request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("smshttp: build the request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)
	res, err := s.client.Do(req)
	if err != nil {
		return "", errors.New("smshttp: the provider could not be reached")
	}
	defer func() { _ = res.Body.Close() }()
	answer, err := io.ReadAll(io.LimitReader(res.Body, maxAnswer))
	switch {
	case err != nil:
		return "", errors.New("smshttp: read the answer")
	case res.StatusCode >= 400 && res.StatusCode < 500:
		return "", fmt.Errorf("%w: status %d", sms.ErrRejected, res.StatusCode)
	case res.StatusCode < 200 || res.StatusCode > 299:
		return "", fmt.Errorf("smshttp: status %d", res.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(answer, &out) != nil || out.ID == "" {
		return "", errors.New("smshttp: the answer has no id")
	}
	return out.ID, nil
}
