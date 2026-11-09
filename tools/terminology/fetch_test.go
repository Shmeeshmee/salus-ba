package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/seed.golden.jsonl")

// testCategories are the real categories with small limits, matching the responses recorded in testdata/.
func testCategories() []category {
	limits := map[string]int{"symptom": 3, "anatomy": 3, "procedure": 2, "drug": 2, "diagnosis": 3}
	cs := categories()
	for i := range cs {
		cs[i].limit = limits[cs[i].name]
	}
	return cs
}

// recordedWikidata answers like the query service from the responses in testdata/: rank-<category>.json for a
// category's ranking, details.json (filtered to the items asked for) for labels and codes.
type recordedWikidata struct {
	limits  []string
	batches [][]string
	mu      sync.Mutex
	active  int
	busiest int
}

var (
	tagRe   = regexp.MustCompile(`^#salus-terminology (rank (\w+)|details)\n`)
	limitRe = regexp.MustCompile(`LIMIT (\d+)\s*$`)
	itemRe  = regexp.MustCompile(`wd:(Q\d+)`)
)

func (rw *recordedWikidata) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw.mu.Lock()
	rw.active++
	rw.busiest = max(rw.busiest, rw.active)
	rw.mu.Unlock()
	defer func() { rw.mu.Lock(); rw.active--; rw.mu.Unlock() }()

	q := r.PostFormValue("query")
	tag := tagRe.FindStringSubmatch(q)
	if tag == nil {
		http.Error(w, "untagged query", http.StatusBadRequest)
		return
	}
	if tag[1] != "details" {
		rw.limits = append(rw.limits, tag[2]+"="+limitRe.FindStringSubmatch(q)[1])
		http.ServeFile(w, r, "testdata/rank-"+tag[2]+".json")
		return
	}
	var asked []string
	for _, m := range itemRe.FindAllStringSubmatch(q, -1) {
		asked = append(asked, m[1])
	}
	rw.batches = append(rw.batches, asked)
	var all struct {
		Results struct {
			Bindings []row `json:"bindings"`
		} `json:"results"`
	}
	b, _ := os.ReadFile("testdata/details.json")
	_ = json.Unmarshal(b, &all)
	var out []row
	for _, x := range all.Results.Bindings {
		if slices.Contains(asked, strings.TrimPrefix(x["item"].Value, entityPrefix)) {
			out = append(out, x)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{"bindings": out}})
}

func TestFetchAgainstRecordedResponses(t *testing.T) {
	rec := &recordedWikidata{}
	srv := httptest.NewServer(rec)
	defer srv.Close()
	s := &sparql{http: srv.Client(), endpoint: srv.URL, attempts: 1, sleep: sleepCtx}

	terms, err := fetch(context.Background(), s, testCategories(), 4)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	wantLimits := []string{"symptom=3", "anatomy=6", "procedure=8", "drug=10", "diagnosis=13"}
	if !slices.Equal(rec.limits, wantLimits) {
		t.Errorf("rank queries = %v, want %v (each category's limit plus the items already taken)", rec.limits, wantLimits)
	}
	if rec.busiest != 1 {
		t.Errorf("at most %d requests at a time, want 1", rec.busiest)
	}
	for _, b := range rec.batches {
		if len(b) > 4 {
			t.Errorf("a details batch asked for %d items, want at most 4", len(b))
		}
	}
	perCategory := map[string]int{}
	seen := map[string]string{}
	for _, tm := range terms {
		perCategory[tm.Category]++
		if c, dup := seen[tm.Wikidata]; dup {
			t.Errorf("%s is in %s and %s", tm.Wikidata, c, tm.Category)
		}
		seen[tm.Wikidata] = tm.Category
		if len(tm.Labels) == 0 {
			t.Errorf("%s has no label", tm.Wikidata)
		}
	}
	for c, n := range map[string]int{"symptom": 3, "anatomy": 3, "procedure": 2, "drug": 2, "diagnosis": 3} {
		if perCategory[c] != n {
			t.Errorf("%s: %d terms, want %d", c, perCategory[c], n)
		}
	}

	var buf bytes.Buffer
	if err := writeSeed(&buf, terms); err != nil {
		t.Fatalf("writeSeed: %v", err)
	}
	golden(t, "testdata/seed.golden.jsonl", buf.Bytes())
}

func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to write it)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("seed differs from %s:\n%s", path, got)
	}
}

func TestFetchStopsOnAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := &sparql{http: srv.Client(), endpoint: srv.URL, attempts: 1, sleep: sleepCtx}

	if _, err := fetch(context.Background(), s, testCategories(), 4); err == nil {
		t.Error("err = nil, want the query's error")
	}
}

func TestDetailsRules(t *testing.T) {
	lit := func(item, field, v, lang string) row {
		return row{"item": {Value: entityPrefix + item}, "field": {Value: field}, "value": {Value: v, Lang: lang}}
	}
	rows := []row{
		lit("Q1", "icd10", "R51", ""),
		lit("Q1", "icd10", "G44.2", ""),
		lit("Q1", "icd10", "R51", ""),
		lit("Q1", "icd9cm", "784.0", ""), // a diagnosis code: not a procedure
		lit("Q1", "icd9cm", "47.0", ""),
		lit("Q1", "pcs", "0DTJ4ZZ", ""),
		lit("Q1", "label", "headache", "en"),
		lit("Q1", "alias", "cephalalgia", "en"),
		lit("Q1", "label", "главобоља", "sr"),
		lit("Q2", "atc", "N02BE01", ""),
	}
	got := details(rows)
	q1 := got["Q1"]
	if q1.icd10 != "G44.2, R51" || q1.pcs != "0DTJ4ZZ, 47.0" {
		t.Errorf("Q1 codes: icd10 %q, pcs %q; want %q, %q", q1.icd10, q1.pcs, "G44.2, R51", "0DTJ4ZZ, 47.0")
	}
	if got := fmt.Sprint(buildLabels(q1.names)); got != "map[en:[headache cephalalgia] sr-Cyrl:[главобоља] sr-Latn:[glavobolja]]" {
		t.Errorf("Q1 labels = %s", got)
	}
	if got["Q2"].atc != "N02BE01" || len(got["Q2"].names) != 0 {
		t.Errorf("Q2 = %+v", got["Q2"])
	}
}

func TestRankQueryNamesItsCategory(t *testing.T) {
	for _, c := range categories() {
		q := rankQuery(c, 42)
		if !strings.HasPrefix(q, "#salus-terminology rank "+c.name+"\n") || !strings.HasSuffix(strings.TrimSpace(q), "LIMIT 42") {
			t.Errorf("rankQuery(%s) =\n%s", c.name, q)
		}
	}
	if got := len(categories()); got != 5 {
		t.Errorf("%d categories, want 5", got)
	}
}
