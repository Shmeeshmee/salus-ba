//go:build loadtest

package api_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/metrics"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The readings of docs/attachments/SPEC.md §2 "Performance" for package 2.1 (make bench-binary-http): a
// 200 MB generated MP4 uploaded through the api's HTTP server on the real database and object store, its
// throughput and the process's peak heap while it runs (runtime/metrics, sampled every 10 ms), then 1,000
// random 1 MiB ranges of it, p50, p95 and p99. Not part of make test.
func TestBinaryHTTPReadings(t *testing.T) {
	const size = 200_000_000
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	srv := httptest.NewUnstartedServer(app.Handler)
	srv.Config.ReadTimeout, srv.Config.WriteTimeout = 15*time.Second, api.RequestTimeout+5*time.Second // cmd/api's
	srv.Start()
	defer srv.Close()
	m := newMatrix(t, app.Handler, owner, cfg.FHIRBaseURL)
	r := request(t, m, m.future, publishQuestionnaire(t, m.practitioner,
		`{"title":"Placeholder","item":[{"linkId":"v1","text":"Placeholder video","type":"attachment","accept":["video"],"maxFiles":1}]}`))
	do := func(method, path string, body io.Reader, n int64, header map[string]string) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, body)
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = n
		req.Header.Set("Authorization", "Bearer "+m.patient.bearer)
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	head := generatedHead(size)
	runtime.GC()
	base := heap()
	var peak atomic.Uint64
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(10 * time.Millisecond):
				if h := heap(); h > peak.Load() {
					peak.Store(h)
				}
			}
		}
	}()
	clk := clock.System{}
	start := clk.Now()
	resp := do(http.MethodPost, "/questionnaire-responses/"+r.ID+"/items/v1/attachments",
		io.MultiReader(bytes.NewReader(head), io.LimitReader(&pattern{}, size-int64(len(head)))), size, nil)
	took := clk.Now().Sub(start)
	close(stop)
	var file struct{ ID string }
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("the upload = %d %s", resp.StatusCode, b)
	}
	_ = jsonUnmarshal(b, &file)
	defer func() { _ = do(http.MethodDelete, "/binaries/"+file.ID, nil, 0, nil).Body.Close() }() // and its object
	t.Logf("upload of %d bytes: %v, %.1f MB/s; heap %.1f MiB before, peak %.1f MiB (growth %.1f MiB)",
		size, took.Round(time.Millisecond), float64(size)/1e6/took.Seconds(), mib(base), mib(peak.Load()), mib(peak.Load())-mib(base))

	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE id = $1`, file.ID); err != nil {
		t.Fatal(err)
	}
	readRanges(t, do, file.ID, head, size)
}

// readRanges reads 1,000 random 1 MiB ranges of the file id of size bytes with do, checks each against the
// generated file and logs the latencies.
func readRanges(t *testing.T, do func(method, path string, body io.Reader, n int64, header map[string]string) *http.Response, id string, head []byte, size int64) {
	const rangeSize, ranges = 1 << 20, 1000
	clk := clock.System{}
	rng := rand.New(rand.NewPCG(1, 2))
	latencies := make([]time.Duration, 0, ranges)
	for range ranges {
		from := rng.Int64N(size - rangeSize)
		to := from + rangeSize - 1
		start := clk.Now()
		resp := do(http.MethodGet, "/binaries/"+id+"/content", nil, 0, map[string]string{"Range": "bytes=" + strconv.FormatInt(from, 10) + "-" + strconv.FormatInt(to, 10)})
		got, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		latencies = append(latencies, clk.Now().Sub(start))
		if err != nil || resp.StatusCode != http.StatusPartialContent || !bytes.Equal(got, expected(head, from, to)) {
			t.Fatalf("range %d-%d = %d, %d bytes, %v; not the file's", from, to, resp.StatusCode, len(got), err)
		}
	}
	slices.Sort(latencies)
	q := func(p float64) time.Duration {
		return latencies[int(p*float64(len(latencies)-1))].Round(10 * time.Microsecond)
	}
	t.Logf("%d random %d-byte ranges: p50 %v, p95 %v, p99 %v", ranges, rangeSize, q(0.5), q(0.95), q(0.99))
}

// heap is the bytes of live and unswept heap objects (runtime/metrics).
func heap() uint64 {
	s := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	metrics.Read(s)
	return s[0].Value.Uint64()
}

func mib(n uint64) float64 { return float64(n) / (1 << 20) }

// generatedHead is the ftyp, moov (an H.264 and an AAC track) and mdat header of a generated MP4 of size
// bytes; the media that follows is the pattern.
func generatedHead(size int64) []byte {
	box := func(typ string, parts ...[]byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, 0)
		b = append(b, typ...)
		for _, p := range parts {
			b = append(b, p...)
		}
		binary.BigEndian.PutUint32(b, uint32(len(b)))
		return b
	}
	trak := func(entry string) []byte {
		stsd := box("stsd", []byte{0, 0, 0, 0, 0, 0, 0, 1}, box(entry, make([]byte, 8)))
		return box("trak", box("tkhd", make([]byte, 84)), box("mdia", box("mdhd", make([]byte, 24)), box("hdlr", make([]byte, 25)),
			box("minf", box("stbl", stsd, box("stts", make([]byte, 8))))))
	}
	head := slices.Concat(box("ftyp", []byte("isom\x00\x00\x02\x00isomiso2avc1mp41")), box("moov", box("mvhd", make([]byte, 100)), trak("avc1"), trak("mp4a")))
	return slices.Concat(head, binary.BigEndian.AppendUint32(nil, uint32(size-int64(len(head)))), []byte("mdat"))
}

// pattern is the generated media: byte i is i × 7.
type pattern struct{ i int64 }

func (p *pattern) Read(b []byte) (int, error) {
	for j := range b {
		b[j] = byte(p.i * 7)
		p.i++
	}
	return len(b), nil
}

// expected is bytes from to to of the generated file.
func expected(head []byte, from, to int64) []byte {
	out := make([]byte, 0, to-from+1)
	for i := from; i <= to; i++ {
		if i < int64(len(head)) {
			out = append(out, head[i])
			continue
		}
		out = append(out, byte((i-int64(len(head)))*7))
	}
	return out
}

func jsonUnmarshal(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}
