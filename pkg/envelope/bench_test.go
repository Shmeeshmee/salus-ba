package envelope_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/envelope"
)

const benchSize = 16 << 20

func BenchmarkSeal(b *testing.B) {
	dek := dataKey(b)
	p := plaintext(benchSize)
	b.SetBytes(benchSize)
	for b.Loop() {
		w, _ := envelope.NewWriter(io.Discard, dek, fileID)
		if _, err := w.Write(p); err != nil {
			b.Fatal(err)
		}
		if err := w.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpen(b *testing.B) {
	dek := dataKey(b)
	c := seal(b, dek, fileID, plaintext(benchSize))
	b.SetBytes(benchSize)
	for b.Loop() {
		r, _ := envelope.Open(bytes.NewReader(c), dek, fileID)
		if _, err := io.Copy(io.Discard, r); err != nil {
			b.Fatal(err)
		}
	}
}

// window keeps only the ciphertext bytes in [from, to], so a 200 MB file is sealed without holding it.
type window struct {
	kept     []byte
	from, to int64
	at       int64
}

func (w *window) Write(p []byte) (int, error) {
	start, end := max(w.from, w.at), min(w.to+1, w.at+int64(len(p)))
	if start < end {
		w.kept = append(w.kept, p[start-w.at:end-w.at]...)
	}
	w.at += int64(len(p))
	return len(p), nil
}

// BenchmarkOpenRange reads 1 MiB from the middle of a 200 MB file, as a seek in a video does.
func BenchmarkOpenRange(b *testing.B) {
	const size, from, n = 200_000_000, 123_456_789, 1 << 20
	dek := dataKey(b)
	r, err := envelope.OpenRange(dek, fileID, size, from, from+n-1)
	if err != nil {
		b.Fatal(err)
	}
	win := &window{from: r.Start, to: r.End}
	w, _ := envelope.NewWriter(win, dek, fileID)
	chunk := plaintext(1 << 20)
	for left := int64(size); left > 0; left -= int64(len(chunk)) {
		if _, err := w.Write(chunk[:min(left, int64(len(chunk)))]); err != nil {
			b.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(n)
	for b.Loop() {
		r, err := envelope.OpenRange(dek, fileID, size, from, from+n-1)
		if err != nil {
			b.Fatal(err)
		}
		got, err := io.Copy(io.Discard, r.Reader(bytes.NewReader(win.kept)))
		if err != nil || got != n {
			b.Fatalf("read %d bytes: %v", got, err)
		}
	}
}
