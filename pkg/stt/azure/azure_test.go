package azure_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/azure"
)

var _ stt.Recognizer = (*azure.Recognizer)(nil)

// stamp is the time the recognizer's clock gives: every message carries it as its X-Timestamp.
var stamp = time.Date(2026, 11, 9, 9, 30, 0, 123_000_000, time.UTC)

// key is a synthetic subscription key; no real credential is used anywhere in these tests.
const key = "synthetic-test-key-0123456789"

// message is one WebSocket message the fake service received: its headers and its body.
type message struct {
	headers map[string]string
	body    []byte
	binary  bool
}

// service is an httptest TLS server that speaks Azure's speech WebSocket protocol: it records the request and
// every message, and answers with script once it has the WAV header (the first audio message).
type service struct {
	srv      *httptest.Server
	req      chan *http.Request
	msgs     chan message
	script   []string // text messages to send, in order, after the WAV header
	closeNow bool     // close the connection right after the script, without turn.end
}

func newService(t *testing.T, script ...string) *service {
	t.Helper()
	s := &service{req: make(chan *http.Request, 1), msgs: make(chan message, 1000), script: script}
	up := websocket.Upgrader{}
	s.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.req <- r
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		s.serve(conn)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *service) serve(conn *websocket.Conn) {
	scripted := false
	for {
		kind, b, err := conn.ReadMessage()
		if err != nil {
			return
		}
		m := parse(kind, b)
		s.msgs <- m
		if m.headers["path"] == "audio" && !scripted {
			scripted = true
			for _, text := range s.script {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(text))
			}
			if s.closeNow {
				return
			}
		}
	}
}

// parse reads a client message: text is headers, a blank line and a body; binary is a 2-byte big-endian header
// length, the headers and the body.
func parse(kind int, b []byte) message {
	m := message{headers: map[string]string{}, binary: kind == websocket.BinaryMessage}
	var head string
	if m.binary {
		n := int(binary.BigEndian.Uint16(b))
		head, m.body = string(b[2:2+n]), b[2+n:]
	} else {
		h, body, _ := strings.Cut(string(b), "\r\n\r\n")
		head, m.body = h, []byte(body)
	}
	for line := range strings.SplitSeq(head, "\r\n") {
		if name, value, ok := strings.Cut(line, ":"); ok {
			m.headers[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
		}
	}
	return m
}

// textMessage is a message of the service: the path's headers and a JSON body.
func textMessage(path, body string) string {
	return "X-RequestId: 0123456789ABCDEF0123456789ABCDEF\r\nContent-Type: application/json; charset=utf-8\r\nPath: " + path + "\r\n\r\n" + body
}

// dialer reaches the fake service whatever host the recognizer asks for, and trusts its certificate for the
// name example.com (httptest's), so the recognizer's own URL (an EU region's host) is what the test checks.
func (s *service) dialer() *websocket.Dialer {
	pool := x509.NewCertPool()
	pool.AddCert(s.srv.Certificate())
	addr := s.srv.Listener.Addr().String()
	return &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig:  &tls.Config{RootCAs: pool, ServerName: "example.com", MinVersion: tls.VersionTLS12},
		HandshakeTimeout: 5 * time.Second,
	}
}

func newRecognizer(t *testing.T, s *service) *azure.Recognizer {
	t.Helper()
	r, err := azure.New(azure.Config{Region: "westeurope", Key: key}, s.dialer(), clock.Fixed{T: stamp})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (s *service) next(t *testing.T) message {
	t.Helper()
	select {
	case m := <-s.msgs:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("the service received no message")
		return message{}
	}
}

func TestNew_RefusesARegionOutsideTheEUAndAMissingKey(t *testing.T) {
	t.Parallel()
	for name, cfg := range map[string]azure.Config{
		"no region":         {Key: key},
		"a US region":       {Region: "eastus", Key: key},
		"a Swiss region":    {Region: "switzerlandnorth", Key: key},
		"a region with dot": {Region: "westeurope.evil.example", Key: key},
		"no key":            {Region: "westeurope"},
	} {
		_, err := azure.New(cfg, &websocket.Dialer{}, clock.System{})
		if !errors.Is(err, azure.ErrConfig) {
			t.Errorf("%s: New() = %v, want ErrConfig", name, err)
			continue
		}
		if strings.Contains(err.Error(), key) || (cfg.Region != "" && strings.Contains(err.Error(), cfg.Region)) {
			t.Errorf("%s: the error %q holds a value", name, err)
		}
	}
	if _, err := azure.New(azure.Config{Region: "westeurope", Key: key}, nil, clock.System{}); !errors.Is(err, azure.ErrConfig) {
		t.Errorf("New() without a dialer = %v, want ErrConfig", err)
	}
}

func TestNew_AcceptsEveryEURegion(t *testing.T) {
	t.Parallel()
	for _, region := range []string{"westeurope", "northeurope", "germanywestcentral", "francecentral", "swedencentral"} {
		if _, err := azure.New(azure.Config{Region: region, Key: key}, &websocket.Dialer{}, clock.System{}); err != nil {
			t.Errorf("New(%s) = %v, want nil", region, err)
		}
	}
}

func TestStream_RefusesAnUnsupportedLanguageOrSampleRate(t *testing.T) {
	t.Parallel()
	s := newService(t)
	r := newRecognizer(t, s)
	for name, cfg := range map[string]stt.Config{
		"a language without a region": {Lang: "bs", SampleRate: 16000},
		"an unknown language":         {Lang: "xx-XX", SampleRate: 16000},
		"48 kHz":                      {Lang: "bs-BA", SampleRate: 48000},
	} {
		if _, err := r.Stream(t.Context(), cfg); !errors.Is(err, azure.ErrUnsupported) {
			t.Errorf("%s: Stream() = %v, want ErrUnsupported", name, err)
		}
	}
	select {
	case <-s.req:
		t.Error("an unsupported stream connected to the service")
	default:
	}
}

func TestStream_ConnectsToTheRegionsConversationEndpointWithTheKey(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"bs-BA", "de-DE", "ar-SA", "tr-TR"} {
		s := newService(t)
		st, err := newRecognizer(t, s).Stream(t.Context(), stt.Config{Lang: lang, SampleRate: 16000})
		if err != nil {
			t.Fatalf("%s: Stream() = %v", lang, err)
		}
		req := <-s.req
		if req.Host != "westeurope.stt.speech.microsoft.com" || req.URL.Path != "/speech/recognition/conversation/cognitiveservices/v1" {
			t.Errorf("%s: connected to %s%s", lang, req.Host, req.URL.Path)
		}
		if q := req.URL.Query(); q.Get("language") != lang || q.Get("format") != "simple" || q.Has("storeAudio") {
			t.Errorf("%s: query %v, want language=%s and format=simple, no storeAudio", lang, q, lang)
		}
		if req.Header.Get("Ocp-Apim-Subscription-Key") != key || len(req.Header.Get("X-ConnectionId")) != 32 {
			t.Errorf("%s: headers %v, want the key and a 32-hex connection id", lang, req.Header)
		}
		_ = st.CloseSend()
	}
}

func TestStream_SendsTheConfigTheWAVHeaderTheAudioAndTheEnd(t *testing.T) {
	t.Parallel()
	s := newService(t)
	st, err := newRecognizer(t, s).Stream(t.Context(), stt.Config{Lang: "bs-BA", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	cfg := s.next(t)
	if cfg.binary || cfg.headers["path"] != "speech.config" || cfg.headers["content-type"] != "application/json" || !strings.HasPrefix(string(cfg.body), "{") {
		t.Errorf("first message = %+v, want speech.config JSON", cfg)
	}
	header := s.next(t)
	if !header.binary || header.headers["path"] != "audio" || header.headers["content-type"] != "audio/x-wav" {
		t.Fatalf("second message headers = %v, want the WAV header's audio", header.headers)
	}
	h := header.body
	if len(h) != 44 || string(h[0:4]) != "RIFF" || string(h[8:12]) != "WAVE" ||
		binary.LittleEndian.Uint16(h[22:24]) != 1 || binary.LittleEndian.Uint32(h[24:28]) != 16000 || binary.LittleEndian.Uint16(h[34:36]) != 16 {
		t.Errorf("WAV header = %v, want 44 bytes of RIFF/WAVE, mono, 16 kHz, 16 bits", h)
	}
	if header.headers["x-requestid"] != cfg.headers["x-requestid"] || len(cfg.headers["x-requestid"]) != 32 || cfg.headers["x-timestamp"] != "2026-11-09T09:30:00.123Z" {
		t.Errorf("request ids %q and %q, timestamp %q: want one 32-hex id and a timestamp", cfg.headers["x-requestid"], header.headers["x-requestid"], cfg.headers["x-timestamp"])
	}

	frame := []byte{1, 2, 3, 4}
	if err := st.Send(frame); err != nil {
		t.Fatal(err)
	}
	if audio := s.next(t); !audio.binary || audio.headers["path"] != "audio" || string(audio.body) != string(frame) {
		t.Errorf("audio message = %+v, want the frame", audio)
	}
	if err := st.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if end := s.next(t); !end.binary || end.headers["path"] != "audio" || len(end.body) != 0 {
		t.Errorf("end message = %+v, want an empty audio message", end)
	}
	if err := st.Send(frame); !errors.Is(err, stt.ErrClosed) {
		t.Errorf("Send() after CloseSend = %v, want stt.ErrClosed", err)
	}
	if err := st.CloseSend(); !errors.Is(err, stt.ErrClosed) {
		t.Errorf("CloseSend() twice = %v, want stt.ErrClosed", err)
	}
}

func TestStream_GivesHypothesesAsPartialsAndPhrasesAsFinalsThenEOFAtTheTurnsEnd(t *testing.T) {
	t.Parallel()
	s := newService(t,
		textMessage("turn.start", `{"context":{"serviceTag":"x"}}`),
		textMessage("speech.startDetected", `{"Offset":5000000}`),
		textMessage("speech.hypothesis", `{"Text":"placeholder","Offset":5000000,"Duration":4000000}`),
		textMessage("speech.hypothesis", `{"Text":"placeholder words","Offset":5000000,"Duration":8000000}`),
		textMessage("speech.phrase", `{"RecognitionStatus":"Success","DisplayText":"Placeholder words.","Offset":5000000,"Duration":9000000}`),
		textMessage("speech.phrase", `{"RecognitionStatus":"InitialSilenceTimeout","Offset":0,"Duration":0}`),
		textMessage("speech.phrase", `{"RecognitionStatus":"NoMatch","Offset":20000000,"Duration":1000000}`),
		textMessage("speech.endDetected", `{"Offset":30000000}`),
		textMessage("turn.end", `{}`),
	)
	st, err := newRecognizer(t, s).Stream(t.Context(), stt.Config{Lang: "de-DE", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	want := []stt.Result{
		{Text: "placeholder", Lang: "de-DE", Start: 500 * time.Millisecond, End: 900 * time.Millisecond},
		{Text: "placeholder words", Lang: "de-DE", Start: 500 * time.Millisecond, End: 1300 * time.Millisecond},
		{Text: "Placeholder words.", Lang: "de-DE", Start: 500 * time.Millisecond, End: 1400 * time.Millisecond, Final: true},
	}
	for i, w := range want {
		got, err := st.Recv()
		if err != nil || got != w {
			t.Fatalf("Recv() #%d = %+v, %v; want %+v", i, got, err, w)
		}
	}
	if _, err := st.Recv(); !errors.Is(err, io.EOF) {
		t.Errorf("Recv() after turn.end = %v, want io.EOF", err)
	}
}

func TestStream_RecvEndsWithTheContext(t *testing.T) {
	t.Parallel()
	s := newService(t)
	ctx, cancel := context.WithCancel(t.Context())
	st, err := newRecognizer(t, s).Stream(ctx, stt.Config{Lang: "tr-TR", SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := st.Recv(); !errors.Is(err, context.Canceled) {
		t.Errorf("Recv() = %v, want context.Canceled", err)
	}
	if err := st.Send([]byte{1, 2}); err == nil {
		t.Error("Send() after the context ended = nil, want an error")
	}
}

// A failure from the service, a malformed message or a dropped connection is an error that names no text,
// no key and no audio.
func TestStream_ErrorsNameNoTextNorKey(t *testing.T) {
	t.Parallel()
	const secret = "placeholder secret words"
	for name, tc := range map[string]struct {
		script   []string
		closeNow bool
	}{
		"an error status": {script: []string{textMessage("speech.phrase", `{"RecognitionStatus":"Error","DisplayText":"`+secret+`"}`)}},
		"malformed JSON":  {script: []string{textMessage("speech.hypothesis", `{"Text":"`+secret+`",`)}},
		"no body":         {script: []string{"Path: speech.hypothesis"}},
		"a dropped call":  {script: []string{textMessage("speech.hypothesis", `{"Text":"`+secret+`","Offset":0,"Duration":1}`)}, closeNow: true},
	} {
		s := newService(t, tc.script...)
		s.closeNow = tc.closeNow
		st, err := newRecognizer(t, s).Stream(t.Context(), stt.Config{Lang: "ar-SA", SampleRate: 16000})
		if err != nil {
			t.Fatalf("%s: Stream() = %v", name, err)
		}
		var last error
		for range 3 {
			if _, last = st.Recv(); last != nil {
				break
			}
		}
		switch {
		case last == nil || errors.Is(last, io.EOF):
			t.Errorf("%s: Recv() = %v, want an error", name, last)
		case strings.Contains(last.Error(), secret) || strings.Contains(last.Error(), key):
			t.Errorf("%s: the error %q holds the text or the key", name, last)
		}
	}
}

func TestStream_ConnectFailureNamesNoKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied "+key, http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	s := &service{srv: srv}
	_, err := newRecognizer(t, s).Stream(t.Context(), stt.Config{Lang: "bs-BA", SampleRate: 16000})
	if err == nil || strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "401") {
		t.Errorf("Stream() = %v, want an error naming the status 401 and not the key", err)
	}
}
