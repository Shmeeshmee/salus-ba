// Package fcm is push.Sender on Firebase Cloud Messaging's HTTP v1 API, for Android: one request per message
// (POST https://fcm.googleapis.com/v1/projects/<project>/messages:send) on an OAuth access token from the
// project's service account (a signed JWT assertion to Google's token endpoint), reused until a minute before it
// ends. Plain net/http with an injected client, no SDK: two endpoints and three JSON shapes.
//
// Google is a processor outside the EU/EEA (docs/privacy/processors.md): off until the owner adds the service
// account's key (docs/deploy/push.md). The message carries the alert's title and body and nothing else (no data
// payload, no id). Nothing here logs; errors name a status, never a token, a key or the text.
package fcm

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
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
	"strings"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// The endpoints. There is no override (the credentials' token_uri is not used): tests reach their server through
// the client's transport.
const (
	tokenURL = "https://oauth2.googleapis.com/token"
	sendURL  = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	scope    = "https://www.googleapis.com/auth/firebase.messaging"
)

// assertionLife is how long a token request's assertion is valid (Google's maximum); renewEarly is how long before
// an access token's end a new one is fetched.
const (
	assertionLife = time.Hour
	renewEarly    = time.Minute
	maxResponse   = 1 << 16
)

var (
	// ErrConfig is returned by New for credentials that are not a service account's key, or no client.
	ErrConfig = errors.New("fcm: invalid configuration")
	// ErrService is returned when Google answers with an error status or cannot be reached.
	ErrService = errors.New("fcm: the push service failed")
)

// Client sends through FCM. It is safe for concurrent use.
type Client struct {
	expires time.Time
	http    *http.Client
	clock   clock.Clock
	key     *rsa.PrivateKey
	email   string
	project string
	token   string
	mu      sync.Mutex
}

// New returns a Client on the service account's JSON key (as the Firebase console downloads it) and hc, whose
// timeouts and transport the caller sets. Errors never carry the key.
func New(credentials []byte, hc *http.Client, clk clock.Clock) (*Client, error) {
	var sa struct {
		ProjectID   string `json:"project_id"`
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(credentials, &sa); err != nil {
		return nil, fmt.Errorf("%w: the credentials are not JSON", ErrConfig)
	}
	switch {
	case hc == nil:
		return nil, fmt.Errorf("%w: no HTTP client", ErrConfig)
	case sa.ProjectID == "" || sa.ClientEmail == "":
		return nil, fmt.Errorf("%w: the credentials name no project or service account", ErrConfig)
	}
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("%w: the credentials hold no PEM private key", ErrConfig)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	key, ok := parsed.(*rsa.PrivateKey)
	if err != nil || !ok {
		return nil, fmt.Errorf("%w: the private key is not an RSA PKCS #8 key", ErrConfig)
	}
	return &Client{http: hc, clock: clk, key: key, email: sa.ClientEmail, project: sa.ProjectID}, nil
}

// Send posts m as a notification. FCM's UNREGISTERED is push.ErrTokenGone; a 401 drops the access token so the
// next send fetches a new one.
func (c *Client) Send(ctx context.Context, m push.Message) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"message": map[string]any{
		"token":        m.Token,
		"notification": map[string]string{"title": m.Title, "body": m.Body},
		"android":      map[string]string{"priority": "high"},
	}})
	if err != nil {
		return fmt.Errorf("%w: encode", ErrService)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(sendURL, url.PathEscape(c.project)), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: request", ErrService)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: unreachable", ErrService)
	}
	defer func() { _ = res.Body.Close() }()
	answer, _ := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	switch {
	case res.StatusCode == http.StatusOK:
		return nil
	case res.StatusCode == http.StatusNotFound && bytes.Contains(answer, []byte(`"UNREGISTERED"`)):
		return push.ErrTokenGone
	case res.StatusCode == http.StatusUnauthorized:
		c.forget()
	}
	return fmt.Errorf("%w: status %d", ErrService, res.StatusCode)
}

// accessToken returns the cached access token, or fetches one when there is none or it ends within renewEarly.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock.Now()
	if c.token != "" && now.Add(renewEarly).Before(c.expires) {
		return c.token, nil
	}
	assertion, err := c.assertion(now)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assertion}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: token request", ErrService)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: the token endpoint is unreachable", ErrService)
	}
	defer func() { _ = res.Body.Close() }()
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, maxResponse)).Decode(&tok) != nil || tok.AccessToken == "" {
		return "", fmt.Errorf("%w: token status %d", ErrService, res.StatusCode)
	}
	c.token, c.expires = tok.AccessToken, now.Add(time.Duration(tok.ExpiresIn)*time.Second)
	return c.token, nil
}

// forget drops the cached access token.
func (c *Client) forget() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
}

// assertion is the service account's RS256 JWT asking for the messaging scope, issued at now.
func (c *Client) assertion(now time.Time) (string, error) {
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]any{
		"iss": c.email, "scope": scope, "aud": tokenURL, "iat": now.Unix(), "exp": now.Add(assertionLife).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("%w: claims", ErrService)
	}
	signed := header + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("%w: sign", ErrService)
	}
	return signed + "." + enc.EncodeToString(sig), nil
}
