package chargeitemdefinition_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// The service names in these files are placeholders.

func csvFile(lines ...string) chargeitemdefinition.ImportFile {
	return chargeitemdefinition.ImportFile{Name: "services.csv", Data: []byte(strings.Join(lines, "\n"))}
}

func TestImport_PreviewsEveryRowWithoutStoringIt(t *testing.T) {
	repo, audit := &fakeRepository{}, &fakeAuditor{}
	f := csvFile(
		"\ufeffNaziv;Šifra;Cijena;Trajanje;PDV;Način",
		"Placeholder exam;EX-1;25,50;30;17;in-person video",
		"",
		"Placeholder follow-up;;10;;;",
	)
	rows, err := newService(repo, audit).Import(t.Context(), admin, f, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []chargeitemdefinition.ImportRow{
		{Line: 2, Service: chargeitemdefinition.Definition{
			Name: "Placeholder exam", Code: "EX-1", PriceMinor: 2550, DurationMinutes: 30, VATRateBP: ptr(1700),
			Currency: "BAM", OrganizationID: clinicA, Active: true, Modes: []schedule.VisitMode{schedule.ModeInPerson, schedule.ModeVideo},
		}},
		{Line: 4, Service: chargeitemdefinition.Definition{
			Name: "Placeholder follow-up", PriceMinor: 1000, Currency: "BAM", OrganizationID: clinicA, Active: true,
			Modes: []schedule.VisitMode{schedule.ModeInPerson},
		}},
	}
	if len(rows) != len(want) {
		t.Fatalf("Import() = %+v, want %+v", rows, want)
	}
	for i := range want {
		if !sameRow(rows[i], want[i]) {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
	if len(repo.rows) != 0 || len(audit.events) != 0 || repo.outside {
		t.Errorf("a preview stored %d services, audited %d events, outside a transaction %v", len(repo.rows), len(audit.events), repo.outside)
	}
}

func sameRow(a, b chargeitemdefinition.ImportRow) bool {
	vat := func(p *int32) int32 {
		if p == nil {
			return -1
		}
		return *p
	}
	x, y := a.Service, b.Service
	return a.Line == b.Line && slices.Equal(a.Errors, b.Errors) && x.Name == y.Name && x.Code == y.Code &&
		x.PriceMinor == y.PriceMinor && x.DurationMinutes == y.DurationMinutes && vat(x.VATRateBP) == vat(y.VATRateBP) &&
		x.Currency == y.Currency && x.OrganizationID == y.OrganizationID && x.Active == y.Active && slices.Equal(x.Modes, y.Modes)
}

func TestImport_NamesWhatIsWrongWithEachRow(t *testing.T) {
	for row, want := range map[string][]string{
		";;10;;;":                                 {"name_missing"},
		strings.Repeat("x", 201) + ";;10;;;":      {"name_invalid"},
		"Exam;" + strings.Repeat("c", 41) + ";10": {"code_invalid"},
		"Exam;;;;;":                               {"price_invalid"},
		"Exam;;-1;;;":                             {"price_invalid"},
		"Exam;;1,234;;;":                          {"price_invalid"},
		"Exam;;1.000.000;;;":                      {"price_invalid"},
		"Exam;;99999999;;;":                       {"price_invalid"},
		"Exam;;10;4;;":                            {"duration_invalid"},
		"Exam;;10;thirty;;":                       {"duration_invalid"},
		"Exam;;10;;101;":                          {"vat_invalid"},
		"Exam;;10;;x;":                            {"vat_invalid"},
		"Exam;;10;;;home visit":                   {"mode_invalid"},
		";;x;1;-5;fax":                            {"name_missing", "price_invalid", "duration_invalid", "vat_invalid", "mode_invalid"},
		"Exam;;10 KM;30;17%;Video, in-person":     nil,
		"Exam;;10;;;уживо/видео":                  nil,
		"Exam;;10;;;Uživo":                        nil,
	} {
		rows, err := newService(&fakeRepository{}, &fakeAuditor{}).Import(t.Context(), admin, csvFile("name;code;price;duration;vat;modes", row), false)
		if err != nil || len(rows) != 1 || !slices.Equal(rows[0].Errors, want) {
			t.Errorf("Import(%q) = %+v, %v; want errors %q", row, rows, err, want)
		}
	}
}

func TestImport_MarksADuplicateOfTheFileOrOfThePriceList(t *testing.T) {
	existing := exam()
	existing.OrganizationID, existing.ID = clinicA, "d0000000-0000-4000-8000-000000000099"
	repo := &fakeRepository{rows: []chargeitemdefinition.Definition{existing}}
	rows, err := newService(repo, &fakeAuditor{}).Import(t.Context(), admin, csvFile(
		"name,code,price",
		"Placeholder EXAM,,10",
		"Another,EX-1,10",
		"Third,T-1,10",
		"third,,10",
		"Fourth,t-1,10",
	), false)
	if err != nil {
		t.Fatal(err)
	}
	var got [][]string
	for _, r := range rows {
		got = append(got, r.Errors)
	}
	want := [][]string{{"duplicate"}, {"duplicate"}, nil, {"duplicate"}, {"duplicate"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

func TestImport_SavesAllRowsAuditedOrNone(t *testing.T) {
	repo, audit := &fakeRepository{}, &fakeAuditor{}
	svc := newService(repo, audit)
	bad := csvFile("name,price", "First,10", ",10")
	rows, err := svc.Import(t.Context(), admin, bad, true)
	if err != nil || len(rows) != 2 || len(repo.rows) != 0 || len(audit.events) != 0 {
		t.Fatalf("Import(an error, save) = %+v, %v; stored %d, audited %d; want nothing stored", rows, err, len(repo.rows), len(audit.events))
	}
	rows, err = svc.Import(t.Context(), admin, csvFile("name,price", "First,10", "Second,20"), true)
	if err != nil || len(rows) != 2 || len(repo.rows) != 2 || repo.outside {
		t.Fatalf("Import(save) = %+v, %v; stored %d", rows, err, len(repo.rows))
	}
	if rows[0].Service.ID == "" || rows[1].Service.ID != repo.rows[1].ID || !repo.writtenAt.Equal(now) {
		t.Errorf("saved rows = %+v, want their ids", rows)
	}
	for i, e := range audit.events {
		if e.Subtype.Code != auditevent.CodeCreate || e.Outcome != auditevent.OutcomeSuccess || e.AgentAccountID != adminA ||
			e.EntityReference != "ChargeItemDefinition/"+repo.rows[i].ID {
			t.Errorf("audit %d = %+v", i, e)
		}
	}
	if len(audit.events) != 2 {
		t.Errorf("audited %d events, want 2", len(audit.events))
	}
}

// xlsxFile is a workbook of one sheet with inline strings.
func xlsxFile(t *testing.T, rows ...[]string) chargeitemdefinition.ImportFile {
	t.Helper()
	var sheet strings.Builder
	for i, r := range rows {
		sheet.WriteString(`<row r="` + strconv.Itoa(i+1) + `">`)
		for _, v := range r {
			sheet.WriteString(`<c t="inlineStr"><is><t>` + v + `</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData>` + sheet.String() + `</sheetData></worksheet>`,
	} {
		w, _ := z.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return chargeitemdefinition.ImportFile{Name: "services.xlsx", Data: buf.Bytes()}
}

func TestImport_ReadsAnXLSXFile(t *testing.T) {
	rows, err := newService(&fakeRepository{}, &fakeAuditor{}).Import(t.Context(), admin,
		xlsxFile(t, []string{"Price", "Name"}, []string{"12.5", "Placeholder exam"}), false)
	if err != nil || len(rows) != 1 || rows[0].Line != 2 || rows[0].Service.Name != "Placeholder exam" ||
		rows[0].Service.PriceMinor != 1250 || rows[0].Errors != nil {
		t.Errorf("Import(xlsx) = %+v, %v", rows, err)
	}
}

// windows1250 encodes s the way Excel in Bosnia and Herzegovina saves a CSV, for the letters these tests use.
func windows1250(t *testing.T, s string) []byte {
	t.Helper()
	table := map[rune]byte{
		'Š': 0x8A, 'š': 0x9A, 'Ž': 0x8E, 'ž': 0x9E, 'Č': 0xC8, 'č': 0xE8, 'Ć': 0xC6, 'ć': 0xE6, 'Đ': 0xD0, 'đ': 0xF0,
		'–': 0x96,
	}
	var out []byte
	for _, r := range s {
		switch b, ok := table[r]; {
		case ok:
			out = append(out, b)
		case r < 0x80:
			out = append(out, byte(r))
		default:
			t.Fatalf("windows1250: no byte for %q", r)
		}
	}
	return out
}

func TestImport_ReadsAWindows1250CSV(t *testing.T) {
	names := []string{"Pregled kod ljekara – šđčćž", "Ultrazvuk srca – ŠĐČĆŽ", "Kontrolni pregled"}
	f := chargeitemdefinition.ImportFile{Name: "services.csv", Data: windows1250(t,
		"Naziv;Šifra;Cijena\n"+names[0]+";Č-1;25\n"+names[1]+";Ž-2;40\n"+names[2]+";;10\n")}
	rows, err := newService(&fakeRepository{}, &fakeAuditor{}).Import(t.Context(), admin, f, false)
	if err != nil || len(rows) != len(names) {
		t.Fatalf("Import(cp1250) = %+v, %v", rows, err)
	}
	for i, want := range names {
		if rows[i].Service.Name != want || rows[i].Errors != nil {
			t.Errorf("row %d = %q %v, want %q", i, rows[i].Service.Name, rows[i].Errors, want)
		}
	}
	if rows[0].Service.Code != "Č-1" || rows[1].Service.Code != "Ž-2" {
		t.Errorf("codes = %q, %q, want Č-1, Ž-2", rows[0].Service.Code, rows[1].Service.Code)
	}
}

func TestImport_RefusesAFileItCannotRead(t *testing.T) {
	many := []string{"name,price"}
	for i := range 1001 {
		many = append(many, "Service "+strconv.Itoa(i)+",10")
	}
	for name, f := range map[string]chargeitemdefinition.ImportFile{
		"empty":             csvFile(),
		"no name column":    csvFile("title,price", "Exam,10"),
		"no price":          csvFile("name", "Exam"),
		"only a header":     csvFile("name,price"),
		"not UTF-8 or 1250": {Name: "services.csv", Data: []byte("name;price\nPregled \x81;10\n")},
		"broken CSV":        csvFile("name,price", `"Exam,10`),
		"broken XLSX":       {Name: "services.xlsx", Data: []byte("PK\x03\x04 not really")},
		"too many rows":     csvFile(many...),
		"a repeated column": csvFile("name,price,name", "Exam,10,Other"),
	} {
		if _, err := newService(&fakeRepository{}, &fakeAuditor{}).Import(t.Context(), admin, f, false); !errors.Is(err, chargeitemdefinition.ErrInvalidImport) {
			t.Errorf("%s: Import() = %v, want ErrInvalidImport", name, err)
		}
	}
}

func TestImport_RefusesAnyoneButTheClinicsAdminAudited(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		caller chargeitemdefinition.Caller
	}{
		"other staff": {chargeitemdefinition.ErrForbidden, staff},
		"no clinic":   {chargeitemdefinition.ErrNotFound, noClinic},
	} {
		repo, audit := &fakeRepository{}, &fakeAuditor{}
		_, err := newService(repo, audit).Import(t.Context(), tc.caller, csvFile("name,price", "Exam,10"), true)
		if !errors.Is(err, tc.err) || len(repo.rows) != 0 || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: Import() = %v, stored %d, audit %+v", name, err, len(repo.rows), audit.events)
		}
	}
}

func TestImport_ReturnsTheStoresError(t *testing.T) {
	repo := &fakeRepository{err: errBroken}
	if _, err := newService(repo, &fakeAuditor{}).Import(t.Context(), admin, csvFile("name,price", "Exam,10"), false); !errors.Is(err, errBroken) {
		t.Errorf("Import() = %v, want the store's error", err)
	}
}
