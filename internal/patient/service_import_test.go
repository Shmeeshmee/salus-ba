package patient_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The names in these files are synthetic.

func csv(lines ...string) []byte { return []byte(strings.Join(lines, "\n")) }

// importRepository is the fake repository that keeps what Register stores.
type importRepository struct {
	*fakeRepository
	stored []patient.Patient
}

func newImportRepository() *importRepository {
	return &importRepository{fakeRepository: newFakeRepository()}
}

func (r *importRepository) Register(ctx context.Context, p patient.Patient, org string, at time.Time) (patient.Patient, error) {
	p, err := r.fakeRepository.Register(ctx, p, org, at)
	p.ID = "e0000000-0000-4000-8000-" + strings.Repeat("0", 11) + strconv.Itoa(len(r.stored)+1)
	if err == nil {
		r.stored = append(r.stored, p)
	}
	return p, err
}

func TestImport_PreviewsEveryRowWithItsErrorsAndLikelyDuplicates(t *testing.T) {
	t.Parallel()
	repo, audit := newImportRepository(), &fakeAuditor{}
	repo.matching = []patient.Patient{{ID: patientA, GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1)}}
	rows, err := newImportService(repo, audit, &fakeConsents{}).Import(t.Context(), admin, csv(
		"\ufeffIme;Prezime;Datum rođenja;Telefon",
		" ada ; EXAMPLE ;01.01.1990.;061 234-567",
		"Bo;Sample;1991-02-03;",
		";Sample;3.2.1991;+38761",
		"Cy;Sample;31.02.1991;00387 61 000 111",
		"",
		"Bo;Sample;03/02/1991;",
		"Di;Sample;33970;+387 (61) 000/222",
		"Ed;Sample;1985;", // a year alone is no date
	), false)
	if err != nil {
		t.Fatal(err)
	}
	want := []patient.ImportRow{
		{Line: 2, Duplicate: true, Patient: patient.Patient{GivenName: "ada", FamilyName: "EXAMPLE", BirthDate: date(1990, 1, 1), Phone: "+38761234567"}},
		{Line: 3, Patient: patient.Patient{GivenName: "Bo", FamilyName: "Sample", BirthDate: date(1991, 2, 3)}},
		{Line: 4, Errors: []string{"given_name_invalid", "phone_invalid"}, Patient: patient.Patient{FamilyName: "Sample", BirthDate: date(1991, 2, 3), Phone: "+38761"}},
		{Line: 5, Errors: []string{"birth_date_invalid"}, Patient: patient.Patient{GivenName: "Cy", FamilyName: "Sample", Phone: "+38761000111"}},
		{Line: 7, Duplicate: true, Patient: patient.Patient{GivenName: "Bo", FamilyName: "Sample", BirthDate: date(1991, 2, 3)}},
		{Line: 8, Patient: patient.Patient{GivenName: "Di", FamilyName: "Sample", BirthDate: date(1993, 1, 1), Phone: "+38761000222"}},
		{Line: 9, Errors: []string{"birth_date_invalid"}, Patient: patient.Patient{GivenName: "Ed", FamilyName: "Sample"}},
	}
	if !slices.EqualFunc(rows, want, sameImportRow) {
		t.Errorf("Import() =\n%+v,\nwant\n%+v", rows, want)
	}
	if want := []string{"SetClinic", "Matching"}; !slices.Equal(repo.calls, want) || len(repo.stored) != 0 {
		t.Errorf("calls %v, stored %d; want %v and nothing stored", repo.calls, len(repo.stored), want)
	}
	if len(repo.asked) != 4 { // the rows with a valid name and birth date
		t.Errorf("Matching asked for %d people, want 4", len(repo.asked))
	}
	// The clinic's patient the preview found is a read, audited; nothing else is.
	checkEvents(t, audit, searchEvent(admin, patientA, auditevent.OutcomeSuccess))
}

func sameImportRow(a, b patient.ImportRow) bool {
	return a.Line == b.Line && a.Duplicate == b.Duplicate && slices.Equal(a.Errors, b.Errors) && a.Patient.GivenName == b.Patient.GivenName &&
		a.Patient.FamilyName == b.Patient.FamilyName && a.Patient.BirthDate.Equal(b.Patient.BirthDate) && a.Patient.Phone == b.Patient.Phone
}

func countEvent(n int64) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         auditevent.ActionCreate,
		AgentAccountID: staffA, OrganizationID: clinicA, EntityName: "patient", EntityCount: n,
	}
}

func newImportService(repo *importRepository, audit *fakeAuditor, consents *fakeConsents) *patient.Service {
	return patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, consents, &fakeAppointments{})
}

func TestImport_SavesEveryRowAsTheClinicsWithoutAConsentAuditedWithTheCount(t *testing.T) {
	t.Parallel()
	repo, audit, consents := newImportRepository(), &fakeAuditor{}, &fakeConsents{}
	repo.matching = []patient.Patient{{ID: patientA, GivenName: "Ada", FamilyName: "Sample", BirthDate: date(1990, 1, 1)}}
	rows, err := newImportService(repo, audit, consents).Import(t.Context(), admin,
		csv("Given name,Family name,Birth date,Phone", "Ada,Sample,1990-01-01,+38761000111", "Bo,Sample,1991-02-03,"), true)
	if err != nil || len(rows) != 2 {
		t.Fatalf("Import(save) = %+v, %v", rows, err)
	}
	if want := []string{"SetClinic", "Matching", "Register", "Register"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls %v, want %v, all in the transaction", repo.calls, want)
	}
	for i, p := range repo.stored {
		if p.RegisteredBy != clinicA || p.LastUpdated != now || rows[i].Patient.ID != p.ID {
			t.Errorf("stored %+v, row %+v: want the clinic's profile, its id on the row", p, rows[i])
		}
	}
	if !rows[0].Duplicate || rows[0].Errors != nil {
		t.Errorf("row 1 = %+v, want a likely duplicate saved all the same", rows[0])
	}
	if len(consents.inPerson) != 0 {
		t.Errorf("recorded consents %v, want none (V5: the desk records it at the first booking)", consents.inPerson)
	}
	checkEvents(t, audit, searchEvent(admin, patientA, auditevent.OutcomeSuccess),
		event(admin, auditevent.CodeCreate, repo.stored[0].ID, auditevent.OutcomeSuccess),
		event(admin, auditevent.CodeCreate, repo.stored[1].ID, auditevent.OutcomeSuccess), countEvent(2))
}

func TestImport_SavesNothingWhenARowHasAnError(t *testing.T) {
	t.Parallel()
	repo, audit := newImportRepository(), &fakeAuditor{}
	rows, err := newImportService(repo, audit, &fakeConsents{}).Import(t.Context(), admin,
		csv("given name;family name;birth date", "Ada;Sample;1990-01-01", "Bo;;1991-02-03"), true)
	if err != nil || len(rows) != 2 || rows[1].Errors[0] != "family_name_invalid" {
		t.Fatalf("Import(save) = %+v, %v", rows, err)
	}
	if len(repo.stored) != 0 || len(audit.events) != 0 {
		t.Errorf("stored %d, audited %+v; want nothing", len(repo.stored), audit.events)
	}
}

func TestImport_RefusesAFileItCannotRead(t *testing.T) {
	t.Parallel()
	many := []string{"given name,family name,birth date"}
	for i := range 5001 {
		many = append(many, "Name"+strconv.Itoa(i)+",Sample,1990-01-01")
	}
	for name, data := range map[string][]byte{
		"empty":             nil,
		"only a header":     csv("given name,family name,birth date"),
		"no birth date":     csv("given name,family name,phone", "Ada,Sample,+38761000111"),
		"no family name":    csv("ime,datum rođenja", "Ada,1990-01-01"),
		"a repeated column": csv("ime,prezime,datum rođenja,first name", "Ada,Sample,1990-01-01,Ada"),
		"not UTF-8 or 1250": []byte("ime;prezime;datum rođenja\nAda\x81;Sample;1990-01-01\n"),
		"broken XLSX":       []byte("PK\x03\x04 not really"),
		"over 5 000 rows":   csv(many...),
	} {
		if _, err := newImportService(newImportRepository(), &fakeAuditor{}, &fakeConsents{}).Import(t.Context(), admin, data, false); !errors.Is(err, patient.ErrInvalidImport) {
			t.Errorf("%s: Import() = %v, want ErrInvalidImport", name, err)
		}
	}
	rows, err := newImportService(newImportRepository(), &fakeAuditor{}, &fakeConsents{}).Import(t.Context(), admin, csv(many[:5001]...), false)
	if err != nil || len(rows) != 5000 {
		t.Errorf("Import(5 000 rows) = %d rows, %v", len(rows), err)
	}
}

func TestImport_RefusesAnyoneButTheClinicsAdminAudited(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]patient.Caller{"receptionist": reception, "practitioner": practitioner, "patient": me} {
		repo, audit := newImportRepository(), &fakeAuditor{}
		if _, err := newImportService(repo, audit, &fakeConsents{}).Import(t.Context(), c, csv("given name,family name,birth date", "Ada,Sample,1990-01-01"), true); !errors.Is(err, patient.ErrForbidden) {
			t.Errorf("%s: Import() = %v, want ErrForbidden", name, err)
		}
		if len(repo.calls) != 0 {
			t.Errorf("%s: calls %v, want none", name, repo.calls)
		}
		checkEvents(t, audit, event(c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
	}
}

func TestImport_ReturnsTheStoresError(t *testing.T) {
	t.Parallel()
	repo := newImportRepository()
	repo.err = errors.New("boom")
	if _, err := newImportService(repo, &fakeAuditor{}, &fakeConsents{}).Import(t.Context(), admin, csv("given name,family name,birth date", "Ada,Sample,1990-01-01"), true); !errors.Is(err, repo.err) {
		t.Errorf("Import() = %v, want the store's error", err)
	}
}
