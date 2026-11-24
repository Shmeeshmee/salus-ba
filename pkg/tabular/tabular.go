// Package tabular reads the rows of a table saved from a spreadsheet program: an XLSX file, or a CSV in UTF-8 or
// Windows-1250 (the encoding Excel in Bosnia and Herzegovina saves a CSV in) with a semicolon, comma or tab between
// its cells.
package tabular

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/pkg/xlsx"
)

// ErrUnreadable is returned for a file that is neither an XLSX file nor a CSV it can decode.
var ErrUnreadable = errors.New("tabular: unreadable file")

// Row is a non-blank row of a file and its line, from 1.
type Row struct {
	Cells []string
	Line  int
}

// Read returns the file's non-blank rows in order: an XLSX file by its zip signature (its first sheet, at most a
// million cells, see xlsx.Rows), otherwise a CSV in UTF-8 (a byte order mark is dropped) or Windows-1250 whose
// delimiter (semicolon, comma or tab) is the one its first line uses most. ErrUnreadable for anything else.
func Read(data []byte) ([]Row, error) {
	var out []Row
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		all, err := xlsx.Rows(data)
		if err != nil {
			return nil, ErrUnreadable
		}
		for i, cells := range all {
			out = append(out, Row{Cells: cells, Line: i + 1})
		}
	} else {
		var err error
		if out, err = readCSV(data); err != nil {
			return nil, err
		}
	}
	return slices.DeleteFunc(out, func(r Row) bool {
		return !slices.ContainsFunc(r.Cells, func(c string) bool { return strings.TrimSpace(c) != "" })
	}), nil
}

func readCSV(data []byte) ([]Row, error) {
	data, ok := utf8Text(bytes.TrimPrefix(data, []byte("\ufeff")))
	if !ok {
		return nil, ErrUnreadable
	}
	first, _, _ := bytes.Cut(data, []byte("\n"))
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord, r.Comma = -1, ';'
	for _, d := range []rune{',', '\t'} {
		if bytes.Count(first, []byte(string(d))) > bytes.Count(first, []byte(string(r.Comma))) {
			r.Comma = d
		}
	}
	var out []Row
	for {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, ErrUnreadable
		}
		line, _ := r.FieldPos(0)
		out = append(out, Row{Cells: cells, Line: line})
	}
}

// windows1250 maps the bytes 0x80 to 0xFF of Windows-1250 to their runes; 0 marks the five bytes it leaves undefined.
var windows1250 = [128]rune{
	0x20AC, 0x0000, 0x201A, 0x0000, 0x201E, 0x2026, 0x2020, 0x2021,
	0x0000, 0x2030, 0x0160, 0x2039, 0x015A, 0x0164, 0x017D, 0x0179,
	0x0000, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x0000, 0x2122, 0x0161, 0x203A, 0x015B, 0x0165, 0x017E, 0x017A,
	0x00A0, 0x02C7, 0x02D8, 0x0141, 0x00A4, 0x0104, 0x00A6, 0x00A7,
	0x00A8, 0x00A9, 0x015E, 0x00AB, 0x00AC, 0x00AD, 0x00AE, 0x017B,
	0x00B0, 0x00B1, 0x02DB, 0x0142, 0x00B4, 0x00B5, 0x00B6, 0x00B7,
	0x00B8, 0x0105, 0x015F, 0x00BB, 0x013D, 0x02DD, 0x013E, 0x017C,
	0x0154, 0x00C1, 0x00C2, 0x0102, 0x00C4, 0x0139, 0x0106, 0x00C7,
	0x010C, 0x00C9, 0x0118, 0x00CB, 0x011A, 0x00CD, 0x00CE, 0x010E,
	0x0110, 0x0143, 0x0147, 0x00D3, 0x00D4, 0x0150, 0x00D6, 0x00D7,
	0x0158, 0x016E, 0x00DA, 0x0170, 0x00DC, 0x00DD, 0x0162, 0x00DF,
	0x0155, 0x00E1, 0x00E2, 0x0103, 0x00E4, 0x013A, 0x0107, 0x00E7,
	0x010D, 0x00E9, 0x0119, 0x00EB, 0x011B, 0x00ED, 0x00EE, 0x010F,
	0x0111, 0x0144, 0x0148, 0x00F3, 0x00F4, 0x0151, 0x00F6, 0x00F7,
	0x0159, 0x016F, 0x00FA, 0x0171, 0x00FC, 0x00FD, 0x0163, 0x02D9,
}

// utf8Text returns data as UTF-8: unchanged when it is valid UTF-8, otherwise decoded as Windows-1250, the encoding
// Excel in Bosnia and Herzegovina saves a CSV in. ok is false for a byte Windows-1250 does not define.
func utf8Text(data []byte) (text []byte, ok bool) {
	if utf8.Valid(data) {
		return data, true
	}
	text = make([]byte, 0, len(data)+len(data)/2)
	for _, b := range data {
		r := rune(b)
		if b >= 0x80 {
			if r = windows1250[b-0x80]; r == 0 {
				return nil, false
			}
		}
		text = utf8.AppendRune(text, r)
	}
	return text, true
}
