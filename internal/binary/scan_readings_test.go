//go:build loadtest

package binary_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TestScanReadings scans a 200 MB generated MP4 (the video limit) through Scan against the compose store and
// clamd, three different files, and logs the time of each (docs/attachments/SPEC.md §2 "Performance"; clamd's
// resident memory is read beside it from /proc in its container). Run by hand: go test -tags loadtest -run
// TestScanReadings -v ./internal/binary/. A run whose file clamd has seen clean before hits clamd's cache.
func TestScanReadings(t *testing.T) {
	f := newFixture(t)
	const media = binary.MaxVideoSize - 1024
	head := mp4Head(media, "avc1")
	size := int64(len(head)) + media
	svc := f.scanning(scanner(t))
	for run := range 3 { // a different file each run: clamd caches the hashes of clean files
		f.repo.files, f.holder.attached = map[string]binary.Binary{}, nil
		b, _, err := f.svc.Upload(t.Context(), me, binary.Upload{
			Body: io.MultiReader(bytes.NewReader(head), io.LimitReader(counter{seed: byte(run)}, media)), ResponseID: responseA,
			LinkID: "files", Title: title, Size: size,
		})
		if err != nil {
			t.Fatalf("Upload() = %v", err)
		}
		start := clock.System{}.Now()
		got, err := svc.Scan(t.Context(), scanArgs(b))
		took := clock.System{}.Now().Sub(start)
		if err != nil || got.Status != binary.StatusClean {
			t.Fatalf("Scan() = %+v, %v", got, err)
		}
		t.Logf("run %d: %d bytes scanned in %v (%.0f MB/s)", run, size, took.Round(time.Millisecond), float64(size)/took.Seconds()/1e6)
	}
}

// counter yields a repeating counter, as the fixtures' media does.
type counter struct{ seed byte }

func (c counter) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i*7) + c.seed
	}
	return len(p), nil
}
