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
// scope of the file's response (a chat file's conversation) it reads the quarantined file, then, outside any
// transaction, streams its object through the envelope decrypter into clamd. A clean file becomes clean and its
// attachment no longer pending; a clean chat image gets its thumbnail; a file clamd flags becomes rejected, its
// wrapped key cleared and its attachment removed, and its object is deleted after the commit. Each result is
// audited with the job as agent (update for clean, delete for rejected, whose bytes are gone). Any error (clamd
// unreachable, the store, the commit) leaves the file quarantined, so nobody downloads it, and River retries the job;
// a chat file past clamd's stream limit is rejected at once, and one the job gives up on through Unscannable. A file
// that is gone, or no longer quarantined, is left as it is: repeated, the job changes nothing.
func (s *Service) Scan(ctx context.Context, job ScanArgs) (ScanResult, error) {
	b, ok, err := s.quarantined(ctx, job)
	if err != nil || !ok {
		return ScanResult{}, wrapScan(err)
	}
	found, err := s.scan(ctx, b)
	if errors.Is(err, clamd.ErrStreamLimit) && job.ConversationID != "" {
		found, err = clamd.Result{Found: true}, nil // unscannable: a chat file is never left quarantined
	}
	if err != nil {
		return ScanResult{}, wrapScan(err)
	}
	status := StatusClean
	if found.Found {
		status = StatusRejected
	}
	var thumb Thumbnail
	if status == StatusClean && job.ConversationID != "" {
		if thumb, err = s.thumbnail(ctx, b); err != nil {
			return ScanResult{}, wrapScan(err)
		}
	}
	marked, err := s.markScanned(ctx, job, b, status, thumb)
	switch {
	case err != nil || !marked:
		if thumb.ID != "" {
			_ = s.Store.Delete(context.WithoutCancel(ctx), thumb.ID)
		}
		return ScanResult{}, wrapScan(err)
	case status == StatusRejected:
		_ = s.Store.Delete(context.WithoutCancel(ctx), b.ID) // the purger removes an object left behind (SPEC.md §5)
	}
	return ScanResult{Status: status, Signature: found.Signature}, nil
}

// markScanned records the scan's status of b in one transaction of the job's scope, with what goes with it: the
// questionnaire answer's attachment cleared or removed, a chat image's thumbnail, the audit. It reports whether b was
// still quarantined.
func (s *Service) markScanned(ctx context.Context, job ScanArgs, b Binary, status Status, thumb Thumbnail) (bool, error) {
	marked := false
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.jobScope(ctx, job); err != nil {
			return err
		}
		var err error
		if marked, err = s.Repo.MarkScanned(ctx, b.ID, status, s.Clock.Now()); err != nil || !marked {
			return err
		}
		e := entry{agent: agentScan, org: b.OrganizationID, interaction: auditevent.CodeUpdate, binaryID: b.ID, responseID: b.ResponseID}
		if status == StatusRejected {
			e.interaction = auditevent.CodeDelete
		}
		switch {
		case b.ConversationID != "" && thumb.ID != "":
			_, err = s.Repo.SetThumbnail(ctx, b.ID, thumb, s.Clock.Now())
		case b.ConversationID != "": // a chat file's message lists its files from their rows
		case status == StatusClean:
			err = s.Holder.Cleared(ctx, b.ResponseID, b.ID)
		default:
			err = s.Holder.Detach(ctx, b.ResponseID, b.ID)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, e)
	})
	return marked, err
}

// Unscannable rejects a chat file whose scan the job gives up on (its last attempt failed), so it is never left
// quarantined: in the conversation's job scope, audited as the scan's rejection, its object deleted after the commit.
// It reports whether it rejected one; a questionnaire file stays quarantined (nobody downloads it), and a file that
// is gone or already scanned is left as it is.
func (s *Service) Unscannable(ctx context.Context, job ScanArgs) (bool, error) {
	if job.ConversationID == "" {
		return false, nil
	}
	b, ok, err := s.quarantined(ctx, job)
	if err != nil || !ok {
		return false, wrapScan(err)
	}
	marked, err := s.markScanned(ctx, job, b, StatusRejected, Thumbnail{})
	if err != nil || !marked {
		return false, wrapScan(err)
	}
	_ = s.Store.Delete(context.WithoutCancel(ctx), b.ID)
	return true, nil
}

// jobScope scopes the transaction in ctx to the job's file: its response's, or its conversation's.
func (s *Service) jobScope(ctx context.Context, job ScanArgs) error {
	if job.ConversationID != "" {
		return s.Repo.SetChatJob(ctx, job.ConversationID, job.OrganizationID)
	}
	return s.Repo.SetJob(ctx, job.ResponseID, job.OrganizationID)
}

// quarantined reads the job's file in its job scope; false when it is gone or already scanned.
func (s *Service) quarantined(ctx context.Context, job ScanArgs) (Binary, bool, error) {
	var b Binary
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.jobScope(ctx, job); err != nil {
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
