package terminology

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// The files of the curation round trip (docs/terminology/SPEC.md §3 "Curation"): the seed that tools/terminology
// writes (JSON Lines, assets/terminology/README.md) and the curator's spreadsheet (CSV, one row per term).

// bom starts the CSV so that Excel reads it as UTF-8 (Cyrillic, Arabic); Google Sheets ignores it.
const bom = "\ufeff"

// synonymSeparator joins a language's synonyms in one cell; labels never contain "|" (normaliseLabels).
const synonymSeparator = " | "

// csvColumns are the CSV's columns in the order Export writes them; an import takes them in any order.
var csvColumns = func() []string {
	cols := []string{"id", "category", "wikidata", "icd10", "ta98", "atc", "pcs", "rank"}
	cols = append(cols, languages...)
	for _, l := range languages {
		cols = append(cols, l+" synonyms")
	}
	return append(cols, "review_status", "note")
}()

// seedTerm is one line of the seed.
type seedTerm struct {
	Labels   map[string][]string `json:"labels"`
	Wikidata string              `json:"wikidata"`
	Category string              `json:"category"`
	ICD10    string              `json:"icd10"`
	TA98     string              `json:"ta98"`
	ATC      string              `json:"atc"`
	PCS      string              `json:"pcs"`
	Rank     int32               `json:"rank"`
}

// DecodeSeed reads the seed: one term per line, its first label in a language the preferred one. A line that is
// not a seed term is refused; blank lines are skipped.
func DecodeSeed(r io.Reader) ([]ImportRow, []RowError, error) {
	var rows []ImportRow
	var refused []RowError
	lines := bufio.NewScanner(r)
	lines.Buffer(nil, 1<<20)
	for n := 1; lines.Scan(); n++ {
		line := bytes.TrimSpace(lines.Bytes())
		if len(line) == 0 {
			continue
		}
		var s seedTerm
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s); err != nil || dec.More() {
			refused = append(refused, RowError{Line: n, Reason: "not a seed term"})
			continue
		}
		t := Term{Category: s.Category, WikidataID: s.Wikidata, ICD10: s.ICD10, TA98: s.TA98, ATC: s.ATC, PCS: s.PCS, Rank: s.Rank}
		for _, lang := range seedLanguages(s.Labels) {
			for i, text := range s.Labels[lang] {
				t.Labels = append(t.Labels, Label{Language: lang, Text: text, Preferred: i == 0})
			}
		}
		rows = append(rows, ImportRow{Term: t, Line: n})
	}
	if err := lines.Err(); err != nil {
		return nil, nil, fmt.Errorf("terminology: reading the seed: %w", err)
	}
	return rows, refused, nil
}

// seedLanguages are the languages of labels in the enum's order, then any other key, which the import refuses.
func seedLanguages(labels map[string][]string) []string {
	var out, other []string
	for _, l := range languages {
		if _, ok := labels[l]; ok {
			out = append(out, l)
		}
	}
	for l := range labels {
		if !slices.Contains(languages, l) {
			other = append(other, l)
		}
	}
	slices.Sort(other)
	return append(out, other...)
}

// csvNote is the first line of the curator's CSV, one cell (owner's answer, 2026-11-08); DecodeCSV skips the lines
// before the header whose first cell starts with #.
const csvNote = `# For the curator: anatomy Latin follows Terminologia Anatomica (TA98), e.g. "hepar", not "iecur". ` +
	`Fill in missing Bosnian, Croatian and Serbian terms, such as "čir". Set review_status to reviewed when a row ` +
	`is checked. The import skips the lines that start with #.`

// EncodeCSV writes terms as the curator's CSV: a byte order mark, the curator's note, the header, then one row
// per term with the preferred label per language and its synonyms joined with " | ". A cell a spreadsheet would
// read as a formula gets a leading apostrophe, which DecodeCSV removes.
func EncodeCSV(w io.Writer, terms []Term) error {
	if _, err := io.WriteString(w, bom); err != nil {
		return err
	}
	out := csv.NewWriter(w)
	if err := out.Write([]string{csvNote}); err != nil {
		return err
	}
	if err := out.Write(csvColumns); err != nil {
		return err
	}
	for _, t := range terms {
		preferred, synonyms := map[string]string{}, map[string][]string{}
		for _, l := range t.Labels {
			if l.Preferred {
				preferred[l.Language] = l.Text
			} else {
				synonyms[l.Language] = append(synonyms[l.Language], l.Text)
			}
		}
		rec := []string{t.ID, t.Category, t.WikidataID, t.ICD10, t.TA98, t.ATC, t.PCS, strconv.Itoa(int(t.Rank))}
		for _, l := range languages {
			rec = append(rec, preferred[l])
		}
		for _, l := range languages {
			rec = append(rec, strings.Join(synonyms[l], synonymSeparator))
		}
		rec = append(rec, t.ReviewStatus, t.Note)
		for i := range rec {
			rec[i] = guard(rec[i])
		}
		if err := out.Write(rec); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

// DecodeCSV reads the curator's CSV, skipping the note lines before the header (a first cell that starts with #,
// however a spreadsheet quoted or padded it). The header must name every column of EncodeCSV once and no other, in any
// order; otherwise the whole file is refused, since a column renamed in a spreadsheet would empty the labels of
// every row. A row with another number of cells or a rank that is not a whole number is refused.
func DecodeCSV(r io.Reader) ([]ImportRow, []RowError, error) {
	in := bufio.NewReader(r)
	if b, err := in.Peek(len(bom)); err == nil && string(b) == bom {
		_, _ = in.Discard(len(bom))
	}
	cr := csv.NewReader(in)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	for err == nil && len(header) > 0 && strings.HasPrefix(strings.TrimSpace(header[0]), "#") {
		header, err = cr.Read()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("terminology: the CSV has no header: %w", err)
	}
	col, err := csvIndex(header)
	if err != nil {
		return nil, nil, err
	}
	var rows []ImportRow
	var refused []RowError
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return rows, refused, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("terminology: reading the CSV: %w", err)
		}
		line, _ := cr.FieldPos(0)
		t, reason := csvTerm(rec, col, len(header))
		if reason != "" {
			var id string
			if col["id"] < len(rec) {
				id = strings.TrimSpace(rec[col["id"]])
			}
			refused = append(refused, RowError{ID: id, Line: line, Reason: reason})
			continue
		}
		rows = append(rows, ImportRow{Term: t, Line: line})
	}
}

// csvIndex maps each column to its position in header, or says which column is unknown, repeated or missing.
func csvIndex(header []string) (map[string]int, error) {
	col := map[string]int{}
	for i, name := range header {
		name = strings.TrimSpace(name)
		if !slices.Contains(csvColumns, name) {
			return nil, fmt.Errorf("terminology: the CSV has an unknown column %q", name)
		}
		if _, ok := col[name]; ok {
			return nil, fmt.Errorf("terminology: the CSV has the column %q twice", name)
		}
		col[name] = i
	}
	for _, name := range csvColumns {
		if _, ok := col[name]; !ok {
			return nil, fmt.Errorf("terminology: the CSV has no column %q", name)
		}
	}
	return col, nil
}

// csvTerm is the term in rec, or the reason the row is refused.
func csvTerm(rec []string, col map[string]int, cells int) (Term, string) {
	if len(rec) != cells {
		return Term{}, fmt.Sprintf("%d cells, the header has %d", len(rec), cells)
	}
	cell := func(name string) string { return strings.TrimSpace(unguard(rec[col[name]])) }
	var rank int64
	if s := cell("rank"); s != "" {
		var err error
		if rank, err = strconv.ParseInt(s, 10, 32); err != nil {
			return Term{}, "rank: not a whole number"
		}
	}
	t := Term{
		ID: cell("id"), Category: cell("category"), WikidataID: cell("wikidata"), ICD10: cell("icd10"),
		TA98: cell("ta98"), ATC: cell("atc"), PCS: cell("pcs"), ReviewStatus: cell("review_status"),
		Note: cell("note"), Rank: int32(rank), Labels: []Label{},
	}
	for _, l := range languages {
		if p := cell(l); p != "" {
			t.Labels = append(t.Labels, Label{Language: l, Text: p, Preferred: true})
		}
		for _, s := range strings.Split(cell(l+" synonyms"), "|") {
			if s = strings.TrimSpace(s); s != "" {
				t.Labels = append(t.Labels, Label{Language: l, Text: s})
			}
		}
	}
	return t, ""
}

// formulaStart are the first characters that make a spreadsheet read a cell as a formula.
const formulaStart = "=+-@\t\r"

// guard puts an apostrophe before a cell a spreadsheet would run as a formula.
func guard(s string) string {
	if s != "" && strings.ContainsRune(formulaStart, rune(s[0])) {
		return "'" + s
	}
	return s
}

// unguard removes the apostrophe guard put there.
func unguard(s string) string {
	if len(s) > 1 && s[0] == '\'' && strings.ContainsRune(formulaStart, rune(s[1])) {
		return s[1:]
	}
	return s
}
