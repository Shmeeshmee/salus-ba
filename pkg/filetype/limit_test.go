package filetype

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestLimitReader(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 100) // 1,000 bytes
	tests := []struct {
		wantErr error
		name    string
		n       int64
	}{
		{nil, "exactly the declared length", 1000},
		{ErrShort, "fewer bytes than declared", 1001},
		{ErrTooLarge, "more bytes than declared", 999},
		{ErrTooLarge, "any byte past a zero limit", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// OneByteReader and the 7-byte copy buffer make every boundary fall in the middle of a read.
			r := LimitReader(iotest.OneByteReader(bytes.NewReader(data)), tt.n)
			var got bytes.Buffer
			_, err := io.CopyBuffer(&got, struct{ io.Reader }{r}, make([]byte, 7))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if int64(got.Len()) > tt.n {
				t.Fatalf("read %d bytes past a limit of %d", got.Len(), tt.n)
			}
			if tt.wantErr == nil && !bytes.Equal(got.Bytes(), data) {
				t.Fatal("the bytes changed on the way through")
			}
		})
	}
}

func TestLimitReader_EmptyAtZero(t *testing.T) {
	b, err := io.ReadAll(LimitReader(bytes.NewReader(nil), 0))
	if err != nil || len(b) != 0 {
		t.Fatalf("ReadAll = %d bytes, %v", len(b), err)
	}
}

func TestLimitReader_PassesOtherErrorsOn(t *testing.T) {
	boom := errors.New("connection reset")
	_, err := io.ReadAll(LimitReader(iotest.ErrReader(boom), 10))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	_, err = io.ReadAll(LimitReader(io.MultiReader(bytes.NewReader([]byte("abc")), iotest.ErrReader(boom)), 3))
	if !errors.Is(err, boom) {
		t.Fatalf("after the limit: err = %v, want %v", err, boom)
	}
}
