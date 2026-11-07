package envelope

import (
	"crypto/cipher"
	"errors"
	"io"
)

var errClosed = errors.New("envelope: the writer is closed")

// CiphertextSize is the length of the ciphertext of a plaintext of n bytes: every segment, the last one
// included (an empty file has one empty segment), grows by Overhead.
func CiphertextSize(n int64) int64 {
	return n + Overhead*segments(n)
}

// segments is the number of segments of a plaintext of n bytes.
func segments(n int64) int64 {
	return max(1, (n+SegmentSize-1)/SegmentSize)
}

// Writer seals what is written to it into segments and writes them to its destination. It holds one segment
// in memory, whatever the file's size. Close seals the last segment; without it the ciphertext does not open.
type Writer struct {
	dst   io.Writer
	aead  cipher.AEAD
	err   error
	id    string
	buf   []byte // the plaintext of the segment being filled
	out   []byte // the sealed segment
	index uint64
	nonce [12]byte
}

// NewWriter returns a Writer that seals for the file id under the data key.
func NewWriter(dst io.Writer, dek []byte, id string) (*Writer, error) {
	aead, err := newGCM(dek)
	if err != nil {
		return nil, err
	}
	return &Writer{
		dst: dst, aead: aead, id: id,
		buf: make([]byte, 0, SegmentSize), out: make([]byte, 0, cipherSegment),
	}, nil
}

// Write buffers p and seals each segment once the next byte shows it is not the last.
func (w *Writer) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n := 0
	for len(p) > 0 {
		if len(w.buf) == SegmentSize {
			if err := w.seal(false); err != nil {
				w.err = err
				return n, err
			}
		}
		c := copy(w.buf[len(w.buf):SegmentSize], p)
		w.buf = w.buf[:len(w.buf)+c]
		p = p[c:]
		n += c
	}
	return n, nil
}

// Close seals the last segment. It does not close the destination.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	if err := w.seal(true); err != nil {
		w.err = err
		return err
	}
	w.err = errClosed
	return nil
}

func (w *Writer) seal(last bool) error {
	sealed := w.aead.Seal(w.out[:0], segmentNonce(w.nonce[:], w.index), w.buf, segmentAD(w.id, w.index, last))
	if _, err := w.dst.Write(sealed); err != nil {
		return err
	}
	w.index++
	w.buf = w.buf[:0]
	return nil
}
