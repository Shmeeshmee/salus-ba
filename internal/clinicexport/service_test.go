package clinicexport_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

const (
	clinicA = "11111111-1111-4111-8111-111111111111"
	clinicB = "22222222-2222-4222-8222-222222222222"
	adminA  = "33333333-3333-4333-8333-333333333333"
)

var asked = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

type env struct {
	svc     *clinicexport.Service
	repo    *fakeRepo
	audit   *fakeAudit
	jobs    *fakeJobs
	store   *fakeStore
	sources *fakeSources
	clk     *clock.Fixed
}

func newEnv(t *testing.T) *env {
	t.Helper()
	kek, err := envelope.NewKEK("test", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	e := &env{
		repo: newFakeRepo(), audit: &fakeAudit{}, jobs: &fakeJobs{}, store: &fakeStore{objects: map[string][]byte{}},
		sources: &fakeSources{people: map[string]practitioner.Practitioner{}, records: map[string]patient.ClinicRecord{}}, clk: &clock.Fixed{T: asked},
	}
	e.svc = clinicexport.NewService(clinicexport.Ports{
		Repo: e.repo, Audit: e.audit, Tx: fakeTx{}, Clock: e.clk, Jobs: e.jobs, Store: e.store, KEK: kek,
		PriceList: e.sources, Roles: e.sources, Practitioners: practitioners{e.sources}, Appointments: visits{e.sources},
		Receipts: e.sources, Quotes: e.sources, Patients: patients{e.sources},
	})
	return e
}

var admin = clinicexport.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}

func TestRequest_RefusesAnyoneButTheAdmin(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]clinicexport.Caller{
		"practitioner": {AccountID: adminA, OrganizationID: clinicA},
		"patient":      {AccountID: adminA},
	} {
		e := newEnv(t)
		if _, err := e.svc.Request(context.Background(), c); !errors.Is(err, clinicexport.ErrForbidden) {
			t.Errorf("%s: err = %v, want ErrForbidden", name, err)
		}
		if len(e.jobs.builds) != 0 || len(e.audit.events) != 1 || e.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: jobs %v, audit %+v; want no job and one refusal", name, e.jobs.builds, e.audit.events)
		}
	}
}

func TestRequest_QueuesOneExportAndItsBuild(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	got, err := e.svc.Request(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != clinicexport.StatusQueued || !got.RequestedAt.Equal(asked) || !got.ExpiresAt.Equal(asked.Add(7*24*time.Hour)) {
		t.Errorf("export = %+v; want queued, asked now, expiring in 7 days", got)
	}
	if len(e.jobs.builds) != 1 || e.jobs.builds[0] != (clinicexport.BuildArgs{ExportID: got.ID, OrganizationID: clinicA}) {
		t.Errorf("jobs = %+v; want one build of the export", e.jobs.builds)
	}
	if len(e.audit.events) != 1 || e.audit.events[0].Action != auditevent.ActionCreate || e.audit.events[0].AgentAccountID != adminA {
		t.Errorf("audit = %+v; want the admin's create", e.audit.events)
	}
	again, err := e.svc.Request(context.Background(), admin)
	if err != nil || again.ID != got.ID || len(e.jobs.builds) != 1 {
		t.Errorf("second ask = %+v, %v, %d jobs; want the queued one and no second build", again, err, len(e.jobs.builds))
	}
}

func TestGet_OnlyTheClinicsAdminAndExpiredAfterTheWindow(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	x, err := e.svc.Request(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	other := clinicexport.Caller{AccountID: adminA, OrganizationID: clinicB, Admin: true}
	if _, err := e.svc.Get(context.Background(), other, x.ID); !errors.Is(err, clinicexport.ErrNotFound) {
		t.Errorf("another clinic's admin: err = %v, want ErrNotFound", err)
	}
	if _, err := e.svc.Get(context.Background(), clinicexport.Caller{AccountID: adminA, OrganizationID: clinicA}, x.ID); !errors.Is(err, clinicexport.ErrForbidden) {
		t.Errorf("practitioner: err = %v, want ErrForbidden", err)
	}
	if err := e.svc.Build(context.Background(), clinicexport.BuildArgs{ExportID: x.ID, OrganizationID: clinicA}); err != nil {
		t.Fatal(err)
	}
	if got, err := e.svc.Get(context.Background(), admin, x.ID); err != nil || got.Status != clinicexport.StatusReady {
		t.Errorf("after the build = %+v, %v; want ready", got, err)
	}
	e.clk.T = asked.Add(7 * 24 * time.Hour)
	if got, err := e.svc.Get(context.Background(), admin, x.ID); err != nil || got.Status != clinicexport.StatusExpired {
		t.Errorf("after the window = %+v, %v; want expired", got, err)
	}
	if _, _, err := e.svc.Download(context.Background(), admin, x.ID); !errors.Is(err, clinicexport.ErrNotFound) {
		t.Errorf("download after the window: err = %v, want ErrNotFound", err)
	}
}

func TestBuild_StoresTheClinicsDataEncryptedAndTheAdminDownloadsIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vat := int32(1700)
	e.sources.prices = []chargeitemdefinition.Definition{{ID: "d1", Code: "K1", Name: "Pregled, prvi", PriceMinor: 5000, Currency: "BAM",
		VATRateBP: &vat, DurationMinutes: 30, Modes: []schedule.VisitMode{schedule.ModeInPerson}, Active: true}}
	e.sources.roles = []practitionerrole.PractitionerRole{{ID: "r1", PractitionerID: "p1", SpecialtyCode: "394579002", Active: true}}
	e.sources.people["p1"] = practitioner.Practitioner{ID: "p1", GivenName: "Ana", FamilyName: "Test"}
	start := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	e.sources.appointments = []appointment.Seen{{Patient: appointment.Name{Given: "=SUM(A1)", Family: "Primjer"},
		Appointment: appointment.Appointment{ID: "a1", PatientID: "pt1", OrganizationID: clinicA, PractitionerRoleID: "r1", Start: start, End: start.Add(30 * time.Minute),
			Status: appointment.StatusFulfilled, Mode: schedule.ModeInPerson, ServiceID: "d1"}}}
	e.sources.receipts = []invoice.Invoice{{ID: "i1", Number: 7, Kind: invoice.KindReceipt, IssuedAt: start, PatientID: "pt1", AppointmentID: "a1",
		PaymentMethod: invoice.PaymentCash, TotalMinor: 5000, Currency: "BAM", FiscalStatus: invoice.FiscalFiscalized}}
	e.sources.quotes = []invoice.Quote{{ID: "q1", IssuedAt: start, ValidUntil: start.Add(30 * 24 * time.Hour), State: invoice.QuoteSent,
		PatientID: "pt1", AppointmentID: "a1", Note: "Placeholder note", TotalMinor: 12000, Currency: "BAM"}}
	e.sources.registered = []string{"pt1", "gone"} // the second deleted since the list was read
	e.sources.restricted = []string{"pt1", "pt9"}  // records the clinic restricts (D4, S7)
	e.sources.records["pt1"] = patient.ClinicRecord{Patient: patient.Patient{ID: "pt1", GivenName: "Cy", FamilyName: "Primjer",
		BirthDate: time.Date(1980, 3, 4, 0, 0, 0, 0, time.UTC), RegisteredBy: clinicA, VersionID: 1, LastUpdated: start}}

	x, err := e.svc.Request(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	e.clk.T = asked.Add(time.Minute)
	if err := e.svc.Build(context.Background(), clinicexport.BuildArgs{ExportID: x.ID, OrganizationID: clinicA}); err != nil {
		t.Fatal(err)
	}
	sealed := e.store.objects[clinicexport.ObjectKey(x.ID)]
	if len(sealed) == 0 || bytes.Contains(sealed, []byte("Primjer")) || bytes.Contains(sealed, []byte("prices.csv")) {
		t.Fatalf("stored %d bytes; want the archive encrypted", len(sealed))
	}
	checkReadAsTheAdmin(t, e.sources)
	if want := []string{patient.PurposeExport, patient.PurposeExport}; len(e.sources.purposes) != 2 || e.sources.purposes[0] != want[0] || e.sources.purposes[1] != want[1] {
		t.Errorf("records read for %v, want the export's purpose", e.sources.purposes)
	}
	if p := e.sources.periods[0]; !p.To.After(e.clk.T) || !p.From.Before(start) {
		t.Errorf("receipts' period = %+v; want every receipt", p)
	}

	before := len(e.audit.events)
	r, size, err := e.svc.Download(context.Background(), admin, x.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	plain, err := io.ReadAll(r)
	if err != nil || int64(len(plain)) != size {
		t.Fatalf("read %d bytes of %d, %v", len(plain), size, err)
	}
	if len(e.audit.events) != before+1 || e.audit.events[before].Action != auditevent.ActionRead {
		t.Errorf("audit after the download = %+v; want one read", e.audit.events[before:])
	}
	files := unzip(t, plain)
	want := map[string]string{
		"prices.csv": "id,code,name,price_minor,currency,vat_rate_bp,duration_minutes,modes,active\n" +
			"d1,K1,\"Pregled, prvi\",5000,BAM,1700,30,in-person,true\n",
		"staff.csv": "practitioner_role_id,practitioner_id,given_name,family_name,specialty_code,active\n" +
			"r1,p1,Ana,Test,394579002,true\n",
		"appointments.csv": "id,start,end,status,visit_mode,patient_id,patient_given_name,patient_family_name,practitioner_role_id,service_id\n" +
			"a1,2026-11-20T08:00:00Z,2026-11-20T08:30:00Z,fulfilled,in-person,pt1,'=SUM(A1),Primjer,r1,d1\n",
		"receipts.csv": "id,number,kind,issued_at,patient_id,company_name,company_tax_id,appointment_id,reverses_id,payment_method," +
			"total_minor,currency,fiscal_status,fiscal_id\n" +
			"i1,7," + string(invoice.KindReceipt) + ",2026-11-20T08:00:00Z,pt1,,,a1,," + string(invoice.PaymentCash) + ",5000,BAM," +
			string(invoice.FiscalFiscalized) + ",\n",
		"quotes.csv": "id,issued_at,valid_until,state,patient_id,appointment_id,total_minor,currency\n" +
			"q1,2026-11-20T08:00:00Z,2026-12-20T08:00:00Z,sent,pt1,a1,12000,BAM\n",
	}
	for name, body := range want {
		if files[name] != body {
			t.Errorf("%s =\n%s\nwant\n%s", name, files[name], body)
		}
	}
	checkPatientFile(t, files["patients/pt1.json"])
	checkManifest(t, files["manifest.txt"])
	if len(files) != len(want)+2 {
		t.Errorf("archive holds %d files, want %d (the CSVs, one patient's record and the manifest)", len(files), len(want)+2)
	}
}

// checkPatientFile checks a patient's record in the archive: a FHIR collection with the profile, the visit and the
// receipt, and no note of a quote.
func checkPatientFile(t *testing.T, body string) {
	t.Helper()
	var b struct {
		ResourceType string `json:"resourceType"`
		Type         string `json:"type"`
		Entry        []struct {
			Resource struct {
				ResourceType string `json:"resourceType"`
				ID           string `json:"id"`
			} `json:"resource"`
		} `json:"entry"`
	}
	if err := json.Unmarshal([]byte(body), &b); err != nil || b.ResourceType != "Bundle" || b.Type != "collection" {
		t.Fatalf("patients/pt1.json = %s, %v", body, err)
	}
	var got []string
	for _, e := range b.Entry {
		got = append(got, e.Resource.ResourceType+"/"+e.Resource.ID)
	}
	if want := []string{"Patient/pt1", "Appointment/a1", "Invoice/i1"}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("patients/pt1.json holds %v, want %v", got, want)
	}
	if strings.Contains(body, "Placeholder note") {
		t.Error("a patient's file holds a quote's note")
	}
}

func TestBuild_OnlyAQueuedExportAndAFailedOneIsMarked(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	x, err := e.svc.Request(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	args := clinicexport.BuildArgs{ExportID: x.ID, OrganizationID: clinicA}
	e.sources.err = errors.New("down")
	if err := e.svc.Build(context.Background(), args); err == nil {
		t.Fatal("Build with a source down: want an error, so the job retries")
	}
	if err := e.svc.Fail(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.svc.Get(context.Background(), admin, x.ID); got.Status != clinicexport.StatusFailed {
		t.Errorf("status = %s, want failed", got.Status)
	}
	if err := e.svc.Build(context.Background(), args); err != nil {
		t.Errorf("Build of a failed export = %v; want nothing done", err)
	}
	if _, _, err := e.svc.Download(context.Background(), admin, x.ID); !errors.Is(err, clinicexport.ErrNotFound) {
		t.Errorf("download of a failed export: err = %v, want ErrNotFound", err)
	}
	if next, err := e.svc.Request(context.Background(), admin); err != nil || next.ID == x.ID {
		t.Errorf("ask after a failure = %+v, %v; want a new export", next, err)
	}
}

func TestPurger_DeletesTheArchivesAndTheRowsPastTheWindow(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	x, err := e.svc.Request(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Build(context.Background(), clinicexport.BuildArgs{ExportID: x.ID, OrganizationID: clinicA}); err != nil {
		t.Fatal(err)
	}
	p := clinicexport.NewPurger(e.repo, e.store)
	if n, err := p.Purge(context.Background(), asked); err != nil || n != 0 {
		t.Errorf("purge before the request = %d, %v; want none", n, err)
	}
	if n, err := p.Purge(context.Background(), asked.Add(time.Second)); err != nil || n != 1 {
		t.Errorf("purge after = %d, %v; want 1", n, err)
	}
	if len(e.store.objects) != 0 || len(e.repo.rows) != 0 {
		t.Errorf("left %d objects and %d rows; want none", len(e.store.objects), len(e.repo.rows))
	}
}

func unzip(t *testing.T, b []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var sb strings.Builder
		if _, err := io.Copy(&sb, rc); err != nil { //nolint:gosec // a test's own small archive
			t.Fatal(err)
		}
		_ = rc.Close()
		out[f.Name] = sb.String()
	}
	return out
}

// checkReadAsTheAdmin checks every read of the build was made as the admin who asked (each audited by its service).
func checkReadAsTheAdmin(t *testing.T, f *fakeSources) {
	t.Helper()
	var who []string
	for _, c := range f.visitCallers {
		who = append(who, c.AccountID+" "+c.OrganizationID+" "+strconv.FormatBool(c.Admin))
	}
	for _, c := range f.receiptCallers {
		who = append(who, c.AccountID+" "+c.OrganizationID+" "+strconv.FormatBool(c.Admin))
	}
	for _, c := range f.patientCalls {
		who = append(who, c.AccountID+" "+c.OrganizationID+" "+strconv.FormatBool(c.Admin))
	}
	for _, c := range f.roleCallers {
		who = append(who, c.AccountID+" "+c.OrganizationID+" "+strconv.FormatBool(c.Admin))
	}
	if len(f.roleCallers) != 1 || len(f.visitCallers) == 0 || len(f.receiptCallers) == 0 || len(f.patientCalls) == 0 {
		t.Errorf("reads: roles %d, visits %d, receipts and quotes %d, patients %d", len(f.roleCallers), len(f.visitCallers), len(f.receiptCallers), len(f.patientCalls))
	}
	for _, w := range who {
		if w != adminA+" "+clinicA+" true" {
			t.Errorf("read as %q; want the asking admin", w)
		}
	}
}

// checkManifest checks the manifest of the clinic's export: one patient's record, what is not in it, the cancelled
// visits (S11) and the records the clinic restricts (S7).
func checkManifest(t *testing.T, m string) {
	t.Helper()
	if !strings.Contains(m, "Patients' records in this archive: 1\n") ||
		!strings.Contains(m, "Not in this archive: clinical documents, questionnaire answers and the patient's history.") ||
		!strings.Contains(m, "the visits a withdrawal of consent cancelled are in it too") || strings.Contains(m, "in neither") ||
		!strings.Contains(m, "Restricted records (GDPR Art. 18), by patient id:\npt1\npt9\n") {
		t.Errorf("manifest.txt =\n%s", m)
	}
}
