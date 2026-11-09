// Command terminology fetches the medical term list from Wikidata's SPARQL endpoint and writes the seed,
// assets/terminology/seed.jsonl (docs/terminology/SPEC.md §3 "Data"): per category the items that carry its
// identifier, the most written about first (sitelinks), with their labels and aliases in Latin and the ten app
// languages and their codes. Run once by hand with `make terminology-fetch` (network); never part of a build.
//
// Usage: terminology -out <file> [-endpoint url]
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "", "the seed file to write")
	endpoint := flag.String("endpoint", "https://query.wikidata.org/sparql", "Wikidata's SPARQL endpoint")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "terminology: -out is required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// The service stops a query after 60 seconds; the client waits a little longer for its answer.
	client := &http.Client{Timeout: 90 * time.Second}
	if err := run(ctx, client, *endpoint, *out, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "terminology:", err)
		os.Exit(1)
	}
}

// run fetches every category and replaces out only once the whole seed is written, then logs the counts.
func run(ctx context.Context, client *http.Client, endpoint, out string, log io.Writer) error {
	s := &sparql{http: client, endpoint: endpoint, attempts: 5, backoff: 5 * time.Second, sleep: sleepCtx}
	terms, err := fetch(ctx, s, categories(), 100)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(out), ".seed-*.jsonl")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := writeSeed(tmp, terms); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, t := range terms {
		counts[t.Category]++
	}
	for _, c := range []string{"anatomy", "diagnosis", "symptom", "procedure", "drug"} {
		_, _ = fmt.Fprintf(log, "%s %d\n", c, counts[c])
	}
	_, _ = fmt.Fprintf(log, "total %d\n", len(terms))
	return nil
}
