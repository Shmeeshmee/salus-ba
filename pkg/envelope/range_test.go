package envelope_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// fetch plays the object store's ranged GetObject: the inclusive ciphertext bytes the range names.
func fetch(c []byte, r envelope.Range) io.Reader {
	return bytes.NewReader(c[r.Start : r.End+1])
}

func openRange(t *testing.T, dek []byte, c []byte, size, from, to int64) ([]byte, envelope.Range, error) {
	t.Helper()
	r, err := envelope.OpenRange(dek, fileID, size, from, to)
	if err != nil {
		t.Fatalf("OpenRange(%d, %d, %d): %v", size, from, to, err)
	}
	got, err := io.ReadAll(r.Reader(fetch(c, r)))
	return got, r, err
}

func TestOpenRange_YieldsExactlyTheRequestedBytes(t *testing.T) {
	for _, size := range []int64{1, segment, 2 * segment, 3*segment + 500} {
		dek := dataKey(t)
		p := plaintext(int(size))
		c := seal(t, dek, fileID, p)
		var points []int64
		for _, b := range []int64{0, 1, 100, segment - 1, segment, segment + 1, 2*segment - 1, 2 * segment,
			2*segment + 1, 3*segment - 1, 3 * segment, 3*segment + 1, size - 2, size - 1} {
			if b >= 0 && b < size {
				points = append(points, b)
			}
		}
		for _, from := range points {
			for _, to := range points {
				if from > to {
					continue
				}
				got, r, err := openRange(t, dek, c, size, from, to)
				if err != nil {
					t.Fatalf("size %d, %d-%d: %v", size, from, to, err)
				}
				if !bytes.Equal(got, p[from:to+1]) {
					t.Fatalf("size %d, %d-%d: got %d bytes, not the requested plaintext", size, from, to, len(got))
				}
				wantStart := from / segment * cipherSeg
				wantEnd := min((to/segment+1)*cipherSeg, envelope.CiphertextSize(size)) - 1
				if r.Start != wantStart || r.End != wantEnd {
					t.Fatalf("size %d, %d-%d: ciphertext %d-%d, want %d-%d", size, from, to, r.Start, r.End, wantStart, wantEnd)
				}
			}
		}
	}
}

func TestOpenRange_ReadsInSmallSteps(t *testing.T) {
	dek := dataKey(t)
	p := plaintext(3*segment + 500)
	c := seal(t, dek, fileID, p)
	r, err := envelope.OpenRange(dek, fileID, int64(len(p)), 10, int64(len(p))-10)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(io.LimitReader(onebyte{r.Reader(fetch(c, r))}, 1<<30))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, p[10:len(p)-9]) {
		t.Fatal("small reads yield other bytes")
	}
}

// onebyte reads through a 7-byte buffer, so reads end inside segments.
type onebyte struct{ r io.Reader }

func (o onebyte) Read(p []byte) (int, error) { return o.r.Read(p[:min(len(p), 7)]) }

func TestOpenRange_RefusesARangeOutsideTheFile(t *testing.T) {
	dek := dataKey(t)
	cases := map[string][3]int64{
		"from negative":     {100, -1, 10},
		"to past the end":   {100, 0, 100},
		"from after to":     {100, 50, 49},
		"an empty file":     {0, 0, 0},
		"from past the end": {100, 100, 100},
		"a negative size":   {-1, 0, 0},
	}
	for name, c := range cases {
		if _, err := envelope.OpenRange(dek, fileID, c[0], c[1], c[2]); !errors.Is(err, envelope.ErrRange) {
			t.Errorf("%s: err = %v, want ErrRange", name, err)
		}
	}
}

func TestOpenRange_FailsOnTampering(t *testing.T) {
	dek := dataKey(t)
	size := int64(3*segment + 500)
	p := plaintext(int(size))
	c := seal(t, dek, fileID, p)
	r, err := envelope.OpenRange(dek, fileID, size, segment+10, 2*segment+10)
	if err != nil {
		t.Fatal(err)
	}
	fetched := bytes.Clone(c[r.Start : r.End+1])
	flipped := bytes.Clone(fetched)
	flipped[cipherSeg+3] ^= 0x80
	other := seal(t, dek, otherID, p)
	cases := map[string][]byte{
		"a flipped bit":            flipped,
		"the wrong segments":       c[:2*cipherSeg],
		"another file's segments":  other[r.Start : r.End+1],
		"truncated":                fetched[:len(fetched)-1],
		"truncated to one segment": fetched[:cipherSeg],
		"nothing":                  nil,
	}
	for name, body := range cases {
		if _, err := io.ReadAll(r.Reader(bytes.NewReader(body))); !errors.Is(err, envelope.ErrOpen) {
			t.Errorf("%s: err = %v, want ErrOpen", name, err)
		}
	}

	// The file's last segment opens only as the last one: a larger claimed size reads it as a middle one.
	last, err := envelope.OpenRange(dek, fileID, size+segment, size-10, size-1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(last.Reader(bytes.NewReader(c[3*cipherSeg:]))); !errors.Is(err, envelope.ErrOpen) {
		t.Errorf("the last segment opened as a middle one: err = %v, want ErrOpen", err)
	}
}
