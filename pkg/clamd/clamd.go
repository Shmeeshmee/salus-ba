// Package clamd is a client for ClamAV's clamd over TCP: PING, VERSION and INSTREAM (ADR 0010).
//
// Every dial, read and write is bound to the caller's context. Errors never carry the scanned bytes; a
// Result's signature name is ClamAV's, not personal data, but callers keep it away from a patient id in logs.
package clamd

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

var (
	// ErrStreamLimit is returned when the stream is longer than clamd's StreamMaxLength.
	ErrStreamLimit = errors.New("clamd: stream longer than clamd's limit")
	// ErrUnreachable is returned when no connection to clamd could be made.
	ErrUnreachable = errors.New("clamd: unreachable")
)

const (
	chunkSize = 64 << 10 // bytes per INSTREAM chunk
	maxReply  = 4 << 10  // a clamd reply is one short line
)

// past is a deadline already gone: setting it fails every blocked read or write on a connection.
var past = time.Unix(1, 0)

// Result is the outcome of a scan: clean, or found with ClamAV's signature name.
type Result struct {
	Signature string
	Found     bool
}

// Client talks to the clamd at one TCP address. It holds no connection; each call dials its own.
type Client struct {
	addr string
}

// New returns a client for the clamd at addr (host:port).
func New(addr string) *Client {
	return &Client{addr: addr}
}

// Ping checks that clamd answers PONG.
func (c *Client) Ping(ctx context.Context) error {
	reply, err := c.command(ctx, "PING")
	if err != nil {
		return err
	}
	if reply != "PONG" {
		return fmt.Errorf("clamd: PING answered %q", reply)
	}
	return nil
}

// Version returns clamd's version line, such as "ClamAV 1.5.4/27800/Sat Nov  7 08:00:00 2026".
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.command(ctx, "VERSION")
}

// command sends one null-terminated command and returns its reply.
func (c *Client) command(ctx context.Context, name string) (string, error) {
	conn, done, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer done()
	if _, err := io.WriteString(conn, "z"+name+"\x00"); err != nil {
		return "", ioError(ctx, name, err)
	}
	reply, err := readReply(conn)
	if err != nil {
		return "", ioError(ctx, name, err)
	}
	return reply, nil
}

// Scan streams r to clamd with INSTREAM and returns its verdict. A stream past clamd's StreamMaxLength
// returns ErrStreamLimit; an error reading r is returned wrapped, and is not ErrUnreachable.
func (c *Client) Scan(ctx context.Context, r io.Reader) (Result, error) {
	conn, done, err := c.dial(ctx)
	if err != nil {
		return Result{}, err
	}
	defer done()

	// clamd may answer before the stream ends (the size limit) and then close the connection, which fails
	// the next write; reading alongside the writes keeps that answer.
	type answer struct {
		err   error
		reply string
	}
	answered := make(chan answer, 1)
	stop := make(chan struct{})
	go func() {
		reply, err := readReply(conn)
		if err == nil {
			_ = conn.SetWriteDeadline(past) // clamd has answered: stop any write still blocked
		}
		close(stop)
		answered <- answer{reply: reply, err: err}
	}()

	werr := writeStream(conn, r, stop)
	if werr != nil {
		_ = conn.SetReadDeadline(past) // clamd may still be waiting for the rest of the stream
	}
	a := <-answered
	switch {
	case a.err == nil:
		return parseScanReply(a.reply)
	case ctx.Err() != nil:
		return Result{}, fmt.Errorf("clamd: INSTREAM: %w", ctx.Err())
	case werr != nil:
		return Result{}, werr
	default:
		return Result{}, ioError(ctx, "INSTREAM", a.err)
	}
}

// writeStream sends the INSTREAM command, r in length-prefixed chunks and the zero-length terminator. It
// stops early once clamd has answered (stop closed).
func writeStream(conn net.Conn, r io.Reader, stop <-chan struct{}) error {
	w := bufio.NewWriterSize(conn, 4+chunkSize)
	if _, err := w.WriteString("zINSTREAM\x00"); err != nil {
		return fmt.Errorf("clamd: INSTREAM: %w", err)
	}
	buf := make([]byte, chunkSize)
	for {
		select {
		case <-stop:
			return errors.New("clamd: INSTREAM: answered before the end of the stream")
		default:
		}
		n, rerr := r.Read(buf)
		if n > 0 {
			if err := writeChunk(w, buf[:n]); err != nil {
				return err
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return fmt.Errorf("clamd: read the stream: %w", rerr)
		}
	}
	if err := writeChunk(w, nil); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("clamd: INSTREAM: %w", err)
	}
	return nil
}

// writeChunk writes one chunk: its length as a 4-byte big-endian integer, then the bytes.
func writeChunk(w *bufio.Writer, p []byte) error {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(p))) // len(p) <= chunkSize
	if _, err := w.Write(size[:]); err != nil {
		return fmt.Errorf("clamd: INSTREAM: %w", err)
	}
	if _, err := w.Write(p); err != nil {
		return fmt.Errorf("clamd: INSTREAM: %w", err)
	}
	return nil
}

// parseScanReply maps INSTREAM's reply: "stream: OK", "stream: <signature> FOUND" or "<reason> ERROR".
func parseScanReply(reply string) (Result, error) {
	switch {
	case reply == "stream: OK":
		return Result{}, nil
	case strings.HasSuffix(reply, " FOUND"):
		sig := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(reply, "stream:"), " FOUND"))
		if sig == "" {
			return Result{}, fmt.Errorf("clamd: INSTREAM answered %q", reply)
		}
		return Result{Found: true, Signature: sig}, nil
	case strings.Contains(reply, "size limit exceeded"):
		return Result{}, ErrStreamLimit
	default:
		return Result{}, fmt.Errorf("clamd: INSTREAM answered %q", reply)
	}
}

// dial connects to clamd and binds the connection's reads and writes to ctx: its deadline, and once ctx is
// done every blocked read or write fails. done closes the connection.
func (c *Client) dial(ctx context.Context) (net.Conn, func(), error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, fmt.Errorf("clamd: dial: %w", ctx.Err())
		}
		return nil, nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	stopAfter := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(past) })
	done := func() {
		stopAfter()
		_ = conn.Close()
	}
	return conn, done, nil
}

// readReply reads one null-terminated reply.
func readReply(conn net.Conn) (string, error) {
	reply, err := bufio.NewReader(io.LimitReader(conn, maxReply)).ReadString(0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimSuffix(reply, "\x00")), nil
}

// ioError reports a failed exchange, as the context's error once the context is done.
func ioError(ctx context.Context, cmd string, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("clamd: %s: %w", cmd, ctx.Err())
	}
	return fmt.Errorf("clamd: %s: %w", cmd, err)
}
