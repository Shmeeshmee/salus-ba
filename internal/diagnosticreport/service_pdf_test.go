package diagnosticreport_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
)

// labPDF is a synthetic PDF's first bytes.
var labPDF = []byte("%PDF-1.4\n% placeholder lab report\n%%EOF\n")

func pdfUpload(body []byte, version int) diagnosticreport.PDFUpload {
	return diagnosticreport.PDFUpload{Body: bytes.NewReader(body), Title: "placeholder.pdf", Size: int64(len(body)), Version: version}
}

// attached is a preliminary report on the visit, by the practitioner.
func (w world) attached(t *testing.T) diagnosticreport.Report {
	t.Helper()
	r, err := w.svc.Attach(t.Context(), thePractitioner, visitID, blood())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// The lab's PDF goes on a preliminary report at its version, by the visit's practitioner or the clinic's admin: the
// report moves to its next version and says it has a PDF; a second one replaces the first. Each upload is audited as
// an update of the report.
func TestAttachPDF_StoresThePDFAtTheReportsVersionAndReplacesIt(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r := w.attached(t)
	got, err := w.svc.AttachPDF(t.Context(), thePractitioner, r.ID, pdfUpload(labPDF, 1))
	if err != nil {
		t.Fatalf("AttachPDF() = %v", err)
	}
	if got.VersionID != 2 || !got.HasPDF || got.Status != diagnosticreport.StatusPreliminary || w.repo.reports[r.ID].VersionID != 2 {
		t.Fatalf("AttachPDF() = %+v", got)
	}
	if f := w.files.files[r.ID]; f.OrganizationID != clinic || f.PatientID != patientID || f.Title != "placeholder.pdf" {
		t.Errorf("stored %+v, want the report's clinic and patient", f)
	}
	last := w.audit.events[len(w.audit.events)-1]
	if last.EntityReference != "DiagnosticReport/"+r.ID || last.Action != auditevent.ActionUpdate || last.Outcome != auditevent.OutcomeSuccess ||
		last.Purpose.Code != auditevent.CodeTREAT {
		t.Errorf("audit %+v", last)
	}
	again, err := w.svc.AttachPDF(t.Context(), theAdmin, r.ID, pdfUpload(labPDF, 2))
	if err != nil || again.VersionID != 3 {
		t.Fatalf("the admin's replacement: %+v, %v", again, err)
	}
	if read, err := w.svc.Read(t.Context(), thePractitioner, r.ID); err != nil || !read.HasPDF {
		t.Errorf("Read() = %+v, %v; want it with its PDF", read, err)
	}
	if list, err := w.svc.ForVisit(t.Context(), theAdmin, visitID); err != nil || len(list) != 1 || !list[0].HasPDF {
		t.Errorf("ForVisit() = %+v, %v; want it with its PDF", list, err)
	}
}

// Who may not, what is not a PDF and a report that moved on or was released are refused, each audited as a refused
// update of the report, and the report keeps its version.
func TestAttachPDF_Refusals(t *testing.T) {
	t.Parallel()
	huge := pdfUpload(labPDF, 1)
	huge.Size = binary.MaxDocumentSize + 1
	none := pdfUpload(labPDF, 1)
	none.Size = -1
	short := pdfUpload(labPDF, 1)
	short.Size++
	cases := []struct {
		want    error
		name    string
		staff   diagnosticreport.Staff
		in      diagnosticreport.PDFUpload
		release bool
		moved   bool
	}{
		{diagnosticreport.ErrForbidden, "a receptionist", theReceptionist, pdfUpload(labPDF, 1), false, false},
		{diagnosticreport.ErrNotFound, "another practitioner", otherDoctor, pdfUpload(labPDF, 1), false, false},
		{diagnosticreport.ErrVersionConflict, "a stale version", thePractitioner, pdfUpload(labPDF, 7), false, false},
		{diagnosticreport.ErrVersionConflict, "a change during the upload", thePractitioner, pdfUpload(labPDF, 1), false, true},
		{diagnosticreport.ErrReleased, "a released report", thePractitioner, pdfUpload(labPDF, 2), true, false},
		{diagnosticreport.ErrInvalid, "not a PDF", thePractitioner, pdfUpload([]byte("plain text, no PDF"), 1), false, false},
		{diagnosticreport.ErrTooLarge, "too large", thePractitioner, huge, false, false},
		{diagnosticreport.ErrBadBody, "no length", thePractitioner, none, false, false},
		{diagnosticreport.ErrBadBody, "a short body", thePractitioner, short, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld()
			r := w.attached(t)
			if tc.release {
				if _, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 1); err != nil {
					t.Fatal(err)
				}
			}
			if tc.moved {
				w.files.beforeCommit = func() {
					w.repo.reports[r.ID] = func() diagnosticreport.Report { x := w.repo.reports[r.ID]; x.VersionID++; return x }()
				}
			}
			version := w.repo.reports[r.ID].VersionID
			if tc.moved {
				version++ // the other write's
			}
			if _, err := w.svc.AttachPDF(t.Context(), tc.staff, r.ID, tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("AttachPDF() = %v, want %v", err, tc.want)
			}
			if v := w.repo.reports[r.ID].VersionID; v != version {
				t.Errorf("version %d, want %d", v, version)
			}
			if _, ok := w.files.files[r.ID]; ok {
				t.Error("a PDF was stored")
			}
			last := w.audit.events[len(w.audit.events)-1]
			if last.Outcome != auditevent.OutcomeMinorFailure || last.Action != auditevent.ActionUpdate || last.AgentAccountID != tc.staff.AccountID {
				t.Errorf("refusal audit %+v", last)
			}
		})
	}
}

// Staff who may see the report open its PDF once it is clean; before, or without one, it is not found. A receptionist
// is refused, another practitioner does not find it; every read is audited.
func TestPDF_TheStaffOpenACleanPDFOnly(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r := w.attached(t)
	if _, err := w.svc.PDF(t.Context(), theAdmin, r.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("no PDF yet: %v", err)
	}
	if _, err := w.svc.AttachPDF(t.Context(), thePractitioner, r.ID, pdfUpload(labPDF, 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.PDF(t.Context(), theAdmin, r.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("while scanned: %v", err)
	}
	w.files.scanned(r.ID, binary.StatusClean)
	for _, staff := range []diagnosticreport.Staff{theAdmin, thePractitioner} {
		f, err := w.svc.PDF(t.Context(), staff, r.ID)
		if err != nil {
			t.Fatalf("PDF() = %v", err)
		}
		if got := content(t, w, f); !bytes.Equal(got, labPDF) {
			t.Errorf("content %q", got)
		}
		last := w.audit.events[len(w.audit.events)-1]
		if last.EntityReference != "DiagnosticReport/"+r.ID || last.Action != auditevent.ActionRead || last.Outcome != auditevent.OutcomeSuccess ||
			last.AgentAccountID != staff.AccountID {
			t.Errorf("audit %+v", last)
		}
	}
	for want, staff := range map[error]diagnosticreport.Staff{diagnosticreport.ErrForbidden: theReceptionist, diagnosticreport.ErrNotFound: otherDoctor} {
		if _, err := w.svc.PDF(t.Context(), staff, r.ID); !errors.Is(err, want) {
			t.Errorf("PDF() as %s = %v, want %v", staff.AccountID, err, want)
		}
		if last := w.audit.events[len(w.audit.events)-1]; last.Outcome != auditevent.OutcomeMinorFailure || last.AgentAccountID != staff.AccountID {
			t.Errorf("refusal audit %+v", last)
		}
	}
	w.files.scanned(r.ID, binary.StatusRejected)
	if _, err := w.svc.PDF(t.Context(), theAdmin, r.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Errorf("rejected: %v", err)
	}
}

// The patient opens the PDF of their own report only once it is released (and clean); a preliminary one is not found
// for them by any route, audited.
func TestMyPDF_ThePatientsOwnOnceReleased(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r := w.attached(t)
	if _, err := w.svc.AttachPDF(t.Context(), thePractitioner, r.ID, pdfUpload(labPDF, 1)); err != nil {
		t.Fatal(err)
	}
	w.files.scanned(r.ID, binary.StatusClean)
	w.repo.org = ""
	if _, err := w.svc.MyPDF(t.Context(), thePatient, r.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
		t.Fatalf("preliminary: %v", err)
	}
	if last := w.audit.events[len(w.audit.events)-1]; last.Outcome != auditevent.OutcomeMinorFailure || last.Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("refusal audit %+v", last)
	}
	if mine, err := w.svc.Mine(t.Context(), thePatient); err != nil || len(mine) != 0 {
		t.Fatalf("Mine() before the release = %+v, %v", mine, err)
	}
	if _, err := w.svc.Release(t.Context(), thePractitioner, r.ID, 2); err != nil {
		t.Fatal(err)
	}
	w.repo.org = ""
	f, err := w.svc.MyPDF(t.Context(), thePatient, r.ID)
	if err != nil {
		t.Fatalf("MyPDF() = %v", err)
	}
	if got := content(t, w, f); !bytes.Equal(got, labPDF) {
		t.Errorf("content %q", got)
	}
	last := w.audit.events[len(w.audit.events)-1]
	if last.EntityReference != "DiagnosticReport/"+r.ID || last.Purpose.Code != auditevent.CodePATRQT || last.AgentAccountID != thePatient.AccountID ||
		last.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit %+v", last)
	}
	if mine, err := w.svc.MyReport(t.Context(), thePatient, r.ID); err != nil || !mine.HasPDF {
		t.Errorf("MyReport() = %+v, %v; want it with its PDF", mine, err)
	}
	stranger := diagnosticreport.Patient{AccountID: "00000000-0000-4000-8000-0000000000e9", PatientID: "00000000-0000-4000-8000-0000000000b9"}
	for _, p := range []diagnosticreport.Patient{stranger, {AccountID: thePatient.AccountID}} {
		if _, err := w.svc.MyPDF(t.Context(), p, r.ID); !errors.Is(err, diagnosticreport.ErrNotFound) {
			t.Errorf("MyPDF() as %+v = %v, want ErrNotFound", p, err)
		}
	}
}

// Only the visit's practitioner releases a report: the admin and another practitioner are refused, audited.
func TestRelease_OnlyTheVisitsPractitioner(t *testing.T) {
	t.Parallel()
	w := newWorld()
	r := w.attached(t)
	for want, staff := range map[error]diagnosticreport.Staff{diagnosticreport.ErrForbidden: theAdmin, diagnosticreport.ErrNotFound: otherDoctor} {
		if _, err := w.svc.Release(t.Context(), staff, r.ID, 1); !errors.Is(err, want) {
			t.Errorf("Release() as %s = %v, want %v", staff.AccountID, err, want)
		}
		if last := w.audit.events[len(w.audit.events)-1]; last.Outcome != auditevent.OutcomeMinorFailure || last.AgentAccountID != staff.AccountID {
			t.Errorf("refusal audit %+v", last)
		}
	}
	if w.repo.reports[r.ID].Status != diagnosticreport.StatusPreliminary {
		t.Error("released")
	}
}

func content(t *testing.T, w world, f binary.File) []byte {
	t.Helper()
	r, err := w.svc.Content(t.Context(), f, 0, f.Size-1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
