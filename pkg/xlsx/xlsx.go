// Package xlsx reads the cells of an Office Open XML workbook's first sheet as text: enough to import a table someone
// typed into a spreadsheet program. Formulas are read as their last computed value; styles, dates (left as the serial
// number) and every other sheet are not read.
package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
)

// The limits of what is read: a part's size once inflated, and the rows and columns of the sheet.
const (
	maxPart    = 16 << 20
	MaxRows    = 10000
	maxColumns = 16384 // XFD, a spreadsheet's last column
)

var (
	// ErrNotWorkbook is returned for bytes that are not a workbook with a readable first sheet.
	ErrNotWorkbook = errors.New("xlsx: not a workbook")
	// ErrTooLarge is returned for a part larger than 16 MiB once inflated, or a sheet with a row after MaxRows.
	ErrTooLarge = errors.New("xlsx: too large")
)

type workbookXML struct {
	Sheets []struct {
		RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	} `xml:"sheets>sheet"`
}

type relsXML struct {
	Relationships []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// textXML is a string item: plain text, or rich text in runs.
type textXML struct {
	T    string `xml:"t"`
	Runs []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (t textXML) String() string {
	var b strings.Builder
	b.WriteString(t.T)
	for _, r := range t.Runs {
		b.WriteString(r.T)
	}
	return b.String()
}

type sharedXML struct {
	Items []textXML `xml:"si"`
}

type sheetXML struct {
	Rows []struct {
		Cells []struct {
			Ref    string  `xml:"r,attr"`
			Type   string  `xml:"t,attr"`
			Value  string  `xml:"v"`
			Inline textXML `xml:"is"`
		} `xml:"c"`
		R int `xml:"r,attr"`
	} `xml:"sheetData>row"`
}

// Rows returns the cells of the workbook's first sheet as text, row by row from row 1: a row without cells is
// empty, and a row ends at its last cell. ErrNotWorkbook for data that is not one, ErrTooLarge past the limits.
func Rows(data []byte) ([][]string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotWorkbook
	}
	var wb workbookXML
	var rels relsXML
	if err := part(z, "xl/workbook.xml", &wb); err != nil {
		return nil, notFound(err)
	}
	if err := part(z, "xl/_rels/workbook.xml.rels", &rels); err != nil {
		return nil, notFound(err)
	}
	if len(wb.Sheets) == 0 {
		return nil, ErrNotWorkbook
	}
	target := ""
	for _, r := range rels.Relationships {
		if r.ID == wb.Sheets[0].RID {
			target = r.Target
		}
	}
	if strings.HasPrefix(target, "/") {
		target = path.Clean(target[1:])
	} else {
		target = path.Join("xl", target)
	}
	var shared sharedXML
	if err := part(z, "xl/sharedStrings.xml", &shared); err != nil && !errors.Is(err, errMissing) {
		return nil, err
	}
	var sheet sheetXML
	if err := part(z, target, &sheet); err != nil {
		return nil, notFound(err)
	}
	return cells(sheet, shared)
}

// cells lays the sheet's cells out by their references.
func cells(sheet sheetXML, shared sharedXML) ([][]string, error) {
	var out [][]string
	for _, row := range sheet.Rows {
		r := row.R
		if r == 0 {
			r = len(out) + 1
		}
		if r < len(out)+1 {
			return nil, ErrNotWorkbook
		}
		if r > MaxRows {
			return nil, ErrTooLarge
		}
		for len(out) < r {
			out = append(out, nil)
		}
		var line []string
		for _, c := range row.Cells {
			col := len(line)
			if c.Ref != "" {
				var ok bool
				if col, ok = column(c.Ref); !ok || col < len(line) {
					return nil, ErrNotWorkbook
				}
			}
			text := c.Value
			switch c.Type {
			case "s":
				i, err := strconv.Atoi(c.Value)
				if err != nil || i < 0 || i >= len(shared.Items) {
					return nil, ErrNotWorkbook
				}
				text = shared.Items[i].String()
			case "inlineStr":
				text = c.Inline.String()
			}
			for len(line) < col {
				line = append(line, "")
			}
			line = append(line, text)
		}
		out[r-1] = line
	}
	return out, nil
}

// column is the zero-based column of a cell reference such as AB12, and false for one that is not a reference.
func column(ref string) (int, bool) {
	col, i := 0, 0
	for ; i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z'; i++ {
		col = col*26 + int(ref[i]-'A'+1)
		if col > maxColumns {
			return 0, false
		}
	}
	if i == 0 || i == len(ref) {
		return 0, false
	}
	if _, err := strconv.Atoi(ref[i:]); err != nil {
		return 0, false
	}
	return col - 1, true
}

// errMissing is a part the archive does not hold.
var errMissing = errors.New("xlsx: missing part")

// part decodes the archive's part name into v: errMissing for no such part, ErrTooLarge past maxPart,
// ErrNotWorkbook for XML that does not decode.
func part(z *zip.Reader, name string, v any) error {
	f, err := z.Open(name)
	if err != nil {
		return errMissing
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	data, err := io.ReadAll(io.LimitReader(f, maxPart+1))
	switch {
	case err != nil:
		return ErrNotWorkbook
	case len(data) > maxPart:
		return ErrTooLarge
	}
	if xml.Unmarshal(data, v) != nil {
		return ErrNotWorkbook
	}
	return nil
}

// notFound is ErrNotWorkbook for a missing part, err otherwise.
func notFound(err error) error {
	if errors.Is(err, errMissing) {
		return ErrNotWorkbook
	}
	return err
}
