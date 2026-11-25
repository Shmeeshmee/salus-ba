package diagnosticreport_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

var now = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

type world struct {
	svc    *diagnosticreport.Service
	repo   *fakeRepo
	visits *fakeVisits
	audit  *fakeAudit
}

func newWorld() world {
	w := world{repo: newFakeRepo(), visits: &fakeVisits{consent: true}, audit: &fakeAudit{}}
	w.svc = diagnosticreport.NewService(diagnosticreport.Ports{
		Repo: w.repo, Visits: w.visits, Audit: w.audit, Tx: fakeTx{audit: w.audit}, Clock: clock.Fixed{T: now},
	})
	return w
}

var (
	thePractitioner = diagnosticreport.Staff{AccountID: "00000000-0000-4000-8000-0000000000e1", OrganizationID: clinic, PractitionerID: practitioner}
	theAdmin        = diagnosticreport.Staff{AccountID: "00000000-0000-4000-8000-0000000000e2", OrganizationID: clinic, Admin: true}
	theReceptionist = diagnosticreport.Staff{AccountID: "00000000-0000-4000-8000-0000000000e3", OrganizationID: clinic, Reception: true}
	otherDoctor     = diagnosticreport.Staff{AccountID: "00000000-0000-4000-8000-0000000000e4", OrganizationID: clinic, PractitionerID: "00000000-0000-4000-8000-0000000000a9"}
	thePatient      = diagnosticreport.Patient{AccountID: "00000000-0000-4000-8000-0000000000e5", PatientID: patientID}
)

func ptr(f float64) *float64 { return &f }

func blood() diagnosticreport.Draft {
	return diagnosticreport.Draft{Title: "Complete blood count", LOINC: "58410-2", Results: []diagnosticreport.Result{
		{Test: "Haemoglobin", LOINC: "718-7", Value: ptr(13.5), Unit: "g/dL", ReferenceRange: "12-16", Interpretation: diagnosticreport.InterpretationNormal},
		{Test: "Synthetic marker", ValueText: "negative"},
	}}
}

func TestAttach_StoresAPreliminaryReportAndAuditsIt(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r, err := w.svc.Attach(t.Context(), thePractitioner, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != diagnosticreport.StatusPreliminary || r.PatientID != patientID || r.OrganizationID != clinic || len(r.Results) != 2 {
		t.Fatalf("attached %+v", r)
	}
	if w.repo.org != clinic {
		t.Errorf("scope %q, want the clinic", w.repo.org)
	}
	last := w.audit.events[len(w.audit.events)-1]
	if last.EntityReference != "DiagnosticReport/"+r.ID || last.Action != auditevent.ActionCreate || last.Outcome != auditevent.OutcomeSuccess ||
		last.Purpose.Code != auditevent.CodeTREAT || last.OrganizationID != clinic {
		t.Errorf("audit %+v", last)
	}
}

func TestAttach_Refusals(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 201)
	cases := []struct {
		want    error
		draft   func(d *diagnosticreport.Draft)
		name    string
		visit   string
		staff   diagnosticreport.Staff
		consent bool
	}{
		{diagnosticreport.ErrForbidden, nil, "a receptionist", visitID, theReceptionist, true},
		{diagnosticreport.ErrNotFound, nil, "another practitioner's visit", visitID, otherDoctor, true},
		{diagnosticreport.ErrNotFound, nil, "an unknown visit", "00000000-0000-4000-8000-0000000000d9", theAdmin, true},
		{diagnosticreport.ErrNotFound, nil, "no active consent", visitID, thePractitioner, false},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Title = "  " }, "no title", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Title = long }, "a long title", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.LOINC = "abc" }, "a bad LOINC", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[0].Test = "" }, "no test", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[0].ValueText = "high" }, "both values", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[1].ValueText = "" }, "no value", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[1].Unit = "g/dL" }, "a unit without a number", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[0].Interpretation = "X" }, "an unknown flag", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) { d.Results[0].ReferenceRange = long }, "a long range", visitID, thePractitioner, true},
		{diagnosticreport.ErrInvalid, func(d *diagnosticreport.Draft) {
			d.Results = make([]diagnosticreport.Result, 101)
			for i := range d.Results {
				d.Results[i] = diagnosticreport.Result{Test: "T", ValueText: "v"}
			}
		}, "too many values", visitID, thePractitioner, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld()
			w.visits.consent = tc.consent
			d := blood()
			if tc.draft != nil {
				tc.draft(&d)
			}
			_, err := w.svc.Attach(t.Context(), tc.staff, tc.visit, d)
			if !errors.Is(err, tc.want) {
				t.Fatalf("%v, want %v", err, tc.want)
			}
			if len(w.repo.reports) != 0 {
				t.Error("a refused report was stored")
			}
			if errors.Is(tc.want, diagnosticreport.ErrInvalid) {
				return
			}
			if len(w.audit.events) != 1 || w.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
				t.Errorf("audit %+v, want one refusal", w.audit.events)
			}
		})
	}
}

func TestChange_AtItsVersion(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r, err := w.svc.Attach(t.Context(), theAdmin, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	d := blood()
	d.Title, d.Version = "Lipid panel", 1
	got, err := w.svc.Change(t.Context(), thePractitioner, r.ID, d)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Lipid panel" || got.VersionID != 2 || got.PatientID != patientID || got.AppointmentID != visitID {
		t.Fatalf("changed %+v", got)
	}
	if _, err := w.svc.Change(t.Context(), thePractitioner, r.ID, d); !errors.Is(err, diagnosticreport.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	if _, err := w.svc.Change(t.Context(), theReceptionist, r.ID, d); !errors.Is(err, diagnosticreport.ErrForbidden) {
		t.Fatalf("receptionist: %v", err)
	}
	if _, err := w.svc.Change(t.Context(), otherDoctor, r.ID, d); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("other practitioner: %v", err)
	}
	d.Version, d.Title = 2, ""
	if _, err := w.svc.Change(t.Context(), thePractitioner, r.ID, d); !errors.Is(err, diagnosticreport.ErrInvalid) {
		t.Fatalf("no title: %v", err)
	}
}

func TestRelease_ByAPractitionerOnce(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r, err := w.svc.Attach(t.Context(), theAdmin, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Release(t.Context(), theAdmin, r.ID, 1); !errors.Is(err, diagnosticreport.ErrForbidden) {
		t.Fatalf("admin: %v", err)
	}
	if _, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 7); !errors.Is(err, diagnosticreport.ErrVersionConflict) {
		t.Fatalf("stale: %v", err)
	}
	w.visits.consent = false // a revocation does not hold up the release of what was written
	got, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != diagnosticreport.StatusFinal || got.ReleasedAt == nil || !got.ReleasedAt.Equal(now) || got.VersionID != 2 {
		t.Fatalf("released %+v", got)
	}
	if _, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 2); !errors.Is(err, diagnosticreport.ErrReleased) {
		t.Fatalf("again: %v", err)
	}
	d := blood()
	d.Version = 2
	if _, err := w.svc.Change(t.Context(), thePractitioner, r.ID, d); !errors.Is(err, diagnosticreport.ErrReleased) {
		t.Fatalf("change after release: %v", err)
	}
	last := w.audit.events[len(w.audit.events)-1]
	if last.Outcome != auditevent.OutcomeMinorFailure || last.EntityReference != "DiagnosticReport/"+r.ID {
		t.Errorf("refusal audit %+v", last)
	}
}

func TestReads_StaffAndPatient(t *testing.T) {
	t.Parallel()
	w := newWorld()
	first, err := w.svc.Attach(t.Context(), theAdmin, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.svc.Attach(t.Context(), thePractitioner, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	list, err := w.svc.ForVisit(t.Context(), thePractitioner, visitID)
	if err != nil || len(list) != 2 {
		t.Fatalf("for visit: %v %d", err, len(list))
	}
	if _, err := w.svc.ForVisit(t.Context(), theReceptionist, visitID); !errors.Is(err, diagnosticreport.ErrForbidden) {
		t.Fatalf("receptionist list: %v", err)
	}
	if got, err := w.svc.Read(t.Context(), theAdmin, first.ID); err != nil || got.ID != first.ID {
		t.Fatalf("read: %v", err)
	}
	if _, err := w.svc.Read(t.Context(), otherDoctor, first.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("other practitioner: %v", err)
	}
	if _, err := w.svc.Release(t.Context(), thePractitioner, second.ID, 1); err != nil {
		t.Fatal(err)
	}
	w.repo.org = ""
	mine, err := w.svc.Mine(t.Context(), thePatient)
	if err != nil || len(mine) != 1 || mine[0].ID != second.ID {
		t.Fatalf("mine: %v %+v", err, mine)
	}
	if _, err := w.svc.MyReport(t.Context(), thePatient, first.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("an unreleased one: %v", err)
	}
	got, err := w.svc.MyReport(t.Context(), thePatient, second.ID)
	if err != nil || got.ID != second.ID {
		t.Fatalf("my report: %v", err)
	}
	last := w.audit.events[len(w.audit.events)-1]
	if last.Purpose.Code != auditevent.CodePATRQT || last.EntityReference != "DiagnosticReport/"+second.ID || last.AgentAccountID != thePatient.AccountID {
		t.Errorf("patient audit %+v", last)
	}
	if none, err := w.svc.Mine(t.Context(), diagnosticreport.Patient{AccountID: thePatient.AccountID}); err != nil || len(none) != 0 {
		t.Fatalf("no profile: %v %d", err, len(none))
	}
}

// The PDF waits for its storage (a follow-up of LB1): every PDF method still refuses.
func TestPDF_NotBuiltYet(t *testing.T) {
	t.Parallel()
	w := newWorld()
	ctx := t.Context()
	if _, err := w.svc.AttachPDF(ctx, thePractitioner, visitID, strings.NewReader("%PDF-"), 5); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("AttachPDF: %v", err)
	}
	if _, _, err := w.svc.PDF(ctx, thePractitioner, visitID); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("PDF: %v", err)
	}
	if _, _, err := w.svc.MyPDF(ctx, thePatient, visitID); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("MyPDF: %v", err)
	}
}

// The patient's export of one clinic reads, and audits, that clinic's released reports only.
func TestMineAt_NarrowsToTheClinic(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r, err := w.svc.Attach(t.Context(), thePractitioner, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 1); err != nil {
		t.Fatal(err)
	}
	other := r
	other.ID, other.OrganizationID = "00000000-0000-4000-8000-0000000000f1", "00000000-0000-4000-8000-0000000000c2"
	w.repo.reports[other.ID] = func() diagnosticreport.Report {
		x := w.repo.reports[r.ID]
		x.ID, x.OrganizationID = other.ID, other.OrganizationID
		return x
	}()
	n := len(w.audit.events)
	got, err := w.svc.MineAt(t.Context(), thePatient, clinic)
	if err != nil || len(got) != 1 || got[0].ID != r.ID {
		t.Fatalf("MineAt() = %+v, %v", got, err)
	}
	if len(w.audit.events) != n+1 || w.audit.events[n].EntityReference != "DiagnosticReport/"+r.ID {
		t.Errorf("audit %+v", w.audit.events[n:])
	}
	if all, err := w.svc.MineAt(t.Context(), thePatient, ""); err != nil || len(all) != 2 {
		t.Errorf("every clinic's: %d, %v", len(all), err)
	}
}
