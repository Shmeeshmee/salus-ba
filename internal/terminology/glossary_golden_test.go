package terminology_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// Synthetic terms for the glossary's golden files: two symptoms labelled in the patients' languages and Bosnian,
// and a term no line says.
const (
	headache = "00000000-0000-4000-8000-0000000000a1"
	fever    = "00000000-0000-4000-8000-0000000000a2"
	cough    = "00000000-0000-4000-8000-0000000000a3"
)

// goldenPhrases are synthetic placeholder lines a patient might say, one per language the golden files cover
// (SAL-5): German, Arabic and Turkish, each translated into Bosnian. They are no real patient's words.
var goldenPhrases = []struct{ lang, text string }{
	{lang: "de", text: "Ich habe seit drei Tagen starke Kopfschmerzen und Fieber."},
	{lang: "ar", text: "أعاني من صداع شديد وحمى منذ ثلاثة أيام."},
	{lang: "tr", text: "Üç gündür şiddetli baş ağrısı ve ateşim var."},
}

// The glossary built for each golden phrase from the database's terms, as the translation receives it
// (testdata/glossary_<lang>_bs.golden.json; -update rewrites them, then review the diff).
func TestGlossary_GoldenPhrases(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, err := owner.Exec(t.Context(), `
		INSERT INTO term (id, category, rank) VALUES
			('`+headache+`', 'symptom', 120), ('`+fever+`', 'symptom', 150), ('`+cough+`', 'symptom', 90);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES
			('`+headache+`', 'la', 'cephalalgia', true), ('`+headache+`', 'bs', 'glavobolja', true),
			('`+headache+`', 'de', 'Kopfschmerz', true), ('`+headache+`', 'ar', 'صداع', true),
			('`+headache+`', 'tr', 'baş ağrısı', true),
			('`+fever+`', 'bs', 'groznica', true), ('`+fever+`', 'bs', 'temperatura', false),
			('`+fever+`', 'de', 'Fieber', true), ('`+fever+`', 'ar', 'حمى', true), ('`+fever+`', 'tr', 'ateş', true),
			('`+cough+`', 'bs', 'kašalj', true), ('`+cough+`', 'de', 'Husten', true), ('`+cough+`', 'ar', 'سعال', true),
			('`+cough+`', 'tr', 'öksürük', true)`)
	if err != nil {
		t.Fatal(err)
	}
	svc := terminology.NewService(terminology.NewPostgresRepository(app), nil)
	for _, p := range goldenPhrases {
		entries, err := svc.Glossary(t.Context(), p.text, p.lang, "bs")
		if err != nil {
			t.Fatalf("%s: %v", p.lang, err)
		}
		got, err := json.MarshalIndent(struct {
			Line     string                      `json:"line"`
			From     string                      `json:"from"`
			To       string                      `json:"to"`
			Glossary []terminology.GlossaryEntry `json:"glossary"`
		}{Line: p.text, From: p.lang, To: "bs", Glossary: entries}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		golden(t, filepath.Join("testdata", "glossary_"+p.lang+"_bs.golden.json"), append(got, '\n'))
	}
}

// golden compares got with the golden file, or rewrites it with -update.
func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // a test's golden file
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // the test's own golden file
	if err != nil {
		t.Fatalf("%v (run with -update to write it)", err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs:\ngot  %s\nwant %s", path, got, want)
	}
}
