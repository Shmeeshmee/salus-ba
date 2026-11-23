package signup_test

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/signup"
)

// documented returns the description of the response status of the operation method path in api/openapi.yaml, its
// $ref to components/responses resolved; "" when the operation does not document that status.
func documented(t *testing.T, method, path string, status int) string {
	t.Helper()
	b, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	type response struct {
		Ref         string `yaml:"$ref"`
		Description string `yaml:"description"`
	}
	var doc struct {
		Paths      map[string]map[string]yaml.Node         `yaml:"paths"`
		Components struct{ Responses map[string]response } `yaml:"components"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	var op struct{ Responses map[string]response }
	node := doc.Paths[path][strings.ToLower(method)]
	if err := node.Decode(&op); err != nil {
		t.Fatal(err)
	}
	r, ok := op.Responses[strconv.Itoa(status)]
	if !ok {
		return ""
	}
	if name, ok := strings.CutPrefix(r.Ref, "#/components/responses/"); ok {
		r = doc.Components.Responses[name]
	}
	return r.Description
}

// Every error a route's service method returns answers a status and an error code that api/openapi.yaml documents for
// that route; only an unexpected error is a 500.
func TestHandler_EveryDomainErrorIsADocumentedClientError(t *testing.T) {
	const (
		signupBody = `{"email":"admin@example.test","clinicName":"Synthetic clinic","language":"en"}`
		verifyBody = `{"token":"t"}`
		setupBody  = `{"details":"done","billing":"open","hours":"open","practitioner":"skipped","services":"open"}`
	)
	limited := &signup.RateLimitedError{RetryAfter: time.Minute}
	for _, tc := range []struct {
		err          error
		method, path string
		body, code   string
		status       int
	}{
		{method: http.MethodPost, path: "/signup", body: signupBody, err: signup.ErrInvalid, status: http.StatusBadRequest, code: "invalid_signup"},
		{method: http.MethodPost, path: "/signup", body: signupBody, err: limited, status: http.StatusTooManyRequests, code: "rate_limited"},
		{method: http.MethodPost, path: "/signup/verify", body: verifyBody, err: signup.ErrInvalidToken, status: http.StatusBadRequest, code: "invalid_verification"},
		{method: http.MethodPost, path: "/signup/verify", body: verifyBody, err: limited, status: http.StatusTooManyRequests, code: "rate_limited"},
		{method: http.MethodGet, path: "/clinic/setup", err: signup.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
		{method: http.MethodGet, path: "/clinic/setup", err: signup.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
		{method: http.MethodPut, path: "/clinic/setup", body: setupBody, err: signup.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
		{method: http.MethodPut, path: "/clinic/setup", body: setupBody, err: signup.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
		{method: http.MethodPut, path: "/clinic/setup", body: setupBody, err: signup.ErrInvalidSetup, status: http.StatusBadRequest, code: "invalid_request"},
	} {
		name := tc.method + " " + tc.path + " " + tc.err.Error()
		rec := serve(t, &stubService{err: tc.err}, true, tc.method, tc.path, tc.body)
		if rec.Code != tc.status || !strings.Contains(rec.Body.String(), `"error":"`+tc.code+`"`) {
			t.Errorf("%s = %d %s, want %d %s", name, rec.Code, rec.Body, tc.status, tc.code)
		}
		if d := documented(t, tc.method, tc.path, tc.status); !strings.Contains(d, "`"+tc.code+"`") {
			t.Errorf("%s: api/openapi.yaml documents %d as %q, want the code `%s`", name, tc.status, d, tc.code)
		}
	}
	if rec := serve(t, &stubService{err: errors.New("connection reset")}, true, http.MethodGet, "/clinic/setup", ""); rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "connection reset") {
		t.Errorf("GET /clinic/setup with an unexpected error = %d %s, want a 500 that says nothing of it", rec.Code, rec.Body)
	}
}
