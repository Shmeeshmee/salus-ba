package datarequest

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// BuildExport builds the archive of the queued export of args.RequestID (S4, D7): what the patient reads in the app,
// per clinic (every clinic for the account target, the one clinic for a clinic target) as FHIR Bundles beside their
// files, and, for the account target, their account data and profile, each read audited by the service that owns it
// as the patient's own request, every audit event naming the request (AuditRequestID); written into one zip, encrypted
// with a data key of its own and stored under this attempt's own key; then the export is ready for ExportWindow from
// now (S13) and the request is fulfilled. An export that is not queued (built, failed, gone) is left alone.
func (s *Service) BuildExport(ctx context.Context, args ExportArgs) error {
	ctx = forRequest(ctx, args.RequestID)
	e, previous, err := s.claimExport(ctx, args)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	if previous != "" { // an earlier attempt's archive: its claim is gone, it can never be served
		if err := s.p.Export.Store.Delete(ctx, previous); err != nil {
			return fmt.Errorf("datarequest: build export: %w", err)
		}
	}
	archive, err := s.exportArchive(ctx, e)
	if err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	if err := s.sealExport(ctx, &e, archive); err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	if err := s.readyExport(ctx, e); err != nil {
		return fmt.Errorf("datarequest: build export: %w", err)
	}
	return nil
}

// claimExport reads the queued export of args and claims a new object key for this attempt (D7), returning the export
// with that key and the key of an earlier attempt it replaced. ErrNotFound for an export that is not queued.
func (s *Service) claimExport(ctx context.Context, args ExportArgs) (StoredExport, string, error) {
	x := s.p.Export
	attempt, err := newAttemptID()
	if err != nil {
		return StoredExport{}, "", err
	}
	var e StoredExport
	var previous string
	err = x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, args.AccountID); err != nil {
			return err
		}
		var err error
		if e, err = x.Repo.ExportOf(ctx, args.RequestID); err != nil {
			return err
		}
		e.ObjectKey = ExportObjectKey(e.RequestID, attempt)
		var ok bool
		if previous, ok, err = x.Repo.ClaimExport(ctx, e.RequestID, e.ObjectKey); err != nil {
			return err
		}
		if !ok { // built, failed or gone
			return ErrNotFound
		}
		return nil
	})
	return e, previous, err
}

// readyExport makes the stored export e ready for ExportWindow from now (S13) and fulfils its request. When its
// request was withdrawn or answered meanwhile, or a later attempt claimed the export, its archive is deleted and never
// served; the export is failed only while this attempt's claim is still its own (a later attempt may still build).
func (s *Service) readyExport(ctx context.Context, e StoredExport) error {
	x := s.p.Export
	e.ReadyAt = s.p.Clock.Now()
	e.ExpiresAt = e.ReadyAt.Add(ExportWindow)
	var lost bool
	err := x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, e.AccountID); err != nil {
			return err
		}
		ok, err := x.Repo.MarkExportReady(ctx, e)
		if err != nil {
			return err
		}
		if !ok {
			lost = true
			_, err := x.Repo.MarkClaimFailed(ctx, e) // only while the claim is ours: a later attempt may still build
			return err
		}
		if _, err := x.Repo.Fulfil(ctx, e.RequestID); err != nil {
			return err
		}
		return x.Audit.Record(ctx, exportJobEvent(auditevent.OutcomeSuccess))
	})
	if err != nil {
		return err
	}
	if lost {
		return x.Store.Delete(ctx, e.ObjectKey)
	}
	return nil
}

// newAttemptID is a random id of one build attempt, for its object key.
func newAttemptID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// accountJSON is account.json: the account data of the patient, nothing of its credentials.
type accountJSON struct {
	MFAEnrolledAt *time.Time `json:"mfaEnrolledAt,omitempty"`
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Language      string     `json:"language,omitempty"`
}

// exportArchive reads the patient's data for the export e and writes it as one zip, with manifest.txt last: for the
// account target account.json and patient.json (the profile), and for each clinic (the one of a clinic target)
// clinics/<organization id>/ with its record.json, quotes.json and files.
func (s *Service) exportArchive(ctx context.Context, e StoredExport) ([]byte, error) {
	x := s.p.Export
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	record, err := x.Records.Export(ctx, e.AccountID, e.OrganizationID)
	if err != nil {
		return nil, err
	}
	if e.OrganizationID == "" {
		if err := s.writeAccount(ctx, zw, e); err != nil {
			return nil, err
		}
		if record != nil {
			if err := writeExportFile(zw, e, "patient.json", record.Patient); err != nil {
				return nil, err
			}
		}
	}
	if record != nil {
		for _, p := range record.Parts {
			for _, f := range p.Files {
				if err := writeExportFile(zw, e, clinicDir(p.OrganizationID)+f.Name, f.Body); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := writeExportFile(zw, e, "manifest.txt", []byte(exportManifest(record, e.OrganizationID == ""))); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeAccount writes account.json: the patient's account, nothing of its credentials.
func (s *Service) writeAccount(ctx context.Context, zw *zip.Writer, e StoredExport) error {
	a, err := s.p.Export.Accounts.Me(ctx, e.AccountID)
	if err != nil {
		return err
	}
	out := accountJSON{ID: a.ID, Email: a.Email, Language: a.Language}
	if !a.MFAEnrolledAt.IsZero() {
		at := a.MFAEnrolledAt.UTC()
		out.MFAEnrolledAt = &at
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return writeExportFile(zw, e, "account.json", append(body, '\n'))
}

// clinicDir is the folder of the clinic id's part in the archive.
func clinicDir(id string) string { return "clinics/" + id + "/" }

func writeExportFile(zw *zip.Writer, e StoredExport, name string, body []byte) error {
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: e.RequestedAt})
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// exportManifest says what the archive holds, clinic by clinic, and what it does not.
func exportManifest(record *clinicexport.OwnExport, account bool) string {
	out := "Salus: your data\n\nEach record.json is a FHIR R4 Bundle (collection). Files are named by opaque ids; the\n" +
		"records refer to them, and to the clinics, by the same ids.\n\n"
	if account {
		out += "account.json  your account: its e-mail, language and when two-step sign-in was set up\n"
	}
	switch {
	case record == nil:
		out += "You have no patient profile, so there is no record.\n"
	case account:
		out += "patient.json  your profile\n"
	}
	if record != nil {
		for _, p := range record.Parts {
			out += "\n" + clinicDir(p.OrganizationID) + "  what you can read in the app of this clinic:\n"
			for _, h := range p.Holds {
				out += "  " + h + "\n"
			}
			for _, f := range p.Unreadable { // WAVE6.md PX: named, the rest of the archive still built
				out += "  could not be opened, so not in this archive: " + f + "\n"
			}
		}
	}
	return out
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
	return x.Store.Put(ctx, e.ObjectKey, &sealed, int64(sealed.Len()))
}
