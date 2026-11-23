package xlsx_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/xlsx"
)

// book builds an XLSX file from its parts, as a spreadsheet program writes them.
func book(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const (
	workbook = `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Services" sheetId="2" r:id="rId7"/><sheet name="Other" sheetId="1" r:id="rId1"/></sheets></workbook>`
	rels = `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId7" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="/xl/worksheets/sheet2.xml"/>
</Relationships>`
	shared = `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="3" uniqueCount="3">
<si><t>name</t></si><si><t>price</t></si><si><r><t>Placeholder </t></r><r><rPr><b/></rPr><t>exam</t></r></si></sst>`
	first = `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="s"><v>1</v></c></row>
<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2" t="inlineStr"><is><t>EX-1</t></is></c><c r="C2"><v>25.5</v></c></row>
<row r="4"><c r="A4" t="str"><v>Formula text</v></c><c r="C4" t="b"><v>1</v></c></row>
</sheetData></worksheet>`
	other = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1"><c r="A1" t="inlineStr"><is><t>wrong sheet</t></is></c></row></sheetData></worksheet>`
)

func TestRows_ReadsTheFirstSheetOfTheWorkbookAsText(t *testing.T) {
	t.Parallel()
	data := book(t, map[string]string{
		"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels, "xl/sharedStrings.xml": shared,
		"xl/worksheets/sheet2.xml": first, "xl/worksheets/sheet1.xml": other,
	})
	got, err := xlsx.Rows(data)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"name", "", "price"},
		{"Placeholder exam", "EX-1", "25.5"},
		{},
		{"Formula text", "", "1"},
	}
	if len(got) != len(want) {
		t.Fatalf("Rows() = %q, want %q", got, want)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("row %d = %q, want %q", i+1, got[i], want[i])
		}
	}
}

func TestRows_ReadsAWorkbookWithoutSharedStrings(t *testing.T) {
	t.Parallel()
	data := book(t, map[string]string{
		"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels, "xl/worksheets/sheet2.xml": other,
	})
	got, err := xlsx.Rows(data)
	if err != nil || len(got) != 1 || !slices.Equal(got[0], []string{"wrong sheet"}) {
		t.Errorf("Rows() = %q, %v", got, err)
	}
}

func TestRows_RefusesWhatIsNotAWorkbook(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"not a zip":        []byte("name;price\n"),
		"no workbook":      book(t, map[string]string{"word/document.xml": "<w/>"}),
		"missing sheet":    book(t, map[string]string{"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels}),
		"bad shared index": book(t, map[string]string{"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels, "xl/sharedStrings.xml": shared, "xl/worksheets/sheet2.xml": `<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>9</v></c></row></sheetData></worksheet>`}),
		"bad reference":    book(t, map[string]string{"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels, "xl/worksheets/sheet2.xml": `<worksheet><sheetData><row r="1"><c r="11"><v>1</v></c></row></sheetData></worksheet>`}),
	} {
		if _, err := xlsx.Rows(data); !errors.Is(err, xlsx.ErrNotWorkbook) {
			t.Errorf("%s: Rows() = %v, want ErrNotWorkbook", name, err)
		}
	}
}
