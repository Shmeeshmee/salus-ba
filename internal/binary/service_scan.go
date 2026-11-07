package binary

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// Scanner scans a file's plaintext for malware (pkg/clamd.Client, the compose clamd).
type Scanner interface {
	Scan(ctx context.Context, r io.Reader) (clamd.Result, error)
}

// The jobs, as the agents of their audit events (their kinds).
const (
	agentScan    = "binary_scan"
	agentDiscard = "binary_discard"
)

// ScanResult is what a scan did: the file's new status, empty when there was nothing to scan, and for a
// rejected file the signature clamd found (a ClamAV name, not personal data; never logged beside a patient id).
type ScanResult struct {
	Status    Status
	Signature string
}

// Scan is the binary_scan job, enqueued with every upload (docs/attachments/SPEC.md §2 "Scan"). In the job
// scope of the file's response it reads the quarantined file, then, outside any transaction, streams its object
// through the envelope decrypter into clamd. A clean file becomes clean and its attachment no longer pending; a
// file clamd flags becomes rejected, its wrapped key cleared and its attachment removed, and its object is
// deleted after the commit. Each result is audited with the job as agent (update for clean, delete for rejected,
// whose bytes are gone). Any error (clamd unreachable, the store, the commit) leaves the file quarantined, so
// nobody downloads it, and River retries the job. A file that is gone, or no longer quarantined, is left as it is:
// repeated, the job changes nothing.
func (s *Service) Scan(ctx context.Context, job ScanArgs) (ScanResult, error) {
	b, ok, err := s.quarantined(ctx, job)
	if err != nil || !ok {
		return ScanResult{}, wrapScan(err)
	}
	found, err := s.scan(ctx, b)
	if err != nil {
		return ScanResult{}, wrapScan(err)
	}
	status := StatusClean
	if found.Found {
		status = StatusRejected
	}
	marked := false
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetJob(ctx, job.ResponseID, job.OrganizationID); err != nil {
			return err
		}
		var err error
		if marked, err = s.Repo.MarkScanned(ctx, b.ID, status, s.Clock.Now()); err != nil || !marked {
			return err
		}
		e := entry{agent: agentScan, org: b.OrganizationID, interaction: auditevent.CodeUpdate, binaryID: b.ID, responseID: b.ResponseID}
		if status == StatusClean {
			err = s.Holder.Cleared(ctx, b.ResponseID, b.ID)
		} else {
			e.interaction = auditevent.CodeDelete
			err = s.Holder.Detach(ctx, b.ResponseID, b.ID)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, e)
	})
	switch {
	case err != nil:
		return ScanResult{}, wrapScan(err)
	case !marked:
		return ScanResult{}, nil
	case status == StatusRejected:
		_ = s.Store.Delete(context.WithoutCancel(ctx), b.ID) // the purger removes an object left behind (SPEC.md §5)
	}
	return ScanResult{Status: status, Signature: found.Signature}, nil
}

// quarantined reads the job's file in its job scope; false when it is gone or already scanned.
func (s *Service) quarantined(ctx context.Context, job ScanArgs) (Binary, bool, error) {
	var b Binary
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetJob(ctx, job.ResponseID, job.OrganizationID); err != nil {
			return err
		}
		var err error
		b, err = s.Repo.ByID(ctx, job.BinaryID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Binary{}, false, nil
	case err != nil:
		return Binary{}, false, err
	}
	return b, b.Status == StatusQuarantined, nil
}

// scan streams the plaintext of b from the store into the scanner.
func (s *Service) scan(ctx context.Context, b Binary) (clamd.Result, error) {
	if b.KEKID != s.KEK.ID() {
		return clamd.Result{}, errors.New("the data key is wrapped by another key-encryption key")
	}
	dek, err := s.KEK.Unwrap(b.WrappedKey, b.ID)
	if err != nil {
		return clamd.Result{}, err
	}
	body, err := s.Store.GetRange(ctx, b.ID, 0, envelope.CiphertextSize(b.Size)-1)
	if err != nil {
		return clamd.Result{}, err
	}
	defer func() { _ = body.Close() }()
	plain, err := envelope.Open(body, dek, b.ID)
	if err != nil {
		return clamd.Result{}, err
	}
	return s.Scanner.Scan(ctx, plain)
}

func wrapScan(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("binary: scan: %w", err)
}
