package tabular_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/tabular"
)

func TestRead_ReadsACSVByTheDelimiterOfItsFirstLine(t *testing.T) {
	for name, data := range map[string]string{
		"semicolon": "\ufeffa;b\n1;\"x;y\"\n\n ; \n2;z\n",
		"comma":     "a,b\n1,\"x;y\"\n\n , \n2,z\n",
		"tab":       "a\tb\n1\t\"x;y\"\n\n \t \n2\tz\n",
	} {
		rows, err := tabular.Read([]byte(data))
		want := []tabular.Row{{Cells: []string{"a", "b"}, Line: 1}, {Cells: []string{"1", "x;y"}, Line: 2}, {Cells: []string{"2", "z"}, Line: 5}}
		if err != nil || !slices.EqualFunc(rows, want, same) {
			t.Errorf("%s: Read() = %+v, %v, want %+v", name, rows, err, want)
		}
	}
}

func same(a, b tabular.Row) bool { return a.Line == b.Line && slices.Equal(a.Cells, b.Cells) }

func TestRead_DecodesWindows1250(t *testing.T) {
	// "Šifra;Čćžđ" and "–" as Excel in Bosnia and Herzegovina saves them.
	rows, err := tabular.Read([]byte("\x8aifra;\xc8\xe6\x9e\xf0\n\x96;x\n"))
	want := []tabular.Row{{Cells: []string{"Šifra", "Čćžđ"}, Line: 1}, {Cells: []string{"–", "x"}, Line: 2}}
	if err != nil || !slices.EqualFunc(rows, want, same) {
		t.Errorf("Read(cp1250) = %+v, %v, want %+v", rows, err, want)
	}
}

func TestRead_ReadsAnXLSXFile(t *testing.T) {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"xl/workbook.xml":            `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="S" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
			`<row r="1"><c r="A1" t="inlineStr"><is><t>a</t></is></c></row><row r="3"><c r="A3"><v>7</v></c></row></sheetData></worksheet>`,
	} {
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
	rows, err := tabular.Read(buf.Bytes())
	want := []tabular.Row{{Cells: []string{"a"}, Line: 1}, {Cells: []string{"7"}, Line: 3}}
	if err != nil || !slices.EqualFunc(rows, want, same) {
		t.Errorf("Read(xlsx) = %+v, %v, want %+v", rows, err, want)
	}
}

func TestRead_RefusesAFileItCannotRead(t *testing.T) {
	for name, data := range map[string]string{
		"not UTF-8 or 1250": "a;b\nx\x81;1\n",
		"broken CSV":        "a,b\n\"x,1\n",
		"broken XLSX":       "PK\x03\x04 not really",
	} {
		if _, err := tabular.Read([]byte(data)); !errors.Is(err, tabular.ErrUnreadable) {
			t.Errorf("%s: Read() = %v, want ErrUnreadable", name, err)
		}
	}
}

func TestRead_ReturnsNothingForAnEmptyFile(t *testing.T) {
	if rows, err := tabular.Read(nil); err != nil || len(rows) != 0 {
		t.Errorf("Read(nil) = %+v, %v", rows, err)
	}
}
