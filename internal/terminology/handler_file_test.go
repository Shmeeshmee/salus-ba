package terminology_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

func TestDecodeSeed(t *testing.T) {
	t.Parallel()
	seed := `{"labels":{"la":["hepar","iecur"],"en":["liver"]},"wikidata":"Q9368","category":"anatomy","ta98":"A05.8.01.001","rank":180}

{"labels":{"en":["fever"]},"wikidata":"Q38933","category":"symptom","icd10":"R50.9","atc":"x","pcs":"y","rank":120}
{"labels":
{"labels":{},"wikidata":"Q1","category":"drug","rank":1,"colour":"red"}
`

	rows, refused, err := terminology.DecodeSeed(strings.NewReader(seed))

	want := []terminology.ImportRow{
		{Line: 1, Term: terminology.Term{Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001", Rank: 180, Labels: labels("la", "hepar", "la", "iecur", "en", "liver")}},
		{Line: 3, Term: terminology.Term{Category: "symptom", WikidataID: "Q38933", ICD10: "R50.9", ATC: "x", PCS: "y", Rank: 120, Labels: labels("en", "fever")}},
	}
	wantRefused := []terminology.RowError{{Line: 4, Reason: "not a seed term"}, {Line: 5, Reason: "not a seed term"}}
	if err != nil || !reflect.DeepEqual(rows, want) || !reflect.DeepEqual(refused, wantRefused) {
		t.Errorf("DecodeSeed = %+v, %+v, %v;\nwant %+v, %+v", rows, refused, err, want, wantRefused)
	}
}

// The terms of the CSV tests: a reviewed term with Latin synonyms, a draft with a note, a comma and a label a
// spreadsheet would read as a formula.
var csvTerms = []terminology.Term{
	{
		ReviewedAt: time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC), ID: "00000000-0000-4000-8000-000000000001",
		Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001", ReviewStatus: "reviewed", Rank: 180,
		Labels: labels("la", "hepar", "la", "iecur", "la", "jecur", "sr-Cyrl", "јетра", "en", "liver"),
	},
	{
		ID: "00000000-0000-4000-8000-000000000002", Category: "drug", WikidataID: "Q180809", ATC: "A11HA05",
		ReviewStatus: "draft", Note: "Check the Latin, then mark reviewed.", Rank: 40,
		Labels: labels("en", "-(+)-biotin", "en", "vitamin B7, biotin"),
	},
}

const csvHeader = "id,category,wikidata,icd10,ta98,atc,pcs,rank," +
	"la,bs,hr,sr-Latn,sr-Cyrl,en,de,ar,tr,it,fr," +
	"la synonyms,bs synonyms,hr synonyms,sr-Latn synonyms,sr-Cyrl synonyms,en synonyms,de synonyms,ar synonyms,tr synonyms,it synonyms,fr synonyms," +
	"review_status,note\n"

func TestEncodeCSV(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	if err := terminology.EncodeCSV(&out, csvTerms); err != nil {
		t.Fatalf("EncodeCSV: %v", err)
	}

	want := "\ufeff" + csvNote + csvHeader +
		"00000000-0000-4000-8000-000000000001,anatomy,Q9368,,A05.8.01.001,,,180,hepar,,,,јетра,liver,,,,,,iecur | jecur,,,,,,,,,,,reviewed,\n" +
		"00000000-0000-4000-8000-000000000002,drug,Q180809,,,A11HA05,,40,,,,,,'-(+)-biotin,,,,,,,,,,,\"vitamin B7, biotin\",,,,,,draft,\"Check the Latin, then mark reviewed.\"\n"
	if out.String() != want {
		t.Errorf("EncodeCSV =\n%s\nwant\n%s", out.String(), want)
	}
}

// What EncodeCSV writes, DecodeCSV reads back: the terms without their review dates.
func TestDecodeCSV_ReadsWhatEncodeCSVWrites(t *testing.T) {
	t.Parallel()
	var file bytes.Buffer
	if err := terminology.EncodeCSV(&file, csvTerms); err != nil {
		t.Fatal(err)
	}

	rows, refused, err := terminology.DecodeCSV(&file)

	if err != nil || len(refused) != 0 || len(rows) != 2 {
		t.Fatalf("DecodeCSV = %d rows, %v, %v; want 2 rows", len(rows), refused, err)
	}
	for i, r := range rows {
		want := csvTerms[i]
		want.ReviewedAt = time.Time{}
		if r.Line != i+3 || !reflect.DeepEqual(r.Term, want) {
			t.Errorf("row %d = line %d %+v,\nwant line %d %+v", i, r.Line, r.Term, i+3, want)
		}
	}
}

func TestDecodeCSV_TakesTheColumnsInAnyOrderAndRefusesABadRow(t *testing.T) {
	t.Parallel()
	columns := strings.Split(strings.TrimSpace(csvHeader), ",")
	reversed := make([]string, len(columns))
	for i, c := range columns {
		reversed[len(columns)-1-i] = c
	}
	cells := func(values map[string]string) string {
		out := make([]string, len(reversed))
		for i, c := range reversed {
			out[i] = values[c]
		}
		return strings.Join(out, ",")
	}
	file := strings.Join(reversed, ",") + "\n" +
		cells(map[string]string{"id": "i1", "category": "symptom", "rank": "7", "en": "fever", "en synonyms": " pyrexia |  | high temperature ", "review_status": "reviewed"}) + "\n" +
		cells(map[string]string{"id": "i2", "rank": "seven"}) + "\n" +
		"i3,too,few\n" +
		cells(map[string]string{"id": "i4", "category": "symptom"}) + "\n"

	rows, refused, err := terminology.DecodeCSV(strings.NewReader(file))

	want := []terminology.ImportRow{
		{Line: 2, Term: terminology.Term{ID: "i1", Category: "symptom", Rank: 7, ReviewStatus: "reviewed", Labels: labels("en", "fever", "en", "pyrexia", "en", "high temperature")}},
		{Line: 5, Term: terminology.Term{ID: "i4", Category: "symptom", Labels: []terminology.Label{}}},
	}
	wantRefused := []terminology.RowError{{ID: "i2", Line: 3, Reason: "rank: not a whole number"}, {Line: 4, Reason: "3 cells, the header has 32"}}
	if err != nil || !reflect.DeepEqual(rows, want) || !reflect.DeepEqual(refused, wantRefused) {
		t.Errorf("DecodeCSV = %+v, %+v, %v;\nwant %+v, %+v", rows, refused, err, want, wantRefused)
	}
}

// csvNote is the curator's note EncodeCSV writes before the header (owner's answer, 2026-11-08).
const csvNote = `"# For the curator: anatomy Latin follows Terminologia Anatomica (TA98), e.g. ""hepar"", not ""iecur"". ` +
	`Fill in missing Bosnian, Croatian and Serbian terms, such as ""čir"". Set review_status to reviewed when a ` +
	`row is checked. The import skips the lines that start with #."` + "\n"

// A spreadsheet saves the note back quoted or not, padded with empty cells; the import skips it wherever the
// first cell starts with # before the header, and counts its line.
func TestDecodeCSV_SkipsTheNoteLinesBeforeTheHeader(t *testing.T) {
	t.Parallel()
	row := "i1,symptom,,,,,,7,,,,,,fever,,,,,,,,,,,,,,,,,reviewed,"
	for _, c := range []struct{ name, note string }{
		{"as exported", csvNote},
		{"unquoted", "# For the curator: check each row\n"},
		{"padded by a spreadsheet", `"# For the curator: anatomy Latin, TA98",,,,,` + "\n"},
		{"two lines", "# one\n#two\n"},
	} {
		file := "\ufeff" + c.note + csvHeader + row + "\n"
		rows, refused, err := terminology.DecodeCSV(strings.NewReader(file))
		wantLine := strings.Count(c.note, "\n") + 2
		if err != nil || len(refused) != 0 || len(rows) != 1 || rows[0].Term.ID != "i1" || rows[0].Line != wantLine {
			t.Errorf("%s: DecodeCSV = %+v, %v, %v; want i1 on line %d", c.name, rows, refused, err, wantLine)
		}
	}
}

func TestDecodeCSV_RefusesAFileWithAnUnknownMissingOrRepeatedColumn(t *testing.T) {
	t.Parallel()
	for name, header := range map[string]string{
		"unknown":  strings.TrimSpace(csvHeader) + ",comments",
		"missing":  strings.TrimSuffix(strings.TrimSpace(csvHeader), ",note"),
		"repeated": strings.TrimSpace(csvHeader) + ",note",
		"empty":    "",
	} {
		if _, _, err := terminology.DecodeCSV(strings.NewReader(header + "\n")); err == nil {
			t.Errorf("DecodeCSV with a header with a column %s: error = nil", name)
		}
	}
}
