package notification

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// maxAnswer is how much of an endpoint's answer is read (and dropped) so the connection can be reused.
const maxAnswer = 4096

// errBlockedAddress is a connection refused because its address is not public.
var errBlockedAddress = errors.New("notification: the webhook's address is not public")

type httpPoster struct {
	client *http.Client
}

// NewWebhookPoster returns the Poster of the clinics' webhooks. It builds its own client, because its guards are the
// point: https with the system's roots, no proxy from the environment, no redirect followed, at most timeout per
// attempt, and a dialer that refuses, at connect time on the resolved address (so a name that is rebound to an
// internal address later is refused too), every address that is not public: loopback, private, link-local,
// unique-local, multicast, unspecified, carrier-grade NAT. Errors never carry the URL.
func NewWebhookPoster(timeout time.Duration) Poster {
	return newPoster(timeout, &tls.Config{MinVersion: tls.VersionTLS12}, publicAddr)
}

// newPoster is the poster on tlsConfig, connecting only to the addresses allow accepts (tests trust their server).
func newPoster(timeout time.Duration, tlsConfig *tls.Config, allow func(netip.Addr) bool) *httpPoster {
	dialer := &net.Dialer{Timeout: timeout, Control: func(_, address string, _ syscall.RawConn) error {
		ap, err := netip.ParseAddrPort(address)
		if err != nil || !allow(ap.Addr()) {
			return errBlockedAddress
		}
		return nil
	}}
	return &httpPoster{client: &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil, DialContext: dialer.DialContext, TLSClientConfig: tlsConfig, ForceAttemptHTTP2: true,
			TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout, MaxIdleConnsPerHost: 2, IdleConnTimeout: time.Minute,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (p *httpPoster) Post(ctx context.Context, url string, body []byte, header map[string]string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, errors.New("notification: the webhook's URL is not valid")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", "Salus-Webhooks/1")
	res, err := p.client.Do(req)
	switch {
	case errors.Is(err, errBlockedAddress):
		return 0, errBlockedAddress
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return 0, fmt.Errorf("notification: the webhook did not answer in time: %w", context.DeadlineExceeded)
	case err != nil:
		return 0, errors.New("notification: the webhook is unreachable")
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxAnswer))
	_ = res.Body.Close()
	return res.StatusCode, nil
}
