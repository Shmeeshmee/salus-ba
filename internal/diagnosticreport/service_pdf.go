package diagnosticreport

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
)

// Files are the reports' PDFs through binary's upload and scan pipeline (binary.Service, ADR 0010): binary keeps the
// rules of a file (a PDF from its bytes, its size, its encryption, its scan); the report decides who uploads and
// reads one, and audits it, in the transactions binary joins (its scope set by the report).
type Files interface {
	UploadReport(ctx context.Context, in binary.ReportUpload, commit func(ctx context.Context, b binary.Binary) error) (binary.Binary, error)
	ReportFiles(ctx context.Context, reportIDs []string) ([]binary.Binary, error)
	OpenReport(ctx context.Context, b binary.Binary) (binary.File, error)
	Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error)
}

// AttachPDF stores the lab's PDF of a preliminary report at its version, replacing one already there, for the staff
// who may change the report (Change's rule): the bytes go through binary (a PDF only, at most binary.MaxDocumentSize,
// encrypted, scanned before anyone opens it) after a first check, and the report is checked again, moved to its next
// version and audited in the transaction that stores the file. ErrReleased once released (a released report's PDF is
// frozen), ErrVersionConflict when it moved on, ErrInvalid for a file that is not a PDF, ErrTooLarge, ErrBadBody; every
// refusal is audited.
func (s *Service) AttachPDF(ctx context.Context, c Staff, id string, in PDFUpload) (Report, error) {
	var r Report
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		r, err = s.writable(ctx, c, id, in.Version)
		return err
	})
	if err != nil {
		return Report{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeUpdate, id, err)
	}
	var out Report
	_, err = s.p.Files.UploadReport(ctx, binary.ReportUpload{
		Body: in.Body, ReportID: r.ID, OrganizationID: r.OrganizationID, PatientID: r.PatientID, Title: in.Title, Size: in.Size,
	}, func(ctx context.Context, _ binary.Binary) error {
		again, err := s.writable(ctx, c, id, in.Version)
		if err != nil {
			return err
		}
		ok, err := s.p.Repo.Touch(ctx, id, in.Version)
		switch {
		case err != nil:
			return fmt.Errorf("diagnosticreport: attach pdf: %w", err)
		case !ok:
			return ErrVersionConflict
		}
		out = again
		out.VersionID, out.HasPDF = in.Version+1, true
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		// The bytes are read: the refusal is audited even when the client has gone.
		return Report{}, s.refused(context.WithoutCancel(ctx), c.AccountID, c.OrganizationID, auditevent.CodeUpdate, id, fileError(err))
	}
	return out, nil
}

// writable is the report id at version, when the caller may change it and it is not released.
func (s *Service) writable(ctx context.Context, c Staff, id string, version int) (Report, error) {
	r, err := s.report(ctx, c, id)
	switch {
	case err != nil:
		return Report{}, err
	case r.Status == StatusFinal:
		return Report{}, ErrReleased
	case r.VersionID != version:
		return Report{}, ErrVersionConflict
	}
	return r, nil
}

// fileError is binary's refusal of an upload's bytes as the report's.
func fileError(err error) error {
	switch {
	case errors.Is(err, binary.ErrUnsupportedType):
		return ErrInvalid
	case errors.Is(err, binary.ErrTooLarge):
		return ErrTooLarge
	case errors.Is(err, binary.ErrLengthRequired), errors.Is(err, binary.ErrShortBody):
		return ErrBadBody
	}
	return err
}

// PDF opens the PDF of a report of the staff's clinic for the staff who may see the report (Read's rule), once its
// scan says it is clean: ErrNotFound without one, or while it is scanned. The read is audited and committed before a
// byte is sent; Content then streams it.
func (s *Service) PDF(ctx context.Context, c Staff, id string) (binary.File, error) {
	var f binary.File
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.report(ctx, c, id); err != nil {
			return err
		}
		var err error
		if f, err = s.open(ctx, id); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, staffEvent(c, auditevent.CodeRead, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return binary.File{}, s.refused(ctx, c.AccountID, c.OrganizationID, auditevent.CodeRead, id, err)
	}
	return f, nil
}

// MyPDF opens the PDF of one of the patient's released reports, once clean; ErrNotFound (audited) for any other, a
// preliminary one first of all.
func (s *Service) MyPDF(ctx context.Context, c Patient, id string) (binary.File, error) {
	var f binary.File
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if c.PatientID == "" {
			return ErrNotFound
		}
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		r, err := s.p.Repo.ByID(ctx, id)
		switch {
		case err != nil:
			return err
		case r.Status != StatusFinal: // the patient's scope shows no other; never serve one
			return ErrNotFound
		}
		if f, err = s.open(ctx, id); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, patientEvent(c, auditevent.CodeRead, id, auditevent.OutcomeSuccess))
	})
	if errors.Is(err, ErrNotFound) {
		return binary.File{}, s.refused(ctx, c.AccountID, "", auditevent.CodeRead, id, err)
	}
	return f, err
}

// Content streams the bytes from to to (inclusive) of a PDF that PDF or MyPDF opened (binary.Service.Content).
func (s *Service) Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error) {
	return s.p.Files.Content(ctx, f, from, to)
}

// open opens the clean PDF of the report id in the transaction's scope; ErrNotFound for none, or one not clean.
func (s *Service) open(ctx context.Context, id string) (binary.File, error) {
	files, err := s.p.Files.ReportFiles(ctx, []string{id})
	switch {
	case err != nil:
		return binary.File{}, fmt.Errorf("diagnosticreport: pdf: %w", err)
	case len(files) == 0:
		return binary.File{}, ErrNotFound
	}
	f, err := s.p.Files.OpenReport(ctx, files[len(files)-1])
	switch {
	case errors.Is(err, binary.ErrScanning), errors.Is(err, binary.ErrNotFound):
		return binary.File{}, ErrNotFound
	case err != nil:
		return binary.File{}, fmt.Errorf("diagnosticreport: pdf: %w", err)
	}
	return f, nil
}

// withPDFs sets HasPDF on the reports of list that have a PDF that is not rejected, as the transaction's scope shows.
func (s *Service) withPDFs(ctx context.Context, list []Report) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	for i, r := range list {
		ids[i] = r.ID
	}
	files, err := s.p.Files.ReportFiles(ctx, ids)
	if err != nil {
		return fmt.Errorf("diagnosticreport: pdfs: %w", err)
	}
	has := make(map[string]bool, len(files))
	for _, f := range files {
		has[f.DiagnosticReportID] = true
	}
	for i := range list {
		list[i].HasPDF = has[list[i].ID]
	}
	return nil
}
