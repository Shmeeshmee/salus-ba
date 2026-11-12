package binary

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1; it is no security control (SPEC.md §3 O8)
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/filetype"
)

// headSize is how much of a file is read to detect its type before anything is stored.
const headSize = 4096

// afterBody bounds what an upload still does once its bytes are read: the refusal's audit, or the commit
// and the deletion of an object left over.
const afterBody = 30 * time.Second

// Upload stores one file for the item in.LinkID of the response in.ResponseID, as its patient
// (Holder.UploadTarget), and returns it with whether this call created it (SPEC.md §2 "Upload pipeline"):
//  1. the holder's target and the limits by the declared length, in a transaction of their own;
//  2. the first 4 KiB, whose bytes give the type (never the request's Content-Type or the name);
//  3. the rest streamed through an exact-length reader, the MP4 checker and SHA-1 into the envelope
//     encrypter and one put of known length, outside any transaction, in constant memory;
//  4. one transaction under an advisory lock on the response: the target and the limits again, the row
//     quarantined, the holder's attachment pending, the scan job and the audit.
//
// A file the item already holds with the same SHA-1 and size is returned instead (an identical retry), and
// the new object is deleted, as it is after any failure once stored. Every call is audited once, refusals
// included: the holder's, ErrLengthRequired, ErrTooLarge, ErrResponseFull, ErrTooManyFiles,
// ErrUnsupportedType and ErrShortBody.
func (s *Service) Upload(ctx context.Context, c Caller, in Upload) (Binary, bool, error) {
	t, err := s.precheck(ctx, c, in)
	if err != nil {
		return Binary{}, false, err
	}
	b, err := s.seal(ctx, in, t)
	// Once the bytes are read, the refusal's audit and the commit run even when the client has gone: net/http
	// cancels the request's context when its connection fails mid-body (PROGRESS.md, review of PR #25).
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), afterBody)
	defer cancel()
	if refusal(err) {
		return Binary{}, false, s.refuse(ctx, c, in.ResponseID, t.OrganizationID, err)
	}
	if err != nil {
		return Binary{}, false, fmt.Errorf("binary: upload: %w", err)
	}
	out, created, err := s.commit(ctx, c, in, b)
	if err != nil || !created {
		_ = s.Store.Delete(ctx, b.ID) // the purger removes an object left behind (SPEC.md §5)
	}
	return out, created, err
}

// precheck is step 1: the holder's target, or the refusal, audited.
func (s *Service) precheck(ctx context.Context, c Caller, in Upload) (Target, error) {
	var t Target
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.scope(ctx, c); err != nil {
			return err
		}
		var err error
		if t, err = s.Holder.UploadTarget(ctx, c, in.ResponseID, in.LinkID); err != nil {
			return s.holderRefusal(ctx, c, in.ResponseID, c.OrganizationID, err, &refused)
		}
		var largest int64
		for _, k := range t.Accept {
			largest = max(largest, k.Limit())
		}
		switch {
		case in.Size < 0:
			refused = ErrLengthRequired
		case in.Size > largest:
			refused = ErrTooLarge
		default:
			if refused, err = s.room(ctx, in, t, nil); err != nil {
				return err
			}
		}
		if refused != nil {
			return s.record(ctx, refusedUpload(c, in.ResponseID, t.OrganizationID))
		}
		return nil
	})
	switch {
	case err != nil:
		return Target{}, fmt.Errorf("binary: upload: %w", err)
	case refused != nil:
		return Target{}, refused
	}
	return t, nil
}

// holderRefusal audits the holder's refusal err of an upload, in the clinic org as far as the call knows it,
// and keeps it in refused, or returns err when it is a failure. ErrNotFound discloses nothing; the other
// refusals name the response the caller sees.
func (s *Service) holderRefusal(ctx context.Context, c Caller, responseID, org string, err error, refused *error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		responseID = ""
	case !refusal(err):
		return err
	}
	*refused = err
	return s.record(ctx, refusedUpload(c, responseID, org))
}

// refusedUpload is the audit entry of a refused upload by c: the response's, when it is disclosed.
func refusedUpload(c Caller, responseID, org string) entry {
	return entry{c: c, org: org, interaction: auditevent.CodeCreate, responseID: responseID, outcome: auditevent.OutcomeMinorFailure}
}

// refuse audits the refusal err of an upload to the response in the clinic org, found once its bytes were
// read, and returns it.
func (s *Service) refuse(ctx context.Context, c Caller, responseID, org string, err error) error {
	if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error { return s.record(ctx, refusedUpload(c, responseID, org)) }); aerr != nil {
		return fmt.Errorf("binary: upload: %w", aerr)
	}
	return err
}

// room is the refusal of an upload in of in.Size bytes to the target t when the item's files or the
// response's bytes leave no room for it, read in the transaction in ctx; nil when there is room. With
// b, the file stored for it, an identical file the item holds is not refused here (the commit returns it);
// without, a file the size of one the item holds may be its identical retry, so the limits wait for its bytes.
func (s *Service) room(ctx context.Context, in Upload, t Target, b *Binary) (error, error) {
	files, err := s.Repo.OfItem(ctx, in.ResponseID, in.LinkID)
	if err != nil {
		return nil, err
	}
	if b == nil && slices.ContainsFunc(files, func(f Binary) bool { return f.Size == in.Size }) {
		return nil, nil
	}
	used, err := s.Repo.ResponseSize(ctx, in.ResponseID)
	switch {
	case err != nil:
		return nil, err
	case used+in.Size > MaxResponseSize:
		return ErrResponseFull, nil
	case len(files) >= t.MaxFiles:
		return ErrTooManyFiles, nil
	}
	return nil, nil
}

// seal is steps 2 and 3: the type from the first bytes, then the file encrypted into the store under a new
// id; it returns the file's row, not yet stored.
func (s *Service) seal(ctx context.Context, in Upload, t Target) (Binary, error) {
	return s.sealFile(ctx, in.Body, in.Size, t.Accept, Binary{
		OrganizationID: t.OrganizationID, PatientID: t.PatientID, AppointmentID: t.AppointmentID, ResponseID: in.ResponseID,
		LinkID: in.LinkID, Title: in.Title,
	})
}

// sealFile reads exactly size bytes of src: its type from the first bytes, one of accept and within its kind's
// limit, then the whole file encrypted into the store under a new id; it returns b, its parent and title set by the
// caller, with the file's id, type, size, SHA-1 and wrapped key, not yet stored.
func (s *Service) sealFile(ctx context.Context, src io.Reader, size int64, accept []Kind, b Binary) (Binary, error) {
	body := filetype.LimitReader(src, size)
	head := make([]byte, min(headSize, size))
	if _, err := io.ReadFull(body, head); err != nil {
		return Binary{}, bodyError(err)
	}
	typ, err := filetype.Detect(head)
	kind := kindOf(string(typ))
	switch {
	case err != nil, !slices.Contains(accept, kind):
		return Binary{}, ErrUnsupportedType
	case size > kind.Limit():
		return Binary{}, ErrTooLarge
	}
	b.ID, b.ContentType, b.KEKID, b.Size, b.SegmentSize = newID(), string(typ), s.KEK.ID(), size, envelope.SegmentSize
	dek, err := envelope.NewDataKey()
	if err != nil {
		return Binary{}, err
	}
	if b.WrappedKey, err = s.KEK.Wrap(dek, b.ID); err != nil {
		return Binary{}, err
	}
	b.SHA1, err = s.put(ctx, b, dek, io.MultiReader(bytes.NewReader(head), body), typ == filetype.MP4)
	return b, err
}

// errStopped stops the sealing goroutine once the put is over.
var errStopped = errors.New("binary: the put is over")

// put streams src (the whole plaintext of b) through SHA-1, the MP4 checker when mp4, and the envelope
// encrypter into the store, and returns the SHA-1. The last segment is sealed only once the checker has seen
// the whole file, so a refused file never completes its put. A refusal of the bytes wins over the put's error
// it causes.
func (s *Service) put(ctx context.Context, b Binary, dek []byte, src io.Reader, mp4 bool) ([]byte, error) {
	pr, pw := io.Pipe()
	h := sha1.New() //nolint:gosec // see the import
	done := make(chan error, 1)
	go func() {
		err := encrypt(pw, dek, b.ID, src, h, mp4)
		pw.CloseWithError(err) // nil closes it with EOF
		done <- err
	}()
	putErr := s.Store.Put(ctx, b.ID, pr, envelope.CiphertextSize(b.Size))
	pr.CloseWithError(errStopped) // a put that returned early leaves the goroutine nothing to block on
	sealErr := <-done
	switch {
	case sealErr != nil && !errors.Is(sealErr, errStopped):
		if putErr == nil { // cannot happen: the last segment follows the checks; never keep such an object
			_ = s.Store.Delete(context.WithoutCancel(ctx), b.ID)
		}
		return nil, bodyError(sealErr)
	case putErr != nil:
		return nil, putErr
	}
	return h.Sum(nil), nil
}

// encrypt seals src into dst for the file id, teeing the plaintext into h and, for an MP4, the checker.
func encrypt(dst io.Writer, dek []byte, id string, src io.Reader, h io.Writer, mp4 bool) error {
	w, err := envelope.NewWriter(dst, dek, id)
	if err != nil {
		return err
	}
	var check *filetype.Checker
	sink := h
	if mp4 {
		check = filetype.NewChecker()
		sink = io.MultiWriter(h, check)
	}
	if _, err := io.Copy(w, io.TeeReader(src, sink)); err != nil {
		return err
	}
	if check != nil {
		if err := check.Close(); err != nil {
			return err
		}
	}
	return w.Close()
}

// bodyError maps an error reading or checking the upload's bytes to its refusal; others pass on.
func bodyError(err error) error {
	switch {
	case errors.Is(err, filetype.ErrUnsupported), errors.Is(err, filetype.ErrMalformed):
		return ErrUnsupportedType
	case errors.Is(err, filetype.ErrShort), errors.Is(err, filetype.ErrTooLarge), errors.Is(err, io.ErrUnexpectedEOF):
		return ErrShortBody
	}
	return err
}

// commit is step 4: the stored file b recorded and attached, or the identical file the item already holds,
// or the refusal (audited, committed).
func (s *Service) commit(ctx context.Context, c Caller, in Upload, b Binary) (Binary, bool, error) {
	var out Binary
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.scope(ctx, c); err != nil {
			return err
		}
		t, err := s.Holder.UploadTarget(ctx, c, in.ResponseID, in.LinkID)
		if err != nil {
			return s.holderRefusal(ctx, c, in.ResponseID, b.OrganizationID, err, &refused)
		}
		if err := s.Repo.LockResponse(ctx, in.ResponseID); err != nil {
			return err
		}
		e := entry{c: c, org: t.OrganizationID, interaction: auditevent.CodeCreate, responseID: in.ResponseID}
		if out, err = s.identical(ctx, b); err != nil {
			return err
		}
		if out.ID != "" {
			e.binaryID = out.ID
			return s.record(ctx, e)
		}
		if refused, err = s.room(ctx, in, t, &b); err != nil {
			return err
		}
		if refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		if out, err = s.attach(ctx, b); err != nil {
			return err
		}
		e.binaryID = out.ID
		return s.record(ctx, e)
	})
	switch {
	case err != nil:
		return Binary{}, false, fmt.Errorf("binary: upload: %w", err)
	case refused != nil:
		return Binary{}, false, refused
	}
	return out, out.ID == b.ID, nil
}

// identical is the file the item of b already holds with b's SHA-1 and size, or a zero Binary.
func (s *Service) identical(ctx context.Context, b Binary) (Binary, error) {
	files, err := s.Repo.OfItem(ctx, b.ResponseID, b.LinkID)
	if err != nil {
		return Binary{}, err
	}
	for _, f := range files {
		if f.Size == b.Size && bytes.Equal(f.SHA1, b.SHA1) {
			return f, nil
		}
	}
	return Binary{}, nil
}

// attach stores the row of b, quarantined, lists it pending in its holder's answer and enqueues its scan.
func (s *Service) attach(ctx context.Context, b Binary) (Binary, error) {
	out, err := s.Repo.Insert(ctx, b, s.Clock.Now())
	if err != nil {
		return Binary{}, err
	}
	err = s.Holder.Attach(ctx, out.ResponseID, Attachment{
		LinkID: out.LinkID, BinaryID: out.ID, ContentType: out.ContentType, Title: out.Title, Hash: out.SHA1, Size: out.Size, Pending: true,
	})
	if err != nil {
		return Binary{}, err
	}
	return out, s.Jobs.EnqueueScan(ctx, ScanArgs{BinaryID: out.ID, ResponseID: out.ResponseID, OrganizationID: out.OrganizationID})
}

// newID is a random (version 4) UUID in its canonical lowercase form: the file's id, its object key and the
// additional data of its encryption.
func newID() string {
	var u [16]byte
	_, _ = rand.Read(u[:]) // crypto/rand never fails
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}
