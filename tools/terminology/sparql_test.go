package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestSparql points a client at srv and records its waits instead of sleeping.
func newTestSparql(srv *httptest.Server, waits *[]time.Duration) *sparql {
	return &sparql{
		http:     srv.Client(),
		endpoint: srv.URL,
		attempts: 4,
		backoff:  2 * time.Second,
		sleep: func(_ context.Context, d time.Duration) error {
			*waits = append(*waits, d)
			return nil
		},
	}
}

func TestQuerySendsAPoliteRequest(t *testing.T) {
	var got *http.Request
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_ = r.ParseForm()
		query = r.PostForm.Get("query")
		_, _ = w.Write([]byte(`{"results":{"bindings":[{"item":{"type":"uri","value":"http://www.wikidata.org/entity/Q9368"},` +
			`"value":{"type":"literal","value":"jetra","xml:lang":"bs"}}]}}`))
	}))
	defer srv.Close()
	var waits []time.Duration

	rows, err := newTestSparql(srv, &waits).query(context.Background(), "SELECT ?item WHERE {}")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if query != "SELECT ?item WHERE {}" {
		t.Errorf("query sent = %q", query)
	}
	ua := got.Header.Get("User-Agent")
	if !strings.Contains(ua, "Salus") || strings.Contains(ua, "@") {
		t.Errorf("User-Agent = %q, want one naming Salus without an e-mail address", ua)
	}
	if a := got.Header.Get("Accept"); a != "application/sparql-results+json" {
		t.Errorf("Accept = %q", a)
	}
	if len(rows) != 1 || rows[0]["item"].Value != "http://www.wikidata.org/entity/Q9368" ||
		rows[0]["value"] != (value{Value: "jetra", Lang: "bs"}) {
		t.Errorf("rows = %+v", rows)
	}
}

func TestQueryBacksOffOn429(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		switch calls {
		case 1:
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		case 3:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_, _ = w.Write([]byte(`{"results":{"bindings":[]}}`))
		}
	}))
	defer srv.Close()
	var waits []time.Duration

	if _, err := newTestSparql(srv, &waits).query(context.Background(), "q"); err != nil {
		t.Fatalf("query: %v", err)
	}
	want := []time.Duration{7 * time.Second, 4 * time.Second, 8 * time.Second}
	if calls != 4 || len(waits) != len(want) {
		t.Fatalf("calls = %d, waits = %v; want 4 calls and waits %v", calls, waits, want)
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Errorf("waits = %v, want %v (Retry-After, then the doubling backoff)", waits, want)
		}
	}
}

func TestQueryGivesUp(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantCalls int
	}{
		{"after the last attempt", http.StatusTooManyRequests, 4},
		{"at once on a bad query", http.StatusBadRequest, 1},
		{"at once on a server error", http.StatusInternalServerError, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			var waits []time.Duration

			_, err := newTestSparql(srv, &waits).query(context.Background(), "q")
			var se *statusError
			if !errors.As(err, &se) || se.status != tt.status {
				t.Errorf("err = %v, want HTTP %d", err, tt.status)
			}
			if calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestQueryStopsWhenTheWaitIsCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	s := newTestSparql(srv, new([]time.Duration))
	s.sleep = sleepCtx
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := s.query(ctx, "q"); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestQueryRefusesABrokenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`upstream request timeout`))
	}))
	defer srv.Close()

	if _, err := newTestSparql(srv, new([]time.Duration)).query(context.Background(), "q"); err == nil {
		t.Error("err = nil, want a decode error")
	}
}
