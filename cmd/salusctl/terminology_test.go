package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A synthetic seed: two terms and a line that is not one.
const seed = `{"labels":{"la":["hepar","iecur"],"en":["liver"]},"wikidata":"Q9368","category":"anatomy","ta98":"A05.8.01.001","rank":180}
{"labels":{"en":["fever","pyrexia"]},"wikidata":"Q38933","category":"symptom","icd10":"R50.9","rank":120}
{"labels":
`

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTerminologyImport_LoadsTheSeedOnceAndReportsTheRefusedRows(t *testing.T) {
	_, count := ownerDB(t)
	path := writeFile(t, "seed.jsonl", seed)
	var out bytes.Buffer

	err := run(t.Context(), []string{"terminology", "import", path}, &out)

	if err == nil || !strings.Contains(err.Error(), "rows refused: 1") {
		t.Errorf("import = %v, want an error naming the refused row", err)
	}
	for _, want := range []string{"created 2, updated 0, unchanged 0, refused 1", "line 3: not a seed term", "terms: anatomy 1, symptom 1"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output %q: want %q", out.String(), want)
		}
	}
	if n := count(`SELECT count(*) FROM term WHERE review_status = 'unreviewed'`); n != 2 {
		t.Errorf("%d draft terms, want 2", n)
	}
	if n := count(`SELECT count(*) FROM term_label`); n != 5 {
		t.Errorf("%d labels, want 5", n)
	}

	out.Reset()
	path = writeFile(t, "seed.jsonl", strings.Join(strings.Split(seed, "\n")[:2], "\n"))
	if err := run(t.Context(), []string{"terminology", "import", path}, &out); err != nil {
		t.Fatalf("import again: %v", err)
	}
	if !strings.Contains(out.String(), "created 0, updated 0, unchanged 2, refused 0") || count(`SELECT count(*) FROM term`) != 2 {
		t.Errorf("import again: %q; want nothing created", out.String())
	}
}

// The curation round trip: export the drafts, mark one reviewed and fill a label in the CSV, import it.
func TestTerminologyExportImport_RoundTrip(t *testing.T) {
	_, count := ownerDB(t)
	if err := run(t.Context(), []string{"terminology", "import", writeFile(t, "seed.jsonl", strings.Join(strings.Split(seed, "\n")[:2], "\n"))}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	csvPath := filepath.Join(t.TempDir(), "terms.csv")
	var out bytes.Buffer

	if err := run(t.Context(), []string{"terminology", "export", "-o", csvPath, "-status", "unreviewed"}, &out); err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out.String(), "terms exported: 2, to "+csvPath) {
		t.Errorf("export output %q", out.String())
	}
	file, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(file), "\n")
	if len(lines) != 5 || !strings.Contains(lines[0], "# For the curator") || !strings.Contains(lines[1], "id,category,wikidata") ||
		!strings.Contains(lines[2], ",Q9368,") {
		t.Fatalf("exported %q: want the curator's note, the header and the two terms, the most common first", file)
	}
	// liver: reviewed, its Croatian label filled in (the hr column is the 11th).
	cells := strings.Split(lines[2], ",")
	cells[10], cells[len(cells)-2] = "jetra", "approved"
	lines[2] = strings.Join(cells, ",")
	edited := writeFile(t, "terms.csv", strings.Join(lines, "\n"))

	out.Reset()
	if err := run(t.Context(), []string{"terminology", "import", edited}, &out); err != nil {
		t.Fatalf("import the CSV: %v", err)
	}
	if !strings.Contains(out.String(), "created 0, updated 1, unchanged 1, refused 0") {
		t.Errorf("import the CSV: %q", out.String())
	}
	if n := count(`SELECT count(*) FROM term t JOIN term_label l ON l.term_id = t.id
		WHERE t.wikidata_id = 'Q9368' AND t.review_status = 'approved' AND t.reviewed_at IS NOT NULL
		  AND l.language = 'hr' AND l.label = 'jetra' AND l.preferred`); n != 1 {
		t.Error("liver is not reviewed with its Croatian label")
	}

	out.Reset()
	if err := run(t.Context(), []string{"terminology", "export", "-o", csvPath, "-status", "unreviewed"}, &out); err != nil || !strings.Contains(out.String(), "terms exported: 1,") {
		t.Errorf("export the drafts after the review = %v, %q; want one term", err, out.String())
	}
}

func TestTerminology_RejectsAnUnknownCommandOrMissingArguments(t *testing.T) {
	ownerDB(t)
	dir := t.TempDir()
	for _, args := range [][]string{
		{"terminology"}, {"terminology", "translate"}, {"terminology", "import"},
		{"terminology", "import", filepath.Join(dir, "missing.jsonl")}, {"terminology", "import", writeFile(t, "terms.txt", "")},
		{"terminology", "import", writeFile(t, "terms.csv", "id,comments\n")},
		{"terminology", "export"}, {"terminology", "export", "-o", filepath.Join(dir, "x.csv"), "-status", "done"},
		{"terminology", "export", "-o", filepath.Join(dir, "no", "such", "dir.csv")},
	} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%q): error = nil", args)
		}
	}
}
