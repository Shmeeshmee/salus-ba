package notification

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// trusting is a poster that trusts srv's certificate and connects to any address (the test's server is loopback).
func trusting(srv *httptest.Server) *httpPoster {
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return newPoster(5*time.Second, &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, func(netip.Addr) bool { return true })
}

// A post sends the body and headers and returns the status; a redirect is answered as it is, never followed.
func TestPoster_PostsTheBodyAndDoesNotFollowARedirect(t *testing.T) {
	var got struct {
		body, signature, agent, contentType string
		hits                                int
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.hits++
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
			return
		}
		b, _ := io.ReadAll(r.Body)
		got.body, got.signature, got.agent, got.contentType = string(b), r.Header.Get("Salus-Signature"), r.Header.Get("User-Agent"), r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	p := trusting(srv)
	status, err := p.Post(t.Context(), srv.URL+"/hook", []byte(`{"id":"x"}`), map[string]string{"Salus-Signature": "t=1,v1=ab", "Content-Type": "application/json"})
	if err != nil || status != http.StatusAccepted || got.body != `{"id":"x"}` || got.signature != "t=1,v1=ab" || got.agent != "Salus-Webhooks/1" || got.contentType != "application/json" {
		t.Errorf("Post() = %d, %v; the server got %+v", status, err, got)
	}
	got.hits = 0
	if status, err := p.Post(t.Context(), srv.URL+"/moved", nil, nil); err != nil || status != http.StatusTemporaryRedirect || got.hits != 1 {
		t.Errorf("Post(redirect) = %d, %v after %d requests; want the 307 itself", status, err, got.hits)
	}
}

// A server that does not answer in time, or cannot be reached, is an error that never holds the URL.
func TestPoster_TimesOutAndNeverNamesTheURL(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	p := newPoster(200*time.Millisecond, &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, func(netip.Addr) bool { return true })
	_, err := p.Post(context.Background(), srv.URL+"/secret-path?token=synthetic", nil, nil)
	if err == nil || strings.Contains(err.Error(), "secret-path") || strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("Post(slow) = %v, want an error without the URL", err)
	}
	if _, err := p.Post(context.Background(), "https://127.0.0.1:1/secret-path", nil, nil); err == nil || strings.Contains(err.Error(), "secret-path") {
		t.Errorf("Post(closed port) = %v", err)
	}
}
