// Package apns is push.Sender on the Apple Push Notification service's provider API, for iOS: one HTTP/2 request per
// message (POST https://api.push.apple.com/3/device/<token>, or the sandbox's for development builds) with a
// provider token, an ES256 JWT signed with the team's APNs auth key (.p8), renewed every 50 minutes. Plain net/http
// with an injected client (it must speak HTTP/2), no SDK.
//
// Apple is a processor outside the EU/EEA (docs/privacy/processors.md): off until the owner adds the key
// (docs/deploy/push.md). The payload is the alert's title and body and nothing else. Nothing here logs; errors name
// a status and Apple's reason, never a token, a key or the text.
package apns

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// The hosts. There is no override: tests reach their server through the client's transport.
const (
	productionHost = "https://api.push.apple.com"
	sandboxHost    = "https://api.sandbox.push.apple.com"
)

// tokenLife is how long a provider token is reused: Apple refuses one older than an hour and throttles renewals
// more often than every 20 minutes.
const (
	tokenLife   = 50 * time.Minute
	maxResponse = 1 << 14
)

var (
	// ErrConfig is returned by New for a missing setting, a key that is not a P-256 .p8 key, or no client.
	ErrConfig = errors.New("apns: invalid configuration")
	// ErrService is returned when APNs answers with an error status or cannot be reached.
	ErrService = errors.New("apns: the push service failed")
)

// Config is the team's APNs auth key (.p8, PEM) with its Key ID, the Team ID, the app's bundle id (the topic), and
// whether the app's tokens are the sandbox's (a development build).
type Config struct {
	KeyID    string
	TeamID   string
	BundleID string
	Key      []byte
	Sandbox  bool
}

// Client sends through APNs. It is safe for concurrent use.
type Client struct {
	issued time.Time
	http   *http.Client
	clock  clock.Clock
	key    *ecdsa.PrivateKey
	host   string
	keyID  string
	teamID string
	topic  string
	jwt    string
	mu     sync.Mutex
}

// New returns a Client for cfg on hc (HTTP/2: its transport must allow it). Errors never carry the key.
func New(cfg Config, hc *http.Client, clk clock.Clock) (*Client, error) {
	switch {
	case hc == nil:
		return nil, fmt.Errorf("%w: no HTTP client", ErrConfig)
	case cfg.KeyID == "" || cfg.TeamID == "" || cfg.BundleID == "":
		return nil, fmt.Errorf("%w: the key id, the team id and the bundle id are required", ErrConfig)
	}
	block, _ := pem.Decode(cfg.Key)
	if block == nil {
		return nil, fmt.Errorf("%w: the key is not a PEM .p8 key", ErrConfig)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	key, ok := parsed.(*ecdsa.PrivateKey)
	if err != nil || !ok || key.Curve != elliptic.P256() {
		return nil, fmt.Errorf("%w: the key is not a P-256 PKCS #8 key", ErrConfig)
	}
	host := productionHost
	if cfg.Sandbox {
		host = sandboxHost
	}
	return &Client{http: hc, clock: clk, key: key, host: host, keyID: cfg.KeyID, teamID: cfg.TeamID, topic: cfg.BundleID}, nil
}

// Send posts m's alert to its device. 410 and BadDeviceToken are push.ErrTokenGone; an expired provider token is
// renewed and the send tried once more.
func (c *Client) Send(ctx context.Context, m push.Message) error {
	reason, err := c.post(ctx, m)
	if reason == "ExpiredProviderToken" {
		c.renew()
		_, err = c.post(ctx, m)
	}
	return err
}

// post makes one request and returns Apple's reason with the error of a refusal.
func (c *Client) post(ctx context.Context, m push.Message) (string, error) {
	jwt, err := c.providerToken()
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]any{"aps": map[string]any{
		"alert": map[string]string{"title": m.Title, "body": m.Body}, "sound": "default",
	}})
	if err != nil {
		return "", fmt.Errorf("%w: encode", ErrService)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/3/device/"+url.PathEscape(m.Token), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%w: request", ErrService)
	}
	req.Header.Set("Authorization", "bearer "+jwt)
	req.Header.Set("apns-topic", c.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: unreachable", ErrService)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == http.StatusOK {
		return "", nil
	}
	var answer struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, maxResponse)).Decode(&answer)
	if res.StatusCode == http.StatusGone || answer.Reason == "BadDeviceToken" || answer.Reason == "Unregistered" {
		return answer.Reason, push.ErrTokenGone
	}
	return answer.Reason, fmt.Errorf("%w: status %d %s", ErrService, res.StatusCode, reasonText(answer.Reason))
}

// reasonText is Apple's reason if it is a plain word (it always is), so nothing else reaches an error.
func reasonText(r string) string {
	for _, ch := range r {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
			return ""
		}
	}
	return r
}

// providerToken returns the current provider token, signing a new one when it is tokenLife old.
func (c *Client) providerToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock.Now()
	if c.jwt != "" && now.Sub(c.issued) < tokenLife {
		return c.jwt, nil
	}
	enc := base64.RawURLEncoding
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": c.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": c.teamID, "iat": now.Unix()})
	signed := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signed))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, sum[:])
	if err != nil {
		return "", fmt.Errorf("%w: sign", ErrService)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	c.jwt, c.issued = signed+"."+enc.EncodeToString(sig), now
	return c.jwt, nil
}

// renew drops the provider token, so the next send signs a new one.
func (c *Client) renew() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jwt = ""
}
