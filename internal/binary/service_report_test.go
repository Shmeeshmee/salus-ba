package binary_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clamd"
)

// reportR is a lab result of patient A in clinic A (docs/clinic-cms/WAVE6.md U5).
const reportR = "d1000000-0000-4000-8000-0000000000d2"

// reportUpload is the lab's PDF for report R.
func reportUpload(body []byte) binary.ReportUpload {
	return binary.ReportUpload{
		Body: bytes.NewReader(body), ReportID: reportR, OrganizationID: clinicA, PatientID: patientA, Title: title, Size: int64(len(body)),
	}
}

// enterReport is diagnosticreport's part of an upload's transaction: the clinic's scope set, and the files it saw.
type enterReport struct {
	err   error
	repo  *fakeRepository
	files []binary.Binary
}

func (e *enterReport) commit(_ context.Context, b binary.Binary) error {
	if e.err != nil {
		return e.err
	}
	e.repo.clinic = clinicA
	e.files = append(e.files, b)
	return nil
}

// The lab's PDF (U5): a PDF only, from its bytes, sealed into the store outside any transaction; then in one
// transaction the report's own checks and audit (commit), the row quarantined under the report and its scan job,
// which names the report. binary audits nothing: the report does.
func TestUploadReport_StoresThePDFAndEnqueuesItsScanUnderTheReport(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	e := &enterReport{repo: f.repo}
	body := pdf()
	got, err := f.svc.UploadReport(t.Context(), reportUpload(body), e.commit)
	if err != nil {
		t.Fatalf("UploadReport() = %v", err)
	}
	if got.DiagnosticReportID != reportR || got.OrganizationID != clinicA || got.PatientID != patientA || got.ResponseID != "" ||
		got.ConversationID != "" || got.ContentType != "application/pdf" || got.Status != binary.StatusQuarantined || got.Title != title {
		t.Errorf("UploadReport() = %+v", got)
	}
	if len(e.files) != 1 || e.files[0].ID != got.ID {
		t.Errorf("commit saw %v, want the file once", e.files)
	}
	if want := []binary.ScanArgs{{BinaryID: got.ID, DiagnosticReportID: reportR, OrganizationID: clinicA}}; !slices.Equal(f.jobs.scans, want) {
		t.Errorf("scans = %v, want %v", f.jobs.scans, want)
	}
	if len(f.audit.events) != 0 || f.holder.uploadTargets != 0 {
		t.Errorf("binary audited %d events or asked the questionnaire holder: the report decides and audits", len(f.audit.events))
	}
	if !bytes.Equal(f.plaintext(t, got), body) {
		t.Error("the stored object does not open to the upload")
	}
}

// A second PDF before the release replaces the first: its row goes in the same transaction, its object after it.
func TestUploadReport_ReplacesTheReportsPDF(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	e := &enterReport{repo: f.repo}
	first, err := f.svc.UploadReport(t.Context(), reportUpload(pdf()), e.commit)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.UploadReport(t.Context(), reportUpload(pdf()), e.commit)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.repo.files[first.ID]; ok {
		t.Error("the first PDF's row is still there")
	}
	if _, ok := f.repo.files[second.ID]; !ok {
		t.Error("the second PDF's row is missing")
	}
	if !slices.Contains(f.store.deleted, first.ID) || slices.Contains(f.store.deleted, second.ID) {
		t.Errorf("deleted objects %v, want the first only", f.store.deleted)
	}
}

// A refusal of the bytes (no length, past 20 MB, anything but a PDF, a short body) leaves nothing and never reaches
// the report's commit; a refusal of the commit leaves no row and no object, and the PDF already there stays.
func TestUploadReport_RefusesAnythingButAPDFAndLeavesNothing(t *testing.T) {
	t.Parallel()
	errReport := errors.New("the report refused")
	short := reportUpload(pdf())
	short.Size += 10
	none := reportUpload(pdf())
	none.Size = -1
	huge := reportUpload(pdf())
	huge.Size = binary.MaxDocumentSize + 1
	for _, tt := range []struct {
		name      string
		commitErr error
		want      error
		in        binary.ReportUpload
	}{
		{name: "no length", in: none, want: binary.ErrLengthRequired},
		{name: "too large", in: huge, want: binary.ErrTooLarge},
		{name: "a photo", in: reportUpload(pngBytes(t)), want: binary.ErrUnsupportedType},
		{name: "text", in: reportUpload([]byte("plain text, no file type")), want: binary.ErrUnsupportedType},
		{name: "a short body", in: short, want: binary.ErrShortBody},
		{name: "the report's refusal", in: reportUpload(pdf()), commitErr: errReport, want: errReport},
	} {
		f := newFixture(t)
		kept, err := f.svc.UploadReport(t.Context(), reportUpload(pdf()), (&enterReport{repo: f.repo}).commit)
		if err != nil {
			t.Fatal(err)
		}
		puts := len(f.store.puts)
		e := &enterReport{repo: f.repo, err: tt.commitErr}
		if _, err := f.svc.UploadReport(t.Context(), tt.in, e.commit); !errors.Is(err, tt.want) {
			t.Errorf("%s: UploadReport() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.repo.files) != 1 || f.repo.files[kept.ID].ID != kept.ID || len(f.jobs.scans) != 1 {
			t.Errorf("%s: left %d rows and %d scans, want the PDF there before", tt.name, len(f.repo.files), len(f.jobs.scans))
		}
		if tt.commitErr == nil && len(e.files) != 0 {
			t.Errorf("%s: the commit ran", tt.name)
		}
		for _, key := range f.store.puts[puts:] {
			if !slices.Contains(f.store.deleted, key) {
				t.Errorf("%s: the object %s was left in the store", tt.name, key)
			}
		}
		if slices.Contains(f.store.deleted, kept.ID) {
			t.Errorf("%s: the PDF there before was deleted", tt.name)
		}
	}
}

// reportFile uploads a PDF to report R through the service, with status.
func (f *fixture) reportFile(t *testing.T, status binary.Status) binary.Binary {
	t.Helper()
	b, err := f.svc.UploadReport(t.Context(), reportUpload(pdf()), (&enterReport{repo: f.repo}).commit)
	if err != nil {
		t.Fatal(err)
	}
	b.Status = status
	if status != binary.StatusQuarantined {
		b.ScannedAt = now
	}
	f.repo.files[b.ID] = b
	return b
}

// inReport runs fn in a transaction scoped as diagnosticreport scopes one: the clinic's.
func (f *fixture) inReport(t *testing.T, fn func(ctx context.Context) error) error {
	t.Helper()
	return fakeTransactor{f: f}.WithTx(t.Context(), func(ctx context.Context) error {
		f.repo.clinic = clinicA
		return fn(ctx)
	})
}

// The report lists its files that are not rejected, and opens a clean one: ErrScanning while it is scanned,
// ErrNotFound once rejected or for a file of no report.
func TestReportFiles_ListsAndOpensTheReportsPDF(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	scanning := f.reportFile(t, binary.StatusQuarantined)
	err := f.inReport(t, func(ctx context.Context) error {
		got, err := f.svc.ReportFiles(ctx, []string{reportR, unknownID})
		if err != nil || len(got) != 1 || got[0].ID != scanning.ID {
			t.Errorf("ReportFiles() = %+v, %v; want the PDF", got, err)
		}
		if _, err := f.svc.OpenReport(ctx, got[0]); !errors.Is(err, binary.ErrScanning) {
			t.Errorf("OpenReport() of a quarantined PDF = %v, want ErrScanning", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	clean := f.reportFile(t, binary.StatusClean)
	err = f.inReport(t, func(ctx context.Context) error {
		file, err := f.svc.OpenReport(ctx, clean)
		if err != nil {
			return err
		}
		r, err := f.svc.Content(ctx, file, 0, file.Size-1)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		if got, err := io.ReadAll(r); err != nil || !bytes.Equal(got, pdf()) {
			t.Errorf("Content() = %q, %v", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rejected := clean
	rejected.Status = binary.StatusRejected
	f.repo.files[rejected.ID] = rejected
	chat := f.chatFile(t, pdf(), binary.StatusClean)
	err = f.inReport(t, func(ctx context.Context) error {
		if got, err := f.svc.ReportFiles(ctx, []string{reportR}); err != nil || len(got) != 0 {
			t.Errorf("ReportFiles() = %+v, %v; want none once rejected", got, err)
		}
		if _, err := f.svc.OpenReport(ctx, rejected); !errors.Is(err, binary.ErrNotFound) {
			t.Errorf("OpenReport() of a rejected PDF = %v, want ErrNotFound", err)
		}
		if _, err := f.svc.OpenReport(ctx, chat); !errors.Is(err, binary.ErrNotFound) {
			t.Errorf("OpenReport() of a chat file = %v, want ErrNotFound", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The scan of a report's PDF runs in the report's job scope and never asks the questionnaire holder; a clean PDF
// becomes clean, a flagged one rejected with its object deleted.
func TestScan_AReportsPDFIsScannedInTheReportsScope(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		want   binary.Status
		result clamd.Result
	}{
		{want: binary.StatusClean},
		{want: binary.StatusRejected, result: clamd.Result{Found: true, Signature: "Synthetic.Test"}},
	} {
		f := newFixture(t)
		file := f.reportFile(t, binary.StatusQuarantined)
		args := binary.ScanArgs{BinaryID: file.ID, DiagnosticReportID: reportR, OrganizationID: clinicA}
		got, err := f.scanning(fakeScanner{result: tt.result}).Scan(t.Context(), args)
		if err != nil || got.Status != tt.want || f.repo.files[file.ID].Status != tt.want {
			t.Fatalf("Scan() = %+v, %v; want %s", got, err, tt.want)
		}
		interaction := auditevent.CodeUpdate
		if tt.want == binary.StatusRejected {
			interaction = auditevent.CodeDelete
		}
		if want := chatJobEvent(interaction, file.ID); !slices.Equal(f.audit.events, want) {
			t.Errorf("events = %+v, want %+v", f.audit.events, want)
		}
		if len(f.holder.cleared)+len(f.holder.detached) != 0 {
			t.Error("the questionnaire holder was asked about a report's PDF")
		}
		if (tt.want == binary.StatusRejected) != slices.Contains(f.store.deleted, file.ID) {
			t.Errorf("deleted %v", f.store.deleted)
		}
	}
}
