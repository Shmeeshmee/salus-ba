package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunWritesTheSeedAndItsCounts(t *testing.T) {
	srv := httptest.NewServer(&recordedWikidata{})
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "seed.jsonl")
	var log bytes.Buffer

	if err := run(context.Background(), srv.Client(), srv.URL, out, &log); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(b, []byte(`{"labels":`)) {
		t.Fatalf("seed = %.80q, %v", b, err)
	}
	lines := strings.Count(string(b), "\n")
	for _, c := range []string{"anatomy", "diagnosis", "symptom", "procedure", "drug", "total"} {
		if !strings.Contains(log.String(), c+" ") {
			t.Errorf("log has no count for %s:\n%s", c, log.String())
		}
	}
	if !strings.Contains(log.String(), "total "+strconv.Itoa(lines)) {
		t.Errorf("log = %q, want total %d", log.String(), lines)
	}
}

func TestRunKeepsTheOldSeedWhenTheFetchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "seed.jsonl")
	if err := os.WriteFile(out, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run(context.Background(), srv.Client(), srv.URL, out, &bytes.Buffer{}); err == nil {
		t.Fatal("run: err = nil, want the fetch's error")
	}
	if b, _ := os.ReadFile(out); string(b) != "old\n" {
		t.Errorf("seed = %q, want the old one kept", b)
	}
}
