package clamd_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clamd"
)

// client returns a client for the compose clamd at TEST_CLAMD_ADDR. Without it the test fails; it never skips.
func client(t *testing.T) *clamd.Client {
	t.Helper()
	addr := os.Getenv("TEST_CLAMD_ADDR")
	if addr == "" {
		t.Fatal("TEST_CLAMD_ADDR is not set; run the tests through the tools service, which points it at the compose clamd")
	}
	return clamd.New(addr)
}

// testContext bounds a test's calls, so a hung clamd fails the test instead of the run.
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// eicar assembles the EICAR anti-malware test string, so the repository never holds it as a file.
func eicar() []byte {
	parts := []string{`X5O!P%@AP[4\PZX54(P^)7CC)7}`, `$EICAR-STANDARD-ANTIVIRUS`, `-TEST-FILE!$H+H*`}
	return []byte(strings.Join(parts, ""))
}

// unreachableAddr is a local TCP address nothing listens on.
func unreachableAddr(t *testing.T) string {
	t.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestPing_ClamdAnswers(t *testing.T) {
	if err := client(t).Ping(testContext(t)); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestVersion_NamesClamAV(t *testing.T) {
	v, err := client(t).Version(testContext(t))
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if !strings.HasPrefix(v, "ClamAV ") {
		t.Errorf("Version = %q, want it to start with %q", v, "ClamAV ")
	}
}

func TestScan_FindsEICAR(t *testing.T) {
	res, err := client(t).Scan(testContext(t), bytes.NewReader(eicar()))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !res.Found {
		t.Fatal("Scan of the EICAR string: not found")
	}
	if !strings.Contains(res.Signature, "Eicar") {
		t.Errorf("Signature = %q, want an EICAR signature", res.Signature)
	}
}

func TestScan_CleanBytesAreClean(t *testing.T) {
	// Synthetic bytes spanning several chunks, so the chunking is exercised too.
	clean := bytes.Repeat([]byte("synthetic clean bytes, no patient data\n"), 10_000)
	res, err := client(t).Scan(testContext(t), bytes.NewReader(clean))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Found || res.Signature != "" {
		t.Errorf("Scan of clean bytes = %+v, want clean", res)
	}
}

func TestScan_EmptyStreamIsClean(t *testing.T) {
	res, err := client(t).Scan(testContext(t), bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Found {
		t.Errorf("Scan of an empty stream = %+v, want clean", res)
	}
}

// zeros is an endless stream of zero bytes.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestScan_StreamPastClamdsLimitReturnsTheLimitError(t *testing.T) {
	// An endless stream: clamd stops reading at its StreamMaxLength (250 MB in compose), so the test sends no
	// more than the limit plus what is in flight, whatever the limit is.
	_, err := client(t).Scan(testContext(t), zeros{})
	if !errors.Is(err, clamd.ErrStreamLimit) {
		t.Fatalf("Scan past the limit: err = %v, want ErrStreamLimit", err)
	}
}

// cancelAfterFirstRead cancels its context on the first read and then streams zeros forever.
type cancelAfterFirstRead struct {
	cancel context.CancelFunc
}

func (c cancelAfterFirstRead) Read(p []byte) (int, error) {
	c.cancel()
	return zeros{}.Read(p)
}

func TestScan_CancelledContextStopsTheScan(t *testing.T) {
	ctx, cancel := context.WithCancel(testContext(t))
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client(t).Scan(ctx, cancelAfterFirstRead{cancel: cancel})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Scan with a cancelled context: err = %v, want context.Canceled", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Scan did not stop within 30 s of its context being cancelled")
	}
}

func TestScan_AlreadyCancelledContextIsNotUnreachable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client(t).Scan(ctx, bytes.NewReader(eicar()))
	if !errors.Is(err, context.Canceled) || errors.Is(err, clamd.ErrUnreachable) {
		t.Fatalf("Scan with a cancelled context: err = %v, want context.Canceled only", err)
	}
}

// failingReader fails on its first read.
type failingReader struct{}

var errSource = errors.New("source failed")

func (failingReader) Read([]byte) (int, error) { return 0, errSource }

func TestScan_SourceErrorIsReturned(t *testing.T) {
	_, err := client(t).Scan(testContext(t), failingReader{})
	if !errors.Is(err, errSource) {
		t.Fatalf("Scan of a failing source: err = %v, want the source's error", err)
	}
	if errors.Is(err, clamd.ErrUnreachable) || errors.Is(err, clamd.ErrStreamLimit) {
		t.Fatalf("Scan of a failing source: err = %v, blamed on clamd", err)
	}
}

func TestUnreachableAddress_ReturnsTheUnreachableError(t *testing.T) {
	c := clamd.New(unreachableAddr(t))
	ctx := testContext(t)
	if err := c.Ping(ctx); !errors.Is(err, clamd.ErrUnreachable) {
		t.Errorf("Ping: err = %v, want ErrUnreachable", err)
	}
	if _, err := c.Version(ctx); !errors.Is(err, clamd.ErrUnreachable) {
		t.Errorf("Version: err = %v, want ErrUnreachable", err)
	}
	if _, err := c.Scan(ctx, bytes.NewReader(eicar())); !errors.Is(err, clamd.ErrUnreachable) {
		t.Errorf("Scan: err = %v, want ErrUnreachable", err)
	}
}
