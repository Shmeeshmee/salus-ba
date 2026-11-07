package envelope_test

import (
	"bytes"
	"errors"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/envelope"
)

const (
	segment    = envelope.SegmentSize
	cipherSeg  = segment + 16
	fileID     = "0b6f2d1e-6a3c-4f7e-9c51-2f7d4b1a8e90"
	otherID    = "5d0c9a7b-1e2f-4a3b-8c4d-6e5f7a8b9c0d"
	severalMiB = 5<<20 + 12345
)

// plaintext is synthetic, deterministic noise of n bytes.
func plaintext(n int) []byte {
	p := make([]byte, n)
	r := rand.NewChaCha8([32]byte{1, 2, 3})
	_, _ = r.Read(p)
	return p
}

func dataKey(t testing.TB) []byte {
	t.Helper()
	k, err := envelope.NewDataKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func seal(t testing.TB, dek []byte, id string, p []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w, err := envelope.NewWriter(&out, dek, id)
	if err != nil {
		t.Fatal(err)
	}
	// Uneven writes, so segment boundaries fall inside writes.
	for rest := p; len(rest) > 0; {
		n := min(len(rest), 10007)
		if _, err := w.Write(rest[:n]); err != nil {
			t.Fatal(err)
		}
		rest = rest[n:]
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func open(dek []byte, id string, c []byte) ([]byte, error) {
	r, err := envelope.Open(bytes.NewReader(c), dek, id)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

var sizes = []int{0, 1, segment - 1, segment, segment + 1, 2 * segment, severalMiB}

func TestNewDataKey_Returns32RandomBytes(t *testing.T) {
	a, b := dataKey(t), dataKey(t)
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("key sizes %d, %d, want 32", len(a), len(b))
	}
	if bytes.Equal(a, b) {
		t.Fatal("two data keys are equal")
	}
}

func TestWriterAndOpen_RoundTrip(t *testing.T) {
	for _, n := range sizes {
		dek := dataKey(t)
		p := plaintext(n)
		c := seal(t, dek, fileID, p)
		if int64(len(c)) != envelope.CiphertextSize(int64(n)) {
			t.Errorf("%d bytes: ciphertext %d bytes, CiphertextSize says %d", n, len(c), envelope.CiphertextSize(int64(n)))
		}
		got, err := open(dek, fileID, c)
		if err != nil {
			t.Fatalf("%d bytes: open: %v", n, err)
		}
		if !bytes.Equal(got, p) {
			t.Fatalf("%d bytes: the opened plaintext differs", n)
		}
	}
}

func TestWriter_DoesNotWritePlaintext(t *testing.T) {
	p := bytes.Repeat([]byte("synthetic marker "), 10000)
	c := seal(t, dataKey(t), fileID, p)
	if bytes.Contains(c, []byte("synthetic marker")) {
		t.Fatal("the ciphertext holds the plaintext")
	}
}

func TestCiphertextSize(t *testing.T) {
	cases := map[int64]int64{
		0:               16,
		1:               17,
		segment:         cipherSeg,
		segment + 1:     cipherSeg + 17,
		200_000_000:     200_000_000 + 16*3052, // ceil(200e6 / 65536) = 3052 segments
		3 * segment:     3 * cipherSeg,
		3*segment - 100: 3*cipherSeg - 100,
	}
	for n, want := range cases {
		if got := envelope.CiphertextSize(n); got != want {
			t.Errorf("CiphertextSize(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestWriter_WriteAfterCloseFails(t *testing.T) {
	w, err := envelope.NewWriter(io.Discard, dataKey(t), fileID)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("x")); err == nil {
		t.Fatal("a write after Close succeeded")
	}
	if err := w.Close(); err == nil {
		t.Fatal("a second Close succeeded")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk gone") }

func TestWriter_ReturnsTheDestinationsError(t *testing.T) {
	w, err := envelope.NewWriter(failingWriter{}, dataKey(t), fileID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(plaintext(2*segment + 1)); err == nil {
		t.Fatal("a write to a failing destination succeeded")
	}
	if _, err := w.Write([]byte("x")); err == nil {
		t.Fatal("a write after a failure succeeded")
	}
	w2, _ := envelope.NewWriter(failingWriter{}, dataKey(t), fileID)
	if err := w2.Close(); err == nil {
		t.Fatal("Close to a failing destination succeeded")
	}
}

func TestKeySizeIsChecked(t *testing.T) {
	short := make([]byte, 16)
	if _, err := envelope.NewWriter(io.Discard, short, fileID); err == nil {
		t.Error("NewWriter took a 16-byte data key")
	}
	if _, err := envelope.Open(bytes.NewReader(nil), short, fileID); err == nil {
		t.Error("Open took a 16-byte data key")
	}
	if _, err := envelope.OpenRange(short, fileID, 10, 0, 1); err == nil {
		t.Error("OpenRange took a 16-byte data key")
	}
	if _, err := envelope.NewKEK("local-1", short); err == nil {
		t.Error("NewKEK took a 16-byte key")
	}
	kek, err := envelope.NewKEK("local-1", dataKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kek.Wrap(short, fileID); err == nil {
		t.Error("Wrap took a 16-byte data key")
	}
}

func TestOpen_FailsOnTampering(t *testing.T) {
	dek := dataKey(t)
	p := plaintext(3*segment + 500)
	c := seal(t, dek, fileID, p)
	other := seal(t, dek, otherID, p)
	full := seal(t, dek, fileID, plaintext(2*segment))

	flipped := bytes.Clone(c)
	flipped[cipherSeg+100] ^= 0x01

	swapped := bytes.Clone(c)
	copy(swapped[:cipherSeg], c[cipherSeg:2*cipherSeg])
	copy(swapped[cipherSeg:2*cipherSeg], c[:cipherSeg])

	foreign := bytes.Clone(c)
	copy(foreign[cipherSeg:2*cipherSeg], other[cipherSeg:2*cipherSeg])

	cases := map[string][]byte{
		"a flipped bit":                     flipped,
		"two segments swapped":              swapped,
		"a segment from another file":       foreign,
		"truncated inside the last segment": c[:len(c)-1],
		"truncated at a segment boundary":   c[:3*cipherSeg],
		"full segments truncated":           full[:cipherSeg],
		"extended by a byte":                append(bytes.Clone(c), 0),
		"extended by a segment":             append(bytes.Clone(c), c[:cipherSeg]...),
		"extended after a full last":        append(bytes.Clone(full), full[:cipherSeg]...),
		"empty":                             nil,
		"shorter than a tag":                c[:10],
	}
	for name, tampered := range cases {
		if _, err := open(dek, fileID, tampered); !errors.Is(err, envelope.ErrOpen) {
			t.Errorf("%s: err = %v, want ErrOpen", name, err)
		}
	}
	if _, err := open(dek, otherID, c); !errors.Is(err, envelope.ErrOpen) {
		t.Errorf("opened with another id: err = %v, want ErrOpen", err)
	}
	if _, err := open(dataKey(t), fileID, c); !errors.Is(err, envelope.ErrOpen) {
		t.Errorf("opened with another data key: err = %v, want ErrOpen", err)
	}
}

type failingReader struct{ after []byte }

func (r *failingReader) Read(p []byte) (int, error) {
	if len(r.after) == 0 {
		return 0, errors.New("connection reset")
	}
	n := copy(p, r.after)
	r.after = r.after[n:]
	return n, nil
}

func TestOpen_ReturnsTheSourcesError(t *testing.T) {
	dek := dataKey(t)
	c := seal(t, dek, fileID, plaintext(3*segment))
	r, err := envelope.Open(&failingReader{after: c[:cipherSeg+5]}, dek, fileID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(r)
	if err == nil || errors.Is(err, envelope.ErrOpen) {
		t.Fatalf("err = %v, want the source's error", err)
	}
}

func TestKEK_WrapUnwrap(t *testing.T) {
	kek, err := envelope.NewKEK("local-1", dataKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if kek.ID() != "local-1" {
		t.Fatalf("ID() = %q", kek.ID())
	}
	dek := dataKey(t)
	wrapped, err := kek.Wrap(dek, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wrapped, dek) {
		t.Fatal("the wrapped key holds the data key")
	}
	again, _ := kek.Wrap(dek, fileID)
	if bytes.Equal(wrapped, again) {
		t.Fatal("two wraps of one key are equal (nonce reused)")
	}
	got, err := kek.Unwrap(wrapped, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("the unwrapped key differs")
	}

	otherKEK, _ := envelope.NewKEK("local-2", dataKey(t))
	flipped := bytes.Clone(wrapped)
	flipped[len(flipped)-1] ^= 1
	cases := map[string]func() ([]byte, error){
		"another KEK":   func() ([]byte, error) { return otherKEK.Unwrap(wrapped, fileID) },
		"another id":    func() ([]byte, error) { return kek.Unwrap(wrapped, otherID) },
		"a flipped bit": func() ([]byte, error) { return kek.Unwrap(flipped, fileID) },
		"truncated":     func() ([]byte, error) { return kek.Unwrap(wrapped[:5], fileID) },
	}
	for name, unwrap := range cases {
		if _, err := unwrap(); !errors.Is(err, envelope.ErrUnwrap) {
			t.Errorf("%s: err = %v, want ErrUnwrap", name, err)
		}
	}
}
