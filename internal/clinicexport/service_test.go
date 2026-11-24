package clinicexport_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/invoice"
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
		sources: &fakeSources{people: map[string]practitioner.Practitioner{}}, clk: &clock.Fixed{T: asked},
	}
	e.svc = clinicexport.NewService(clinicexport.Ports{
		Repo: e.repo, Audit: e.audit, Tx: fakeTx{}, Clock: e.clk, Jobs: e.jobs, Store: e.store, KEK: kek,
		PriceList: e.sources, Roles: e.sources, Practitioners: practitioners{e.sources}, Appointments: visits{e.sources},
		Receipts: e.sources,
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
		Appointment: appointment.Appointment{ID: "a1", PatientID: "pt1", PractitionerRoleID: "r1", Start: start, End: start.Add(30 * time.Minute),
			Status: appointment.StatusFulfilled, Mode: schedule.ModeInPerson, ServiceID: "d1"}}}
	e.sources.receipts = []invoice.Invoice{{ID: "i1", Number: 7, Kind: invoice.KindReceipt, IssuedAt: start, PatientID: "pt1", AppointmentID: "a1",
		PaymentMethod: invoice.PaymentCash, TotalMinor: 5000, Currency: "BAM", FiscalStatus: invoice.FiscalFiscalized}}

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
	wantAdmin := func(account, org string, isAdmin bool) bool { return account == adminA && org == clinicA && isAdmin }
	if len(e.sources.visitCallers) != 1 || !wantAdmin(e.sources.visitCallers[0].AccountID, e.sources.visitCallers[0].OrganizationID, e.sources.visitCallers[0].Admin) ||
		len(e.sources.receiptCallers) != 1 || !wantAdmin(e.sources.receiptCallers[0].AccountID, e.sources.receiptCallers[0].OrganizationID, e.sources.receiptCallers[0].Admin) ||
		len(e.sources.roleCallers) != 1 || !wantAdmin(e.sources.roleCallers[0].AccountID, e.sources.roleCallers[0].OrganizationID, e.sources.roleCallers[0].Admin) {
		t.Errorf("read as %+v, %+v, %+v; want the asking admin's reads, each audited by its service",
			e.sources.visitCallers, e.sources.receiptCallers, e.sources.roleCallers)
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
	}
	for name, body := range want {
		if files[name] != body {
			t.Errorf("%s =\n%s\nwant\n%s", name, files[name], body)
		}
	}
	if len(files) != len(want) {
		t.Errorf("archive holds %d files, want %d", len(files), len(want))
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
