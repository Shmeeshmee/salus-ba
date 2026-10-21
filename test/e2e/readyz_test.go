//go:build e2e

// Package e2e tests the running stack end to end. Run with `make e2e`, which starts the stack and points
// E2E_BASE_URL at the API on the compose network.
package e2e_test

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestStack_ReadyzReturns200(t *testing.T) {
	base := os.Getenv("E2E_BASE_URL")
	if base == "" {
		t.Fatal("E2E_BASE_URL is not set; run the end-to-end tests with `make e2e`")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	timeout := time.After(30 * time.Second)
	for {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+"/readyz", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-timeout:
			t.Fatalf("GET %s/readyz did not return 200 within 30s (last: %v, %v)", base, resp, err)
		case <-time.After(250 * time.Millisecond):
		}
	}
}
