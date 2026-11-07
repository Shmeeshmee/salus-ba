package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// userAgent names the tool and the project, as Wikimedia's User-Agent policy asks; never an e-mail address.
const userAgent = "Salus-terminology-seed/1.0 (https://github.com/Shmeeshmee/salus; medical term list for a telemedicine app) Go-http-client"

// sparql asks Wikidata's query service one query at a time and backs off when it is told to.
type sparql struct {
	http     *http.Client
	sleep    func(context.Context, time.Duration) error
	endpoint string
	attempts int           // tries per query
	backoff  time.Duration // the first wait without a Retry-After; doubled per retry
}

// value is one cell of a SPARQL JSON result: a literal with its language tag, or an IRI.
type value struct {
	Value string `json:"value"`
	Lang  string `json:"xml:lang"`
}

type row map[string]value

// statusError is a response the service refused; retryable when a later try may succeed.
type statusError struct {
	retryAfter time.Duration
	status     int
}

func (e *statusError) Error() string { return fmt.Sprintf("wikidata: HTTP %d", e.status) }

func (e *statusError) retryable() bool {
	switch e.status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// query runs one SPARQL query, retrying on 429 and on a busy service, waiting Retry-After or the doubling backoff.
func (s *sparql) query(ctx context.Context, q string) ([]row, error) {
	wait := s.backoff
	for attempt := 1; ; attempt++ {
		rows, err := s.once(ctx, q)
		se, ok := err.(*statusError)
		if err == nil || !ok || !se.retryable() || attempt >= s.attempts {
			return rows, err
		}
		d := wait
		if se.retryAfter > 0 {
			d = se.retryAfter
		}
		if err := s.sleep(ctx, d); err != nil {
			return nil, err
		}
		wait *= 2
	}
}

func (s *sparql) once(ctx context.Context, q string) ([]row, error) {
	form := url.Values{"query": {q}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return nil, &statusError{status: resp.StatusCode, retryAfter: time.Duration(secs) * time.Second}
	}
	var body struct {
		Results struct {
			Bindings []row `json:"bindings"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("wikidata: decode the result: %w", err)
	}
	return body.Results.Bindings, nil
}

// sleepCtx waits d or until ctx ends.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
