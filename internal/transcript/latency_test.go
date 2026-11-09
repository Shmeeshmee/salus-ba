package transcript_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/azure"
)

// The partial caption's latency (SAL-4, target under 1 s): from an audio frame handed to the agent to its
// partial event read back from the encounter's stream in the compose Redis (XREAD BLOCK), one frame at a time.
// make bench-stt runs these; the numbers are in docs/plans/consult-intelligence.PROGRESS.md. The LiveKit hop
// before it (a microphone's frame to the agent's decoded frame) is BenchmarkListen_FrameLatency in videosession.

// echo is a recognizer that answers every frame of audio with a partial result at once: the pipeline's own
// latency, with an engine that takes no time.
type echo struct{}

func (echo) Stream(ctx context.Context, cfg stt.Config) (stt.Stream, error) {
	return &echoStream{ctx: ctx, results: make(chan stt.Result, 64)}, nil
}

type echoStream struct {
	ctx     context.Context
	results chan stt.Result
	n       time.Duration
}

func (s *echoStream) Send([]byte) error {
	s.n += 20 * time.Millisecond
	s.results <- stt.Result{Text: "placeholder", Start: 0, End: s.n}
	return nil
}

func (s *echoStream) CloseSend() error { return nil }

func (s *echoStream) Recv() (stt.Result, error) {
	select {
	case r := <-s.results:
		return r, nil
	case <-s.ctx.Done():
		return stt.Result{}, s.ctx.Err()
	}
}

func BenchmarkPartialLatency_FakeEngine(b *testing.B) { benchPartial(b, echo{}) }

// The Azure adapter against an httptest server that speaks its protocol and answers each audio message with a
// hypothesis: the adapter's framing, TLS and WebSocket on top of the pipeline, without Azure's own time.
func BenchmarkPartialLatency_AzureAdapter(b *testing.B) {
	srv := azureEcho(b)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	addr := srv.Listener.Addr().String()
	dialer := &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "example.com", MinVersion: tls.VersionTLS12},
	}
	r, err := azure.New(azure.Config{Region: "westeurope", Key: "synthetic-bench-key"}, dialer, clock.System{})
	if err != nil {
		b.Fatal(err)
	}
	benchPartial(b, r)
}

// azureEcho answers each audio message after the WAV header with a speech.hypothesis.
func azureEcho(b *testing.B) *httptest.Server {
	b.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		audio := 0
		for {
			kind, m, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind != websocket.BinaryMessage || len(m) <= 2+int(binary.BigEndian.Uint16(m)) {
				continue // the configuration, or an empty audio message
			}
			if audio++; audio == 1 {
				continue // the WAV header
			}
			msg := "X-RequestId: 0123456789ABCDEF0123456789ABCDEF\r\nPath: speech.hypothesis\r\n\r\n" +
				`{"Text":"placeholder","Offset":0,"Duration":200000}`
			if conn.WriteMessage(websocket.TextMessage, []byte(msg)) != nil {
				return
			}
		}
	}))
	b.Cleanup(srv.Close)
	return srv
}

// benchPartial runs b.N frames through the agent on r, one at a time, each timed from the frame's hand-over to
// its partial event read back from Redis, and reports p50, p95 and p99.
func benchPartial(b *testing.B, r stt.Recognizer) {
	rdb := redistest.NewClient(b)
	calls := newCalls()
	languages := &fakeLanguages{patient: "bs", clinic: "bs"}
	svc := transcript.NewLive(transcript.LivePorts{
		Calls: calls, Appointments: languages, Accounts: languages, Clinics: languages,
		Publisher: transcript.NewRedisPublisher(rdb), Recognizer: r, Clock: clock.System{},
		Log: slog.New(slog.DiscardHandler),
	})
	b.Cleanup(svc.Stop)
	if err := svc.Transcribe(b.Context(), sessionID, clinicID); err != nil {
		b.Fatal(err)
	}
	frames := calls.publish(videosession.RolePatient)
	frame := make([]byte, 640)
	key := transcript.StreamKey(encounterID)
	last := "0"
	wall := clock.System{}
	took := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for range b.N {
		start := wall.Now()
		frames <- frame
		res, err := rdb.XRead(b.Context(), &goredis.XReadArgs{Streams: []string{key, last}, Count: 1, Block: 5 * time.Second}).Result()
		if err != nil || len(res) == 0 || len(res[0].Messages) == 0 {
			b.Fatalf("no partial on the stream within 5 s: %v", err)
		}
		last = res[0].Messages[0].ID
		took = append(took, wall.Now().Sub(start))
	}
	b.StopTimer()
	slices.Sort(took)
	for _, p := range []struct {
		name string
		q    float64
	}{{"p50-ms", 0.50}, {"p95-ms", 0.95}, {"p99-ms", 0.99}} {
		b.ReportMetric(float64(took[int(p.q*float64(len(took)-1))].Microseconds())/1000, p.name)
	}
}
