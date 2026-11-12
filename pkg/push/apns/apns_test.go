package apns_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/apns"
)

const (
	keyID  = "SYNTHKEY01"
	teamID = "SYNTHTEAM1"
	bundle = "ba.salus.synthetic"
)

type clk struct{ t time.Time }

func (c *clk) Now() time.Time { return c.t }

// apple stands in for the APNs provider API over HTTP/2: it checks the provider token's ES256 signature and claims
// and the headers, and answers the token "gone" with 410 Unregistered, "bad" with 400 BadDeviceToken, "expired" with
// 403 ExpiredProviderToken once.
type apple struct {
	*httptest.Server
	key     *ecdsa.PublicKey
	jwts    map[string]bool
	sent    []string
	hosts   []string
	mu      sync.Mutex
	expired bool
}

func newApple(t *testing.T, key *ecdsa.PublicKey) *apple {
	a := &apple{key: key, jwts: map[string]bool{}}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.check(t, r)
		body, _ := io.ReadAll(r.Body)
		a.sent = append(a.sent, string(body))
		a.hosts = append(a.hosts, r.Host+r.URL.Path)
		switch strings.TrimPrefix(r.URL.Path, "/3/device/") {
		case "gone":
			w.WriteHeader(http.StatusGone)
			_, _ = io.WriteString(w, `{"reason":"Unregistered","timestamp":1}`)
		case "bad":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"reason":"BadDeviceToken"}`)
		case "payload":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"reason":"PayloadEmpty"}`)
		case "expired":
			if !a.expired {
				a.expired = true
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, `{"reason":"ExpiredProviderToken"}`)
			}
		}
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	a.Server = srv
	t.Cleanup(srv.Close)
	return a
}

func (a *apple) check(t *testing.T, r *http.Request) {
	if r.ProtoMajor != 2 || r.Method != http.MethodPost || r.Header.Get("apns-topic") != bundle ||
		r.Header.Get("apns-push-type") != "alert" || r.Header.Get("apns-priority") != "10" {
		t.Errorf("request %s %s, headers %v", r.Proto, r.Method, r.Header)
	}
	jwt := strings.TrimPrefix(r.Header.Get("Authorization"), "bearer ")
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("authorization %q", r.Header.Get("Authorization"))
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if len(sig) != 64 || !ecdsa.Verify(a.key, sum[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Errorf("the provider token's signature does not verify")
	}
	var header, claims map[string]any
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	c, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(h, &header)
	_ = json.Unmarshal(c, &claims)
	if header["alg"] != "ES256" || header["kid"] != keyID || claims["iss"] != teamID {
		t.Errorf("header %v, claims %v", header, claims)
	}
	a.jwts[jwt] = true
}

func (a *apple) client() *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(a.Certificate())
	addr := a.Listener.Addr().String()
	return &http.Client{Transport: &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "example.com", MinVersion: tls.VersionTLS12},
	}}
}

func p8(t *testing.T, key *ecdsa.PrivateKey) []byte {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func setup(t *testing.T, sandbox bool) (*apple, *apns.Client, *clk) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	a := newApple(t, &key.PublicKey)
	c := &clk{t: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)}
	client, err := apns.New(apns.Config{Key: p8(t, key), KeyID: keyID, TeamID: teamID, BundleID: bundle, Sandbox: sandbox}, a.client(), c)
	if err != nil {
		t.Fatal(err)
	}
	return a, client, c
}

// A send is one HTTP/2 request to the device's path with the alert alone, on a provider token reused for 50 minutes
// (Apple refuses one older than an hour, and one renewed more than every 20 minutes).
func TestSend_PostsTheAlertWithAReusedProviderToken(t *testing.T) {
	a, client, c := setup(t, false)
	m := push.Message{Platform: push.IOS, Token: "synthetic-token", Title: "Salus", Body: "New message from your clinic"}
	for range 2 {
		if err := client.Send(t.Context(), m); err != nil {
			t.Fatalf("Send() = %v", err)
		}
	}
	if len(a.jwts) != 1 {
		t.Errorf("%d provider tokens for two sends, want 1", len(a.jwts))
	}
	c.t = c.t.Add(51 * time.Minute)
	if err := client.Send(t.Context(), m); err != nil || len(a.jwts) != 2 {
		t.Errorf("after 51 minutes: %v, %d provider tokens, want a new one", err, len(a.jwts))
	}
	if want := `{"aps":{"alert":{"body":"New message from your clinic","title":"Salus"},"sound":"default"}}`; a.sent[0] != want {
		t.Errorf("sent %s,\nwant %s", a.sent[0], want)
	}
	if a.hosts[0] != "api.push.apple.com/3/device/synthetic-token" {
		t.Errorf("sent to %s", a.hosts[0])
	}
}

func TestSend_UsesTheSandboxWhenTold(t *testing.T) {
	a, client, _ := setup(t, true)
	if err := client.Send(t.Context(), push.Message{Token: "synthetic-token"}); err != nil {
		t.Fatal(err)
	}
	if a.hosts[0] != "api.sandbox.push.apple.com/3/device/synthetic-token" {
		t.Errorf("sent to %s", a.hosts[0])
	}
}

// 410 and BadDeviceToken are ErrTokenGone; an expired provider token is renewed and the send tried once more; any
// other failure names the status and Apple's reason, never the token or the text.
func TestSend_MapsTheAnswers(t *testing.T) {
	a, client, _ := setup(t, false)
	ctx := t.Context()
	for _, tok := range []string{"gone", "bad"} {
		if err := client.Send(ctx, push.Message{Token: tok}); !errors.Is(err, push.ErrTokenGone) {
			t.Errorf("Send(%s) = %v, want ErrTokenGone", tok, err)
		}
	}
	err := client.Send(ctx, push.Message{Token: "payload", Title: "Secret title", Body: "Secret body"})
	if !errors.Is(err, apns.ErrService) || !strings.Contains(err.Error(), "400 PayloadEmpty") || strings.Contains(err.Error(), "Secret") {
		t.Errorf("Send(payload) = %v, want ErrService with the status and reason only", err)
	}
	before := len(a.jwts)
	if err := client.Send(ctx, push.Message{Token: "expired"}); err != nil || len(a.jwts) != before+1 {
		t.Errorf("Send(expired) = %v with %d new tokens, want a retry on a new token", err, len(a.jwts)-before)
	}
}

func TestNew_RefusesAnIncompleteConfiguration(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rsaLike := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("x")})
	good := apns.Config{Key: p8(t, key), KeyID: keyID, TeamID: teamID, BundleID: bundle}
	for name, edit := range map[string]func(*apns.Config){
		"no key id":    func(c *apns.Config) { c.KeyID = "" },
		"no team":      func(c *apns.Config) { c.TeamID = "" },
		"no bundle":    func(c *apns.Config) { c.BundleID = "" },
		"no key":       func(c *apns.Config) { c.Key = nil },
		"not a P-256":  func(c *apns.Config) { c.Key = rsaLike },
		"not PEM text": func(c *apns.Config) { c.Key = []byte("synthetic") },
	} {
		c := good
		edit(&c)
		if _, err := apns.New(c, &http.Client{}, &clk{}); !errors.Is(err, apns.ErrConfig) {
			t.Errorf("%s: New() = %v, want ErrConfig", name, err)
		}
	}
	if _, err := apns.New(good, nil, &clk{}); !errors.Is(err, apns.ErrConfig) {
		t.Errorf("New(no client) = %v", err)
	}
}
