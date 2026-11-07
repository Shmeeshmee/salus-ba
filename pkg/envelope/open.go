package envelope

import (
	"bufio"
	"crypto/cipher"
	"errors"
	"io"
)

// Open returns a reader of the plaintext of a whole ciphertext, read from src one segment at a time. A read
// returns ErrOpen once a segment does not open, which includes a truncated or extended file; plaintext read
// before that came from segments that did open.
func Open(src io.Reader, dek []byte, id string) (io.Reader, error) {
	aead, err := newGCM(dek)
	if err != nil {
		return nil, err
	}
	rd := newReader(aead, id)
	rd.peek = bufio.NewReader(src)
	rd.src = rd.peek
	return rd, nil
}

// Range is a plaintext range of a file and the ciphertext bytes that hold it.
type Range struct {
	aead cipher.AEAD
	id   string
	// Start and End are the first and the last ciphertext byte to fetch, inclusive like an HTTP Range: whole
	// segments, from the one holding the range's first byte to the one holding its last.
	Start, End     int64
	size, from, to int64
}

// OpenRange maps the plaintext bytes from to to (inclusive, like an HTTP Range) of a file of size plaintext
// bytes to the ciphertext to fetch. Range.Reader then decrypts what was fetched.
func OpenRange(dek []byte, id string, size, from, to int64) (Range, error) {
	aead, err := newGCM(dek)
	if err != nil {
		return Range{}, err
	}
	if size <= 0 || from < 0 || from > to || to >= size {
		return Range{}, ErrRange
	}
	return Range{
		aead: aead, id: id, size: size, from: from, to: to,
		Start: from / SegmentSize * cipherSegment,
		End:   min((to/SegmentSize+1)*cipherSegment, CiphertextSize(size)) - 1,
	}, nil
}

// Reader decrypts the ciphertext from Start to End read from src and yields exactly the range's plaintext.
func (r Range) Reader(src io.Reader) io.Reader {
	rd := newReader(r.aead, r.id)
	rd.src = src
	rd.index = uint64(r.from / SegmentSize)
	rd.ranged = true
	rd.cipherSize = CiphertextSize(r.size)
	rd.skip = int(r.from % SegmentSize)
	rd.left = r.to - r.from + 1
	return rd
}

type reader struct {
	src        io.Reader
	aead       cipher.AEAD
	peek       *bufio.Reader // the whole file's source, to tell the last segment by what follows it
	err        error         // sticky: io.EOF once done, or the first failure
	id         string
	cbuf       []byte
	pbuf       []byte
	plain      []byte // opened plaintext not yet read
	cipherSize int64  // ranged: the whole ciphertext's length, which gives each segment's length
	left       int64  // ranged: plaintext bytes still to yield
	index      uint64
	skip       int // ranged: bytes of the first segment before the range
	nonce      [12]byte
	ranged     bool
}

func newReader(aead cipher.AEAD, id string) *reader {
	return &reader{
		aead: aead, id: id,
		cbuf: make([]byte, cipherSegment), pbuf: make([]byte, 0, SegmentSize),
	}
}

func (r *reader) Read(p []byte) (int, error) {
	for len(r.plain) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		if err := r.next(); err != nil {
			r.err = err
			return 0, err
		}
	}
	n := copy(p, r.plain)
	r.plain = r.plain[n:]
	return n, nil
}

// next reads and opens one segment.
func (r *reader) next() error {
	c, last, err := r.read()
	if err != nil {
		return err
	}
	plain, err := r.aead.Open(r.pbuf[:0], segmentNonce(r.nonce[:], r.index), c, segmentAD(r.id, r.index, last))
	if err != nil {
		return ErrOpen
	}
	r.index++
	if !r.ranged {
		r.plain = plain
		if last {
			r.err = io.EOF
		}
		return nil
	}
	plain = plain[min(r.skip, len(plain)):]
	r.skip = 0
	plain = plain[:min(int64(len(plain)), r.left)]
	r.left -= int64(len(plain))
	r.plain = plain
	if r.left == 0 {
		r.err = io.EOF
	}
	return nil
}

// read returns the next ciphertext segment and whether it is the file's last.
func (r *reader) read() ([]byte, bool, error) {
	if r.ranged {
		n := min(cipherSegment, r.cipherSize-int64(r.index)*cipherSegment)
		if n <= 0 {
			return nil, false, ErrOpen
		}
		if _, err := io.ReadFull(r.src, r.cbuf[:n]); err != nil {
			return nil, false, sourceError(err)
		}
		return r.cbuf[:n], int64(r.index+1)*cipherSegment >= r.cipherSize, nil
	}
	n, err := io.ReadFull(r.src, r.cbuf)
	switch {
	case errors.Is(err, io.ErrUnexpectedEOF):
		return r.cbuf[:n], true, nil
	case err != nil:
		return nil, false, sourceError(err)
	}
	if _, err := r.peek.Peek(1); err != nil {
		if errors.Is(err, io.EOF) {
			return r.cbuf, true, nil
		}
		return nil, false, err
	}
	return r.cbuf, false, nil
}

// sourceError maps a source that ended where a segment had to be to ErrOpen, and passes other errors on.
func sourceError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrOpen
	}
	return err
}
