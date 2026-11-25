package binary

import (
	"context"
	"fmt"
	"io"
)

// The lab results' PDFs (docs/clinic-cms/WAVE6.md U5): a file's third kind of parent is a lab result
// (diagnostic_report). The report (diagnosticreport) decides who uploads and reads its PDF and audits each call, in
// the transactions binary joins; binary keeps the rules of the file itself: a PDF from its bytes, its size, its
// encryption, its scan before anyone opens it. A report holds one PDF: a new upload replaces the one there.

// reportKinds are what a lab result takes: its PDF.
var reportKinds = []Kind{KindDocument}

// ReportUpload is the lab's PDF of a report: exactly Size bytes of Body (-1 when the request gave no length).
// OrganizationID and PatientID are the report's.
type ReportUpload struct {
	Body           io.Reader
	ReportID       string
	OrganizationID string
	PatientID      string
	Title          string
	Size           int64
}

// UploadReport stores the PDF of a report (as UploadChat does a chat file): its type from the first bytes, a PDF of
// at most MaxDocumentSize bytes, streamed sealed into the store outside any transaction; then one transaction runs
// commit first (the report's checks and its audit, which set the clinic's scope), deletes the report's PDF already
// there, stores the row quarantined under the report and enqueues its scan. ErrLengthRequired, ErrTooLarge,
// ErrUnsupportedType and ErrShortBody refuse the bytes before commit runs; commit's error is returned as it is. After
// any failure once stored, the new object is deleted; after the commit, the replaced one. Nothing here is audited: the
// report audits the call it serves.
func (s *Service) UploadReport(ctx context.Context, in ReportUpload, commit func(ctx context.Context, b Binary) error) (Binary, error) {
	switch {
	case in.Size < 0:
		return Binary{}, ErrLengthRequired
	case in.Size > MaxDocumentSize:
		return Binary{}, ErrTooLarge
	}
	b, err := s.sealFile(ctx, in.Body, in.Size, reportKinds, Binary{
		OrganizationID: in.OrganizationID, PatientID: in.PatientID, DiagnosticReportID: in.ReportID, Title: in.Title,
	})
	if err != nil {
		return Binary{}, err
	}
	// Once the bytes are read, the commit runs even when the client has gone (as Upload's).
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), afterBody)
	defer cancel()
	var out Binary
	var replaced []string
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := commit(ctx, b); err != nil {
			return err
		}
		var err error
		if replaced, err = s.replace(ctx, b.DiagnosticReportID); err != nil {
			return err
		}
		if out, err = s.Repo.Insert(ctx, b, s.Clock.Now()); err != nil {
			return err
		}
		return s.Jobs.EnqueueScan(ctx, ScanArgs{BinaryID: out.ID, DiagnosticReportID: out.DiagnosticReportID, OrganizationID: out.OrganizationID})
	})
	if err != nil {
		_ = s.Store.Delete(ctx, b.ID) // the purger removes an object left behind (SPEC.md §5)
		return Binary{}, err
	}
	for _, id := range replaced {
		_ = s.Store.Delete(ctx, id) // as above
	}
	return out, nil
}

// replace deletes the files of the report in the scope of the transaction in ctx and returns their ids.
func (s *Service) replace(ctx context.Context, reportID string) ([]string, error) {
	files, err := s.Repo.OfReports(ctx, []string{reportID})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(files))
	for _, f := range files {
		ok, err := s.Repo.Delete(ctx, f.ID)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return nil, fmt.Errorf("binary: replace: the row was not deleted")
		}
		ids = append(ids, f.ID)
	}
	return ids, nil
}

// ReportFiles returns the PDFs of the reports that are not rejected, as the report's scope shows them, oldest first.
func (s *Service) ReportFiles(ctx context.Context, reportIDs []string) ([]Binary, error) {
	if len(reportIDs) == 0 {
		return nil, nil
	}
	files, err := s.Repo.OfReports(ctx, reportIDs)
	if err != nil {
		return nil, fmt.Errorf("binary: report files: %w", err)
	}
	out := files[:0]
	for _, f := range files {
		if f.Status != StatusRejected {
			out = append(out, f)
		}
	}
	return out, nil
}

// OpenReport opens a report's PDF the report has let its caller read, once its scan says it is clean: ErrScanning
// while quarantined, ErrNotFound when rejected or for a file of no report. Content then streams it.
func (s *Service) OpenReport(ctx context.Context, b Binary) (File, error) {
	if b.DiagnosticReportID == "" {
		return File{}, ErrNotFound
	}
	return s.OpenChat(ctx, b, false)
}
