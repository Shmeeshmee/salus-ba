package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// demoUser is one signed-in user of the demo seed: the api's own JSON and FHIR endpoints, called as the web app
// (staff) or the patient app (patients) call them, with the user's session token as a bearer token (the patient app's
// way; the web app's cookie and its X-Salus-CSRF header guard a browser, which the seed is not). Every guard, trigger, audit and
// fiscal rule of the api holds for what it writes.
type demoUser struct {
	http  *http.Client
	base  string
	token string
}

// demoResponse is a call's status, its ETag and its body.
type demoResponse struct {
	etag   string
	body   []byte
	status int
}

// call sends body (JSON, nil for none) to path and returns the response; ifMatch is sent as If-Match when set.
func (u *demoUser) call(ctx context.Context, method, path string, body any, ifMatch string) (demoResponse, error) {
	var in io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return demoResponse{}, err
		}
		in = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.base+path, in)
	if err != nil {
		return demoResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.HasPrefix(path, "/fhir/") {
		req.Header.Set("Content-Type", "application/fhir+json")
	}
	if u.token != "" {
		req.Header.Set("Authorization", "Bearer "+u.token)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	res, err := u.http.Do(req)
	if err != nil {
		return demoResponse{}, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return demoResponse{}, err
	}
	return demoResponse{status: res.StatusCode, etag: res.Header.Get("ETag"), body: raw}, nil
}

// must sends the call and decodes the response into out (nil to skip), failing on any status but want. The error
// names the call and the status, and the api's error code, never a body's personal data: the codes are fixed words.
func (u *demoUser) must(ctx context.Context, want int, method, path string, body, out any) (demoResponse, error) {
	return u.mustIf(ctx, want, method, path, "", body, out)
}

// mustIf is must with If-Match.
func (u *demoUser) mustIf(ctx context.Context, want int, method, path, ifMatch string, body, out any) (demoResponse, error) {
	res, err := u.call(ctx, method, path, body, ifMatch)
	if err != nil {
		return res, fmt.Errorf("demo-seed: %s %s: %w", method, route(path), err)
	}
	if res.status != want {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(res.body, &e)
		return res, fmt.Errorf("demo-seed: %s %s = %d %s, want %d", method, route(path), res.status, e.Error, want)
	}
	if out != nil {
		if err := json.Unmarshal(res.body, out); err != nil {
			return res, fmt.Errorf("demo-seed: %s %s: decode: %w", method, route(path), err)
		}
	}
	return res, nil
}

// route is path without its query, which may carry a name or a birth date.
func route(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i]
	}
	return path
}
