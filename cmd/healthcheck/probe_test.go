package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ok.Close()
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close() // nothing listens on its address any more

	tests := []struct {
		name   string
		stderr string
		args   []string
		want   int
	}{
		{"a 2xx is healthy", "", []string{ok.URL + "/healthz"}, 0},
		{"any other status is unhealthy", "status 503", []string{failing.URL}, 1},
		{"no listener is unhealthy", "healthcheck:", []string{gone.URL}, 1},
		{"no URL is a usage error", "usage: healthcheck <url>", nil, 1},
		{"two URLs are a usage error", "usage: healthcheck <url>", []string{ok.URL, ok.URL}, 1},
		{"a URL that is not http is refused", "healthcheck:", []string{"file:///etc/passwd"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got := run(tt.args, &http.Client{Timeout: time.Second}, &stderr)
			if got != tt.want {
				t.Errorf("run = %d, want %d (stderr %q)", got, tt.want, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.stderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.stderr)
			}
			if tt.want == 0 && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want nothing on success", stderr.String())
			}
		})
	}
}

// A slow endpoint counts as down once the client's timeout passes, so Docker's own timeout is never what ends
// the probe.
func TestRun_SlowEndpointIsUnhealthy(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	defer slow.Close()
	defer close(release)

	var stderr bytes.Buffer
	if got := run([]string{slow.URL}, &http.Client{Timeout: 50 * time.Millisecond}, &stderr); got != 1 {
		t.Errorf("run = %d, want 1", got)
	}
}
