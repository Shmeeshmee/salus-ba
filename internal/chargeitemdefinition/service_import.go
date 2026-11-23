package chargeitemdefinition

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/xlsx"
)

// maxImportRows bounds the services of one import.
const maxImportRows = 1000

// ImportFile is a price list to import (package O2 of docs/clinic-cms): a CSV or XLSX file of services (name, code,
// price, duration, VAT rate, visit types), told apart by its bytes.
type ImportFile struct {
	Name string
	Data []byte
}

// ImportRow is one row of an import as read: the service it would store and the codes of what is wrong with it
// (empty when it is valid).
type ImportRow struct {
	Errors  []string
	Service Definition
	Line    int // the row's line in the file, from 1
}

// headers are the column names an import understands, in English and the staff's languages, lower case.
var headers = map[string]string{
	"name": "name", "naziv": "name", "назив": "name",
	"code": "code", "šifra": "code", "sifra": "code", "шифра": "code",
	"price": "price", "cijena": "price", "cena": "price", "цијена": "price", "цена": "price",
	"duration": "duration", "trajanje": "duration", "трајање": "duration",
	"vat": "vat", "pdv": "vat", "пдв": "vat",
	"modes": "modes", "mode": "modes", "način": "modes", "nacin": "modes", "начин": "modes",
}

// modeNames are the words a visit-mode cell may hold, lower case.
var modeNames = map[string]schedule.VisitMode{
	"in-person": schedule.ModeInPerson, "in_person": schedule.ModeInPerson, "inperson": schedule.ModeInPerson,
	"clinic": schedule.ModeInPerson, "ordinacija": schedule.ModeInPerson, "uživo": schedule.ModeInPerson,
	"уживо": schedule.ModeInPerson, "ординација": schedule.ModeInPerson,
	"video": schedule.ModeVideo, "online": schedule.ModeVideo, "видео": schedule.ModeVideo,
}

// record is a non-blank row of the file and its line.
type record struct {
	cells []string
	line  int
}

// Import reads f into the admin caller's clinic's price list. With save false it only previews every row and its
// errors; with save true it stores all rows (each audited as a create) or, when any row has an error, none. Imported
// services are active. ErrForbidden for other staff and ErrNotFound for anyone else, audited; ErrInvalidImport for a
// file it cannot read.
func (s *Service) Import(ctx context.Context, c Caller, f ImportFile, save bool) ([]ImportRow, error) {
	if err := adminOf(c); err != nil {
		return nil, s.refused(ctx, c, auditevent.CodeCreate, auditevent.ActionCreate, err)
	}
	records, err := read(f.Data)
	if err != nil {
		return nil, err
	}
	columns, err := columnsOf(records[0].cells)
	if err != nil {
		return nil, err
	}
	var rows []ImportRow
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		existing, err := s.Repo.List(ctx, c.OrganizationID)
		if err != nil {
			return err
		}
		rows = parseRows(records[1:], columns, existing, c.OrganizationID)
		if !save || slices.ContainsFunc(rows, func(r ImportRow) bool { return len(r.Errors) > 0 }) {
			return nil
		}
		for i, r := range rows {
			if rows[i].Service, err = s.Repo.Create(ctx, r.Service, s.Clock.Now()); err != nil {
				return err
			}
			if err := s.Audit.Record(ctx, event(auditevent.CodeCreate, auditevent.ActionCreate, c, rows[i].Service.ID, auditevent.OutcomeSuccess)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("chargeitemdefinition: import: %w", err)
	}
	return rows, nil
}

// read returns the file's non-blank rows, the header first: an XLSX file by its zip signature, otherwise a UTF-8 CSV
// whose delimiter (semicolon, comma or tab) is the one its first line uses most.
func read(data []byte) ([]record, error) {
	var out []record
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		all, err := xlsx.Rows(data)
		if err != nil {
			return nil, ErrInvalidImport
		}
		for i, cells := range all {
			out = append(out, record{cells: cells, line: i + 1})
		}
	} else {
		data = bytes.TrimPrefix(data, []byte("\ufeff"))
		if !utf8.Valid(data) {
			return nil, ErrInvalidImport
		}
		first, _, _ := bytes.Cut(data, []byte("\n"))
		r := csv.NewReader(bytes.NewReader(data))
		r.FieldsPerRecord, r.Comma = -1, ';'
		for _, d := range []rune{',', '\t'} {
			if bytes.Count(first, []byte(string(d))) > bytes.Count(first, []byte(string(r.Comma))) {
				r.Comma = d
			}
		}
		for {
			cells, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, ErrInvalidImport
			}
			line, _ := r.FieldPos(0)
			out = append(out, record{cells: cells, line: line})
		}
	}
	out = slices.DeleteFunc(out, func(r record) bool {
		return !slices.ContainsFunc(r.cells, func(c string) bool { return strings.TrimSpace(c) != "" })
	})
	if len(out) < 2 || len(out) > maxImportRows+1 {
		return nil, ErrInvalidImport
	}
	return out, nil
}

// columnsOf maps each known column to its index in the header, or ErrInvalidImport for a header without the name
// and price columns or naming one twice.
func columnsOf(header []string) (map[string]int, error) {
	out := map[string]int{}
	for i, h := range header {
		name, ok := headers[strings.ToLower(strings.TrimSpace(h))]
		if !ok {
			continue
		}
		if _, twice := out[name]; twice {
			return nil, ErrInvalidImport
		}
		out[name] = i
	}
	if _, ok := out["name"]; !ok {
		return nil, ErrInvalidImport
	}
	if _, ok := out["price"]; !ok {
		return nil, ErrInvalidImport
	}
	return out, nil
}

// parseRows reads each record as a service of the clinic org, with the codes of what is wrong with it. A name or a
// code already in the price list or on an earlier row is a duplicate (case does not matter).
func parseRows(records []record, columns map[string]int, existing []Definition, org string) []ImportRow {
	names, codes := map[string]bool{}, map[string]bool{}
	for _, d := range existing {
		names[strings.ToLower(d.Name)] = true
		if d.Code != "" {
			codes[strings.ToLower(d.Code)] = true
		}
	}
	out := make([]ImportRow, 0, len(records))
	for _, rec := range records {
		cell := func(name string) string {
			if i, ok := columns[name]; ok && i < len(rec.cells) {
				return strings.TrimSpace(rec.cells[i])
			}
			return ""
		}
		row := parseRow(cell, org)
		row.Line = rec.line
		name, code := strings.ToLower(row.Service.Name), strings.ToLower(row.Service.Code)
		if name != "" && names[name] || code != "" && codes[code] {
			row.Errors = append(row.Errors, "duplicate")
		}
		names[name], codes[code] = true, code != ""
		out = append(out, row)
	}
	return out
}

// parseRow reads one row through cell, which returns a column's trimmed text.
func parseRow(cell func(string) string, org string) ImportRow {
	d := Definition{OrganizationID: org, Name: cell("name"), Code: cell("code"), Currency: currencyBAM, Active: true}
	var errs []string
	switch n := utf8.RuneCountInString(d.Name); {
	case n == 0:
		errs = append(errs, "name_missing")
	case n > maxNameLength:
		errs = append(errs, "name_invalid")
	}
	if utf8.RuneCountInString(d.Code) > maxCodeLength {
		errs = append(errs, "code_invalid")
	}
	price := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(cell("price")), "KM"), "BAM"))
	var ok bool
	if d.PriceMinor, ok = hundredths(price); !ok || d.PriceMinor > maxPriceMinor {
		errs = append(errs, "price_invalid")
	}
	if v := cell("duration"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if d.DurationMinutes = int32(n); err != nil || n < minDuration || n > maxDuration {
			errs = append(errs, "duration_invalid")
		}
	}
	if v := strings.TrimSpace(strings.TrimSuffix(cell("vat"), "%")); v != "" {
		bp, ok := hundredths(v)
		if !ok || bp > maxVATRateBP {
			errs = append(errs, "vat_invalid")
		}
		d.VATRateBP = ptrTo(int32(min(bp, maxVATRateBP)))
	}
	if d.Modes, ok = modesOf(cell("modes")); !ok {
		errs = append(errs, "mode_invalid")
	}
	return ImportRow{Service: d, Errors: errs}
}

// hundredths reads a non-negative decimal with at most two decimals, the separator a point or a comma, in hundredths:
// "25,50" is 2550. False for anything else.
func hundredths(v string) (int64, bool) {
	whole, frac, _ := strings.Cut(strings.Replace(v, ",", ".", 1), ".")
	if whole == "" || len(whole) > 12 || len(frac) > 2 || !digits(whole) || !digits(frac) {
		return 0, false
	}
	frac += strings.Repeat("0", 2-len(frac))
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	return n, err == nil
}

func digits(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// modesOf reads the visit modes of a cell, separated by spaces, commas, slashes or bars, sorted without
// repeats; an empty cell is in-person. False for a word that is not a mode.
func modesOf(v string) ([]schedule.VisitMode, bool) {
	words := strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune(",;/|+", r) })
	if len(words) == 0 {
		return []schedule.VisitMode{schedule.ModeInPerson}, true
	}
	var out []schedule.VisitMode
	for _, w := range words {
		m, ok := modeNames[w]
		if !ok {
			return nil, false
		}
		if !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	slices.Sort(out)
	return out, true
}

func ptrTo(v int32) *int32 { return &v }
