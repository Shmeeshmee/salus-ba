package filetype

import (
	"errors"
	"io"
)

// LimitReader reads exactly n bytes from r: it fails with ErrShort when r ends before n bytes, and with
// ErrTooLarge when r has more, without returning any byte past n.
func LimitReader(r io.Reader, n int64) io.Reader {
	return &limitReader{r: r, left: n}
}

type limitReader struct {
	r    io.Reader
	left int64
}

func (l *limitReader) Read(p []byte) (int, error) {
	if l.left <= 0 {
		return 0, l.end()
	}
	if int64(len(p)) > l.left {
		p = p[:l.left]
	}
	n, err := l.r.Read(p)
	l.left -= int64(n)
	if errors.Is(err, io.EOF) && l.left > 0 {
		return n, ErrShort
	}
	return n, err
}

// end checks, once n bytes are read, that r has nothing more.
func (l *limitReader) end() error {
	var b [1]byte
	n, err := io.ReadFull(l.r, b[:])
	switch {
	case n > 0:
		return ErrTooLarge
	case errors.Is(err, io.EOF):
		return io.EOF
	}
	return err
}
