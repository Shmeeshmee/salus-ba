// Package azure is an stt.Recognizer on Azure AI Speech's real-time speech-to-text, over its WebSocket protocol
// (the one the Speech SDKs speak): one connection per stream, continuous ("conversation") recognition, 16 kHz
// 16-bit mono PCM in, hypotheses and phrases out. No SDK and no cgo: the protocol is text messages of headers and
// a JSON body, and binary audio messages framed by a 2-byte header length.
//
// It connects only to an EU region's endpoint, wss://<region>.stt.speech.microsoft.com, named by Config.Region;
// there is no endpoint override. It never asks the service to store audio (no storeAudio), and it logs nothing:
// its errors name a status or a step, never a key, an audio byte or recognised text.
package azure

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/stt"
)

var (
	// ErrConfig is returned by New for a missing key, a missing dialer, or a region that is not an EU one.
	ErrConfig = errors.New("azure: invalid configuration")
	// ErrUnsupported is returned by Stream for a language or a sample rate the recognizer does not take.
	ErrUnsupported = errors.New("azure: unsupported stream")
	// ErrService is returned when the service refuses the connection, reports a failure, sends a message that
	// cannot be read, or drops the connection.
	ErrService = errors.New("azure: speech service failed")
)

// euRegions are the Azure regions in the EU/EEA a recognizer may connect to: patient audio stays in the EU
// (CLAUDE.md "Residency and processors").
var euRegions = map[string]bool{
	"westeurope": true, "northeurope": true, "germanywestcentral": true, "francecentral": true,
	"swedencentral": true, "italynorth": true, "polandcentral": true, "spaincentral": true, "norwayeast": true,
}

// locales are the spoken languages a stream takes: the product's languages, each in the locale its speakers
// use (docs/plans/consult-intelligence.md, SAL-4).
var locales = map[string]bool{
	"bs-BA": true, "hr-HR": true, "sr-RS": true, "en-GB": true, "de-DE": true, "ar-SA": true, "tr-TR": true,
	"it-IT": true, "fr-FR": true,
}

// sampleRate is the only rate a stream takes: 16 kHz, the service's PCM rate.
const sampleRate = 16000

// Config is the service's settings: an EU region and the Speech resource's key.
type Config struct {
	Region string
	Key    string
}

// NewDialer is the dialer New takes in production: the system's TLS roots, a handshake bounded by timeout.
func NewDialer(timeout time.Duration) *websocket.Dialer {
	return &websocket.Dialer{HandshakeTimeout: timeout}
}

// Recognizer opens Azure recognition streams.
type Recognizer struct {
	clock    clock.Clock
	dialer   *websocket.Dialer
	endpoint string
	key      string
}

// New returns a Recognizer for cfg that connects with dialer and stamps its messages by clk. It refuses a region
// outside the EU, a missing key and a missing dialer; the error names the setting, never its value.
func New(cfg Config, dialer *websocket.Dialer, clk clock.Clock) (*Recognizer, error) {
	switch {
	case !euRegions[cfg.Region]:
		return nil, fmt.Errorf("%w: the region is not an EU region", ErrConfig)
	case cfg.Key == "":
		return nil, fmt.Errorf("%w: the key is missing", ErrConfig)
	case dialer == nil:
		return nil, fmt.Errorf("%w: the dialer is missing", ErrConfig)
	}
	return &Recognizer{
		clock:    clk,
		dialer:   dialer,
		endpoint: "wss://" + cfg.Region + ".stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1",
		key:      cfg.Key,
	}, nil
}

// Stream connects a recognition of cfg.Lang (a supported locale) at 16 kHz and sends the speech configuration
// and the WAV header; the stream ends with ctx.
func (r *Recognizer) Stream(ctx context.Context, cfg stt.Config) (stt.Stream, error) {
	if !locales[cfg.Lang] || cfg.SampleRate != sampleRate {
		return nil, ErrUnsupported
	}
	q := url.Values{"language": {cfg.Lang}, "format": {"simple"}}
	header := http.Header{"Ocp-Apim-Subscription-Key": {r.key}, "X-ConnectionId": {newID()}}
	conn, resp, err := r.dialer.DialContext(ctx, r.endpoint+"?"+q.Encode(), header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("%w: connect: status %d", ErrService, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: connect: %w", ErrService, err)
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &stream{conn: conn, ctx: ctx, cancel: cancel, clock: r.clock, lang: cfg.Lang, requestID: newID(), results: make(chan result, resultBuffer)}
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	if err := s.start(); err != nil {
		cancel()
		return nil, err
	}
	go s.read()
	return s, nil
}

// resultBuffer is how many results wait for Recv before the reader blocks.
const resultBuffer = 64

// stream is one connection's recognition.
type stream struct {
	ctx       context.Context
	clock     clock.Clock
	conn      *websocket.Conn
	cancel    context.CancelFunc
	results   chan result
	lang      string
	requestID string
	mu        sync.Mutex // one writer at a time
	isDone    bool
}

// result is what the reader hands Recv: a result, or the stream's end (err).
type result struct {
	err error
	res stt.Result
}

// speechConfig is the speech.config message's body: the client's description, nothing about anyone in the call.
const speechConfig = `{"context":{"system":{"name":"salus","version":"1"},"os":{"platform":"linux","name":"salus","version":"1"}}}`

// start sends the speech configuration and the WAV header of the stream's PCM.
func (s *stream) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := s.headers("speech.config", "application/json") + "\r\n" + speechConfig
	if err := s.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return fmt.Errorf("%w: send the configuration", ErrService)
	}
	if err := s.writeAudio(wavHeader(), true); err != nil {
		return fmt.Errorf("%w: send the audio header", ErrService)
	}
	return nil
}

// Send sends a frame of PCM.
func (s *stream) Send(pcm []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isDone {
		return stt.ErrClosed
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if err := s.writeAudio(pcm, false); err != nil {
		return fmt.Errorf("%w: send audio", ErrService)
	}
	return nil
}

// CloseSend sends the end of the audio: an empty audio message.
func (s *stream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isDone {
		return stt.ErrClosed
	}
	s.isDone = true
	if err := s.writeAudio(nil, false); err != nil {
		return fmt.Errorf("%w: send the end of the audio", ErrService)
	}
	return nil
}

// Recv returns the next result; io.EOF after the turn's end; the context's error once it is done. A failure
// closes the connection.
func (s *stream) Recv() (stt.Result, error) {
	select {
	case r := <-s.results:
		if r.err == nil {
			return r.res, nil
		}
		if err := s.ctx.Err(); err != nil { // the reader stops on the connection the context's end closed
			return stt.Result{}, err
		}
		s.cancel()
		return stt.Result{}, r.err
	case <-s.ctx.Done():
		return stt.Result{}, s.ctx.Err()
	}
}

// writeAudio writes a binary audio message; the first one (the WAV header) carries the content type.
func (s *stream) writeAudio(body []byte, first bool) error {
	contentType := ""
	if first {
		contentType = "audio/x-wav"
	}
	head := s.headers("audio", contentType)
	b := make([]byte, 2, 2+len(head)+len(body))
	binary.BigEndian.PutUint16(b, uint16(len(head))) //nolint:gosec // the headers are under 200 bytes
	b = append(append(b, head...), body...)
	return s.conn.WriteMessage(websocket.BinaryMessage, b)
}

// headers are a message's headers, each ended by CRLF.
func (s *stream) headers(path, contentType string) string {
	h := "Path: " + path + "\r\nX-RequestId: " + s.requestID + "\r\nX-Timestamp: " +
		s.clock.Now().UTC().Format("2006-01-02T15:04:05.000Z") + "\r\n"
	if contentType != "" {
		h += "Content-Type: " + contentType + "\r\n"
	}
	return h
}

// read reads the service's messages until the turn ends or the connection fails, and hands Recv the results.
func (s *stream) read() {
	for {
		kind, b, err := s.conn.ReadMessage()
		if err != nil {
			s.send(result{err: fmt.Errorf("%w: the connection closed", ErrService)})
			return
		}
		if kind != websocket.TextMessage {
			continue
		}
		res, keep, end, err := s.parse(b)
		switch {
		case err != nil:
			s.send(result{err: err})
			return
		case end:
			s.send(result{err: io.EOF})
			return
		case keep && !s.send(result{res: res}):
			return
		}
	}
}

// send hands r to Recv unless the stream's context is done.
func (s *stream) send(r result) bool {
	select {
	case s.results <- r:
		return true
	case <-s.ctx.Done():
		return false
	}
}

// tick is the service's time unit: offsets and durations are in 100 ns.
const tick = 100 * time.Nanosecond

// hypothesis is speech.hypothesis's body.
type hypothesis struct {
	Text     string `json:"Text"`
	Offset   int64  `json:"Offset"`
	Duration int64  `json:"Duration"`
}

// phrase is speech.phrase's body in the simple format.
type phrase struct {
	RecognitionStatus string `json:"RecognitionStatus"`
	DisplayText       string `json:"DisplayText"`
	Offset            int64  `json:"Offset"`
	Duration          int64  `json:"Duration"`
}

// parse reads a text message: a result to keep, the turn's end, or an error that names the path only.
func (s *stream) parse(b []byte) (res stt.Result, keep, end bool, err error) {
	head, body, hasBody := strings.Cut(string(b), "\r\n\r\n")
	path := pathOf(head)
	switch {
	case path == "turn.end":
		return stt.Result{}, false, true, nil
	case path != "speech.hypothesis" && path != "speech.phrase":
		return stt.Result{}, false, false, nil
	case !hasBody:
		return stt.Result{}, false, false, fmt.Errorf("%w: %s without a body", ErrService, path)
	}
	if path == "speech.hypothesis" {
		var h hypothesis
		if err := json.Unmarshal([]byte(body), &h); err != nil {
			return stt.Result{}, false, false, fmt.Errorf("%w: unreadable %s", ErrService, path)
		}
		return s.result(h.Text, h.Offset, h.Duration, false), h.Text != "", false, nil
	}
	var p phrase
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		return stt.Result{}, false, false, fmt.Errorf("%w: unreadable %s", ErrService, path)
	}
	switch p.RecognitionStatus {
	case "Success":
		return s.result(p.DisplayText, p.Offset, p.Duration, true), p.DisplayText != "", false, nil
	case "NoMatch", "InitialSilenceTimeout", "BabbleTimeout", "EndOfDictation":
		return stt.Result{}, false, false, nil
	}
	return stt.Result{}, false, false, fmt.Errorf("%w: recognition status %s", ErrService, safeStatus(p.RecognitionStatus))
}

func (s *stream) result(text string, offset, duration int64, final bool) stt.Result {
	start := time.Duration(offset) * tick
	return stt.Result{Text: text, Lang: s.lang, Start: start, End: start + time.Duration(duration)*tick, Final: final}
}

// pathOf is the Path header's value, lower case.
func pathOf(head string) string {
	for line := range strings.SplitSeq(head, "\r\n") {
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "path") {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

// safeStatus keeps a status that is one short word, so an error carries the service's code and nothing else.
func safeStatus(status string) string {
	if status == "" || len(status) > 32 || strings.Trim(status, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") != "" {
		return "unknown"
	}
	return status
}

// wavHeader is the 44-byte RIFF header of 16 kHz 16-bit mono PCM of unknown length, which the service expects
// before the audio.
func wavHeader() []byte {
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)           // the fmt chunk's size
	binary.LittleEndian.PutUint16(h[20:], 1)            // PCM
	binary.LittleEndian.PutUint16(h[22:], 1)            // mono
	binary.LittleEndian.PutUint32(h[24:], sampleRate)   // samples per second
	binary.LittleEndian.PutUint32(h[28:], sampleRate*2) // bytes per second
	binary.LittleEndian.PutUint16(h[32:], 2)            // block align
	binary.LittleEndian.PutUint16(h[34:], 16)           // bits per sample
	copy(h[36:], "data")
	return h
}

// newID is a request or connection id: 32 upper-case hex digits, as the service's own clients send.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
