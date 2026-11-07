package binary_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The compose object store (Garage, ADR 0010), reached from the tools container. Like the databases, the tests
// fail without it; they never skip. The bucket is shared with every other test running against the stack, so
// each test writes only under its own random prefix and deletes what it wrote.
func newStore(t *testing.T, pageSize int32) binary.Store {
	t.Helper()
	endpoint, region, bucket := os.Getenv("TEST_S3_ENDPOINT"), os.Getenv("TEST_S3_REGION"), os.Getenv("TEST_S3_BUCKET")
	keyID, secret := os.Getenv("TEST_S3_ACCESS_KEY_ID"), os.Getenv("TEST_S3_SECRET_ACCESS_KEY")
	if endpoint == "" || region == "" || bucket == "" || keyID == "" || secret == "" {
		t.Fatal("TEST_S3_ENDPOINT, TEST_S3_REGION, TEST_S3_BUCKET, TEST_S3_ACCESS_KEY_ID and TEST_S3_SECRET_ACCESS_KEY are not set; run the tests with `make test`, which points them at the compose object store")
	}
	return binary.NewS3Store(&http.Client{Timeout: 5 * time.Minute}, endpoint, region, bucket, keyID, secret, pageSize)
}

// prefix returns a key prefix unique to the test.
func prefix(t *testing.T) string {
	return "test/" + strings.ReplaceAll(t.Name(), "/", "_") + "-" + strings.ToLower(rand.Text()) + "/"
}

// put stores body under key and deletes it when the test ends.
func put(t *testing.T, s binary.Store, key string, body []byte) {
	t.Helper()
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })
	if err := s.Put(t.Context(), key, onlyReader{bytes.NewReader(body)}, int64(len(body))); err != nil {
		t.Fatalf("Put(%s) error = %v", key, err)
	}
}

// onlyReader hides every method but Read, so the store gets a body it cannot seek, as the upload's pipe is.
type onlyReader struct{ r io.Reader }

func (o onlyReader) Read(p []byte) (int, error) { return o.r.Read(p) }

func getRange(t *testing.T, s binary.Store, key string, from, to int64) []byte {
	t.Helper()
	rc, err := s.GetRange(t.Context(), key, from, to)
	if err != nil {
		t.Fatalf("GetRange(%d, %d) error = %v", from, to, err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("GetRange(%d, %d) read error = %v", from, to, err)
	}
	return b
}

func TestS3Store_PutThenGetRangesOfAStreamedBody(t *testing.T) {
	s := newStore(t, 1000)
	key := prefix(t) + "file"
	body := make([]byte, 3*65552+123)
	_, _ = rand.Read(body)
	put(t, s, key, body)

	last := int64(len(body) - 1)
	for _, c := range []struct {
		name     string
		from, to int64
	}{
		{"whole", 0, last},
		{"start", 0, 65551},
		{"middle", 65552, 2*65552 - 1},
		{"end", 3 * 65552, last},
		{"one byte", 100, 100},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := getRange(t, s, key, c.from, c.to); !bytes.Equal(got, body[c.from:c.to+1]) {
				t.Errorf("GetRange(%d, %d) = %d bytes, not the source's %d", c.from, c.to, len(got), c.to-c.from+1)
			}
		})
	}
}

func TestS3Store_GetRangeOfAMissingKeyIsErrObjectNotFound(t *testing.T) {
	s := newStore(t, 1000)
	_, err := s.GetRange(t.Context(), prefix(t)+"missing", 0, 10)
	if !errors.Is(err, binary.ErrObjectNotFound) {
		t.Fatalf("GetRange() error = %v, want ErrObjectNotFound", err)
	}
}

func TestS3Store_DeleteTwice(t *testing.T) {
	s := newStore(t, 1000)
	key := prefix(t) + "file"
	put(t, s, key, []byte("synthetic bytes"))
	for i := range 2 {
		if err := s.Delete(t.Context(), key); err != nil {
			t.Fatalf("Delete() #%d error = %v", i+1, err)
		}
	}
	if _, err := s.GetRange(t.Context(), key, 0, 1); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Fatalf("GetRange() after Delete error = %v, want ErrObjectNotFound", err)
	}
}

func TestS3Store_AShortBodyFailsAndLeavesNoObject(t *testing.T) {
	s := newStore(t, 1000)
	key := prefix(t) + "short"
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })
	err := s.Put(t.Context(), key, onlyReader{bytes.NewReader(make([]byte, 1000))}, 2000)
	if err == nil {
		t.Fatal("Put() of 1000 bytes declared as 2000 succeeded")
	}
	if _, err := s.GetRange(t.Context(), key, 0, 1); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Fatalf("GetRange() after the failed Put error = %v, want ErrObjectNotFound", err)
	}
}

// listOwn lists the bucket with before and returns the pages and this test's keys (the bucket holds other
// tests' keys too).
func listOwn(t *testing.T, s binary.Store, before time.Time, own string) (pages int, keys []string) {
	t.Helper()
	err := s.ListOlderThan(t.Context(), before, func(page []string) error {
		pages++
		if len(page) == 0 {
			t.Error("ListOlderThan() called fn with an empty page")
		}
		for _, k := range page {
			if strings.HasPrefix(k, own) {
				keys = append(keys, k)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ListOlderThan() error = %v", err)
	}
	slices.Sort(keys)
	return pages, keys
}

func TestS3Store_ListOlderThanPageByPageWithItsTimeFilter(t *testing.T) {
	s := newStore(t, 2)
	own := prefix(t)
	var want []string
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		put(t, s, own+n, []byte("synthetic "+n))
		want = append(want, own+n)
	}
	now := clock.System{}.Now()

	pages, got := listOwn(t, s, now.Add(time.Hour), own)
	if !slices.Equal(got, want) {
		t.Errorf("ListOlderThan(an hour from now) keys = %v, want %v", got, want)
	}
	if pages < 3 {
		t.Errorf("ListOlderThan() with a page of 2 called fn %d times for at least 5 keys, want 3 or more", pages)
	}

	if _, got := listOwn(t, s, now.Add(-time.Hour), own); len(got) != 0 {
		t.Errorf("ListOlderThan(an hour ago) keys = %v, want none of the new objects", got)
	}
}

func TestS3Store_ListOlderThanStopsAtTheFirstError(t *testing.T) {
	s := newStore(t, 1)
	own := prefix(t)
	put(t, s, own+"a", []byte("synthetic a"))
	put(t, s, own+"b", []byte("synthetic b"))
	stop := errors.New("stop")
	calls := 0
	err := s.ListOlderThan(t.Context(), clock.System{}.Now().Add(time.Hour), func([]string) error {
		calls++
		return stop
	})
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("ListOlderThan() = %v after %d calls, want fn's error after 1", err, calls)
	}
}

// generated yields size bytes of a repeating pattern without holding them.
type generated struct{ left int64 }

func (g *generated) Read(p []byte) (int, error) {
	if g.left == 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), g.left)
	for i := range n {
		p[i] = byte(g.left - i)
	}
	g.left -= n
	return int(n), nil
}

// A 200 MB put keeps the heap flat: sampled every 10 ms during the put, the live heap grows by far less than
// the file (the readings are in docs/attachments/PROGRESS.md, package 1.4).
func TestS3Store_A200MBPutKeepsTheHeapFlat(t *testing.T) {
	const size = 200 << 20
	s := newStore(t, 1000)
	key := prefix(t) + "large"
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	base := m.HeapAlloc

	done := make(chan struct{})
	var peak uint64
	var wg sync.WaitGroup
	wg.Go(func() {
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				peak = max(peak, m.HeapAlloc)
			}
		}
	})
	err := s.Put(t.Context(), key, &generated{left: size}, size)
	close(done)
	wg.Wait()
	if err != nil {
		t.Fatalf("Put() of 200 MB error = %v", err)
	}
	growth := int64(peak) - int64(base)
	t.Logf("heap before %d KiB, peak during the put %d KiB, growth %d KiB", base>>10, peak>>10, growth>>10)
	if growth > 16<<20 {
		t.Errorf("the heap grew by %d MiB during a 200 MB put, want under 16 MiB", growth>>20)
	}

	got := getRange(t, s, key, size-4, size-1)
	if want := []byte{4, 3, 2, 1}; !bytes.Equal(got, want) {
		t.Errorf("the last 4 bytes = %v, want %v", got, want)
	}
}

// An io.ReadCloser body (the upload's pipe) went out without its length and Garage refused it
// (InvalidRequest); the put now hides every method of the body but Read.
func TestS3Store_PutsABodyThatIsAReadCloser(t *testing.T) {
	s := newStore(t, 1000)
	key := prefix(t) + "file"
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })
	body := make([]byte, 65552+7)
	_, _ = rand.Read(body)
	pr, pw := io.Pipe()
	go func() { _, _ = pw.Write(body); _ = pw.Close() }()
	if err := s.Put(t.Context(), key, pr, int64(len(body))); err != nil {
		t.Fatalf("Put() of a pipe = %v", err)
	}
	if got := getRange(t, s, key, 0, int64(len(body)-1)); !bytes.Equal(got, body) {
		t.Errorf("GetRange() = %d bytes, not the %d put", len(got), len(body))
	}
}
