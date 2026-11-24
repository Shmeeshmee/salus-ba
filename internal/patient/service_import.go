package patient

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/tabular"
)

// maxImportRows bounds the patients of one import (docs/clinic-cms/WAVE4.md M3).
const maxImportRows = 5000

// ImportRow is one row of a patient import as read: the profile it would store (with its id once saved), the codes
// of what is wrong with it (empty when it is valid), and whether a patient of the same name and birth date is
// already the clinic's or on an earlier row (a likely duplicate, which is saved all the same).
type ImportRow struct {
	Errors    []string
	Patient   Patient
	Line      int // the row's line in the file, from 1
	Duplicate bool
}

// importHeaders are the column names an import understands, in English and the staff's languages, lower case.
var importHeaders = map[string]string{
	"given name": "given", "given_name": "given", "first name": "given", "ime": "given", "име": "given",
	"family name": "family", "family_name": "family", "last name": "family", "surname": "family",
	"prezime": "family", "презиме": "family",
	"birth date": "birth", "birth_date": "birth", "date of birth": "birth", "datum rođenja": "birth",
	"datum rodjenja": "birth", "datum rodenja": "birth", "датум рођења": "birth",
	"phone": "phone", "mobile": "phone", "telefon": "phone", "mobitel": "phone", "mobilni": "phone",
	"телефон": "phone", "мобилни": "phone",
}

// Import reads a CSV or XLSX file of patients (given name, family name, birth date and an optional phone) into the
// clinic of its admin c (package M3 of docs/clinic-cms). With save false it only previews every row, its errors and
// whether it is a likely duplicate; with save true it registers all rows as profiles of the clinic without a consent
// (V5: the desk records it at the first booking) or, when any row has an error, none. The clinic's patients a preview
// finds are audited as a search; each saved profile as a create, and the import once with its count. ErrForbidden to
// anyone else (audited), ErrInvalidImport for a file it cannot read.
func (s *Service) Import(ctx context.Context, c Caller, data []byte, save bool) ([]ImportRow, error) {
	if c.patient() || !c.Admin {
		if err := s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure); err != nil {
			return nil, fmt.Errorf("patient: import: %w", err)
		}
		return nil, ErrForbidden
	}
	records, err := tabular.Read(data)
	if err != nil || len(records) < 2 || len(records) > maxImportRows+1 {
		return nil, ErrInvalidImport
	}
	columns, err := importColumns(records[0].Cells)
	if err != nil {
		return nil, err
	}
	rows := s.parseImport(records[1:], columns)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := s.markDuplicates(ctx, c, rows); err != nil {
			return err
		}
		for _, r := range rows {
			if !save || len(r.Errors) > 0 {
				return nil
			}
		}
		return s.saveImport(ctx, c, rows)
	})
	if err != nil {
		return nil, fmt.Errorf("patient: import: %w", err)
	}
	return rows, nil
}

// importColumns maps each known column to its index in the header, or ErrInvalidImport for a header without the
// name and birth date columns or naming one twice.
func importColumns(header []string) (map[string]int, error) {
	out := map[string]int{}
	for i, h := range header {
		name, ok := importHeaders[strings.ToLower(strings.TrimSpace(h))]
		if !ok {
			continue
		}
		if _, twice := out[name]; twice {
			return nil, ErrInvalidImport
		}
		out[name] = i
	}
	for _, need := range []string{"given", "family", "birth"} {
		if _, ok := out[need]; !ok {
			return nil, ErrInvalidImport
		}
	}
	return out, nil
}

// parseImport reads each record as a profile, with the codes of what is wrong with it, by SPEC.md "Profile".
func (s *Service) parseImport(records []tabular.Row, columns map[string]int) []ImportRow {
	y, m, d := s.clock.Now().UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	out := make([]ImportRow, 0, len(records))
	for _, rec := range records {
		cell := func(name string) string {
			if i, ok := columns[name]; ok && i < len(rec.Cells) {
				return strings.TrimSpace(rec.Cells[i])
			}
			return ""
		}
		row := ImportRow{Line: rec.Line, Patient: Patient{GivenName: cell("given"), FamilyName: cell("family"), Phone: phoneOf(cell("phone"))}}
		if n := utf8.RuneCountInString(row.Patient.GivenName); n == 0 || n > maxName {
			row.Errors = append(row.Errors, "given_name_invalid")
		}
		if n := utf8.RuneCountInString(row.Patient.FamilyName); n == 0 || n > maxName {
			row.Errors = append(row.Errors, "family_name_invalid")
		}
		born, ok := birthDateOf(cell("birth"))
		if !ok || born.Before(earliestBirthDate) || !born.Before(today) {
			row.Errors = append(row.Errors, "birth_date_invalid")
		} else {
			row.Patient.BirthDate = born
		}
		if row.Patient.Phone != "" && !e164(row.Patient.Phone) {
			row.Errors = append(row.Errors, "phone_invalid")
		}
		out = append(out, row)
	}
	return out
}

// birthDateLayouts are the date forms a birth date cell may take besides an Excel date number.
var birthDateLayouts = []string{"2006-01-02", "2.1.2006", "2/1/2006"}

// birthDateOf reads a birth date as 1990-01-31, 31.01.1990 (with or without the final point), 31/01/1990 or an
// Excel date number (days since 1899-12-30, as an XLSX cell formatted as a date holds it).
func birthDateOf(v string) (time.Time, bool) {
	if n, err := strconv.Atoi(v); err == nil {
		return time.Date(1899, 12, 30+n, 0, 0, 0, 0, time.UTC), n > 0
	}
	v = strings.TrimSuffix(v, ".")
	for _, layout := range birthDateLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// phoneOf writes a phone cell as E.164 where it can: spaces, dashes, slashes, points and parentheses dropped, a
// leading 00 read as +, and a leading single 0 as a number in Bosnia and Herzegovina (+387), where the clinics are.
func phoneOf(v string) string {
	v = strings.Map(func(r rune) rune {
		if strings.ContainsRune("  -/.()", r) {
			return -1
		}
		return r
	}, v)
	switch {
	case strings.HasPrefix(v, "00"):
		return "+" + v[2:]
	case strings.HasPrefix(v, "0"):
		return "+387" + v[1:]
	}
	return v
}

// importKey is a profile's name and birth date as two imports of one person share them.
func importKey(p Patient) string {
	return strings.ToLower(p.GivenName) + "\x00" + strings.ToLower(p.FamilyName) + "\x00" + p.BirthDate.Format(dateLayout)
}

// matchable reports whether r has the name and birth date a duplicate is found by.
func matchable(r ImportRow) bool {
	return !slices.Contains(r.Errors, "given_name_invalid") && !slices.Contains(r.Errors, "family_name_invalid") && !r.Patient.BirthDate.IsZero()
}

// markDuplicates marks the rows whose name and birth date are a patient's of the clinic already, or an earlier
// row's; each of the clinic's patients found is audited as a search.
func (s *Service) markDuplicates(ctx context.Context, c Caller, rows []ImportRow) error {
	var ask []Patient
	for _, r := range rows {
		if matchable(r) {
			ask = append(ask, r.Patient)
		}
	}
	if len(ask) == 0 {
		return nil
	}
	found, err := s.repo.Matching(ctx, c.OrganizationID, ask)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	audited := map[string]bool{}
	for _, p := range found {
		seen[importKey(p)] = true
		if !audited[p.ID] {
			audited[p.ID] = true
			if err := s.record(ctx, c, auditevent.CodeSearchType, p.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
	}
	for i, r := range rows {
		if !matchable(r) {
			continue
		}
		k := importKey(r.Patient)
		rows[i].Duplicate = seen[k]
		seen[k] = true
	}
	return nil
}

// saveImport registers every row as a profile of c's clinic, each audited as a create, then audits the import with
// its count (never its rows).
func (s *Service) saveImport(ctx context.Context, c Caller, rows []ImportRow) error {
	for i := range rows {
		p, err := s.repo.Register(ctx, rows[i].Patient, c.OrganizationID, s.clock.Now())
		if err != nil {
			return err
		}
		rows[i].Patient = p
		if err := s.record(ctx, c, auditevent.CodeCreate, p.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
	}
	return s.audit.Record(ctx, auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         auditevent.ActionCreate,
		Outcome:        auditevent.OutcomeSuccess,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     "patient",
		EntityCount:    int64(len(rows)),
	})
}
