package binary

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// Download opens the file id for whoever may read its response in full (Holder.Readable), once its scan
// says it is clean: ErrScanning while quarantined, ErrNotFound when rejected or not seen. One transaction
// reads the row and audits the read (each request, ranges included: a seek in a video is a read of the file)
// and commits before a byte is sent; Content then streams the bytes.
func (s *Service) Download(ctx context.Context, c Caller, id string) (File, error) {
	b, _, err := s.read(ctx, c, id, func(b Binary) error {
		switch b.Status {
		case StatusClean:
			return nil
		case StatusQuarantined:
			return ErrScanning
		}
		return ErrNotFound
	})
	if err != nil {
		return File{}, fmt.Errorf("binary: download: %w", err)
	}
	if b.KEKID != s.KEK.ID() {
		return File{}, errors.New("binary: download: the data key is wrapped by another key-encryption key")
	}
	dek, err := s.KEK.Unwrap(b.WrappedKey, b.ID)
	if err != nil {
		return File{}, fmt.Errorf("binary: download: %w", err)
	}
	return File{Binary: b, dek: dek}, nil
}

// Content streams the plaintext bytes from to to (inclusive, as an HTTP Range) of a file Download opened:
// the whole ciphertext segments that hold them in one ranged read of the store, decrypted. ErrRangeNotSatisfiable
// for a range outside the file. The caller closes the reader; a read error after the first byte means the
// stored object was tampered with or the store failed.
func (s *Service) Content(ctx context.Context, f File, from, to int64) (io.ReadCloser, error) {
	r, err := envelope.OpenRange(f.dek, f.ID, f.Size, from, to)
	switch {
	case errors.Is(err, envelope.ErrRange):
		return nil, ErrRangeNotSatisfiable
	case err != nil:
		return nil, fmt.Errorf("binary: content: %w", err)
	}
	body, err := s.Store.GetRange(ctx, f.ID, r.Start, r.End)
	if err != nil {
		return nil, fmt.Errorf("binary: content: %w", err)
	}
	return struct {
		io.Reader
		io.Closer
	}{r.Reader(body), body}, nil
}
