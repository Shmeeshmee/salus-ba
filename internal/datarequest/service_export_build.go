package datarequest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// BuildExport builds the archive of the queued export of args.RequestID (S4): the patient's own record as a FHIR
// Bundle (record.json, from every clinic for the account target, from the one clinic for a clinic target) and, for
// the account target, their account data (account.json), each read audited by the service that owns it as the
// patient's own request, written into one zip, encrypted with a data key of its own and stored; then the export is
// ready for ExportWindow and the request is fulfilled. An export that is not queued (built, failed, gone) is left
// alone.
func (s *Service) BuildExport(ctx context.Context, args ExportArgs) error {
	x := s.p.Export
	var e StoredExport
	err := x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, args.AccountID); err != nil {
			return err
		}
		var err error
		e, err = x.Repo.ExportOf(ctx, args.RequestID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("datarequest: build export: %w", err)
	case e.Status != ExportQueued:
		return nil
	}
	archive, err := s.exportArchive(ctx, e)
	if err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	if err := s.sealExport(ctx, &e, archive); err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	e.ReadyAt = s.p.Clock.Now()
	e.ExpiresAt = e.ReadyAt.Add(ExportWindow)
	err = x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, e.AccountID); err != nil {
			return err
		}
		ok, err := x.Repo.MarkExportReady(ctx, e)
		if err != nil {
			return err
		}
		if !ok { // withdrawn or answered meanwhile: never served; the purger deletes its archive
			_, err := x.Repo.MarkExportFailed(ctx, e.RequestID)
			return err
		}
		if _, err := x.Repo.Fulfil(ctx, e.RequestID); err != nil {
			return err
		}
		return x.Audit.Record(ctx, exportJobEvent(auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	return nil
}

// accountJSON is account.json: the account data of the patient, nothing of its credentials.
type accountJSON struct {
	MFAEnrolledAt *time.Time `json:"mfaEnrolledAt,omitempty"`
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Language      string     `json:"language,omitempty"`
}

// exportArchive reads the patient's data for the export e and writes it as one zip, with manifest.txt last.
func (s *Service) exportArchive(ctx context.Context, e StoredExport) ([]byte, error) {
	x := s.p.Export
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	record, err := x.Records.Bundle(ctx, e.AccountID, e.OrganizationID)
	if err != nil {
		return nil, err
	}
	if record != nil {
		if err := writeExportFile(zw, e, "record.json", record); err != nil {
			return nil, err
		}
	}
	if e.OrganizationID == "" {
		a, err := x.Accounts.Me(ctx, e.AccountID)
		if err != nil {
			return nil, err
		}
		out := accountJSON{ID: a.ID, Email: a.Email, Language: a.Language}
		if !a.MFAEnrolledAt.IsZero() {
			at := a.MFAEnrolledAt.UTC()
			out.MFAEnrolledAt = &at
		}
		body, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := writeExportFile(zw, e, "account.json", append(body, '\n')); err != nil {
			return nil, err
		}
	}
	if err := writeExportFile(zw, e, "manifest.txt", []byte(exportManifest(record != nil, e.OrganizationID == ""))); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeExportFile(zw *zip.Writer, e StoredExport, name string, body []byte) error {
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: e.RequestedAt})
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// exportManifest says what the archive holds and what it does not.
func exportManifest(record, account bool) string {
	out := "Salus: your data\n\n"
	if record {
		out += "record.json   your record as a FHIR R4 Bundle (collection): your profile, your consents, your\n" +
			"              appointments (the cancelled ones too) and the receipts issued to you\n"
	} else {
		out += "You have no patient profile, so there is no record.\n"
	}
	if account {
		out += "account.json  your account: its e-mail, language and when two-step sign-in was set up\n"
	}
	return out + "\nNot in this archive yet: visit notes, documents, questionnaire answers and files. You can read\n" +
		"them in the app, and the clinic that holds them can hand them over.\n"
}

// sealExport encrypts the archive under a new data key bound to the request's id, stores it outside any transaction,
// and sets what the row keeps of it on e.
func (s *Service) sealExport(ctx context.Context, e *StoredExport, archive []byte) error {
	x := s.p.Export
	dek, err := envelope.NewDataKey()
	if err != nil {
		return err
	}
	if e.WrappedKey, err = x.KEK.Wrap(dek, e.RequestID); err != nil {
		return err
	}
	e.KEKID, e.Size = x.KEK.ID(), int64(len(archive))
	var sealed bytes.Buffer
	w, err := envelope.NewWriter(&sealed, dek, e.RequestID)
	if err != nil {
		return err
	}
	if _, err := w.Write(archive); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return x.Store.Put(ctx, ExportObjectKey(e.RequestID), &sealed, int64(sealed.Len()))
}
