package fcm_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/fcm"
)

const (
	project = "synthetic-project"
	email   = "salus-push@synthetic-project.iam.gserviceaccount.com"
)

// clk is a clock the test moves.
type clk struct{ t time.Time }

func (c *clk) Now() time.Time { return c.t }

// google stands in for Google's OAuth token endpoint and the FCM HTTP v1 API, on one TLS server: it checks the
// service account's signed assertion and the bearer it issued, and answers a send to the token "gone" with FCM's
// UNREGISTERED.
type google struct {
	*httptest.Server
	key      *rsa.PublicKey
	sent     []map[string]any
	paths    []string
	mu       sync.Mutex
	tokens   int
	rejectAt int // answer 401 to the send with this number (1-based), once
	sends    int
}

func newGoogle(t *testing.T, key *rsa.PublicKey) *google {
	g := &google{key: key}
	g.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		switch r.URL.Path {
		case "/token":
			g.token(t, w, r)
		default:
			g.send(t, w, r)
		}
	}))
	t.Cleanup(g.Close)
	return g
}

func (g *google) token(t *testing.T, w http.ResponseWriter, r *http.Request) {
	if r.Host != "oauth2.googleapis.com" || r.Method != http.MethodPost || r.FormValue("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		t.Errorf("token request %s %s %v", r.Method, r.Host, r.Form)
	}
	parts := strings.Split(r.FormValue("assertion"), ".")
	if len(parts) != 3 {
		t.Fatalf("assertion %q", r.FormValue("assertion"))
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(g.key, crypto.SHA256, sum[:], sig); err != nil {
		t.Errorf("assertion signature: %v", err)
	}
	var claims map[string]any
	b, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(b, &claims)
	if claims["iss"] != email || claims["aud"] != "https://oauth2.googleapis.com/token" ||
		claims["scope"] != "https://www.googleapis.com/auth/firebase.messaging" || claims["exp"].(float64)-claims["iat"].(float64) != 3600 {
		t.Errorf("claims %v", claims)
	}
	g.tokens++
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "synthetic-access-" + string(rune('0'+g.tokens)), "expires_in": 3599, "token_type": "Bearer"})
}

func (g *google) send(t *testing.T, w http.ResponseWriter, r *http.Request) {
	g.sends++
	g.paths = append(g.paths, r.Host+r.URL.Path)
	if g.sends == g.rejectAt {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if want := "Bearer synthetic-access-" + string(rune('0'+g.tokens)); r.Header.Get("Authorization") != want {
		t.Errorf("authorization %q, want %q", r.Header.Get("Authorization"), want)
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	g.sent = append(g.sent, body)
	msg, _ := body["message"].(map[string]any)
	switch msg["token"] {
	case "gone":
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"code":404,"status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`)
	case "busy":
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		_, _ = io.WriteString(w, `{"name":"projects/synthetic-project/messages/1"}`)
	}
}

// client is an http.Client that reaches g for every host, trusting its certificate as the hosts' (the endpoints
// are constants: there is no override).
func (g *google) client() *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(g.Certificate())
	addr := g.Listener.Addr().String()
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "example.com", MinVersion: tls.VersionTLS12},
	}}
}

func credentials(t *testing.T, key *rsa.PrivateKey) []byte {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]string{
		"type": "service_account", "project_id": project, "client_email": email,
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"token_uri":   "https://attacker.example/token", // never used: the token endpoint is Google's, always
	})
	return b
}

func setup(t *testing.T) (*google, *fcm.Client, *clk) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	g := newGoogle(t, &key.PublicKey)
	c := &clk{t: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)}
	client, err := fcm.New(credentials(t, key), g.client(), c)
	if err != nil {
		t.Fatal(err)
	}
	return g, client, c
}

// A send is one FCM HTTP v1 request to the project, with the alert as a notification and nothing else, on an access
// token from the service account's assertion, reused until it is about to expire.
func TestSend_PostsTheNotificationWithACachedAccessToken(t *testing.T) {
	g, client, c := setup(t)
	m := push.Message{Platform: push.Android, Token: "synthetic-token", Title: "Salus", Body: "New message from your clinic"}

	for range 2 {
		if err := client.Send(t.Context(), m); err != nil {
			t.Fatalf("Send() = %v", err)
		}
	}
	c.t = c.t.Add(59 * time.Minute) // within a minute of the token's end: a new one
	if err := client.Send(t.Context(), m); err != nil {
		t.Fatalf("Send() = %v", err)
	}
	if g.tokens != 2 {
		t.Errorf("%d access tokens, want 2", g.tokens)
	}
	want := `{"message":{"android":{"priority":"high"},"notification":{"body":"New message from your clinic","title":"Salus"},"token":"synthetic-token"}}`
	if got, _ := json.Marshal(g.sent[0]); string(got) != want {
		t.Errorf("sent %s,\nwant %s", got, want)
	}
	if g.paths[0] != "fcm.googleapis.com/v1/projects/synthetic-project/messages:send" {
		t.Errorf("path %s", g.paths[0])
	}
}

// UNREGISTERED is ErrTokenGone; a 401 drops the access token so the next send fetches one; any other failure is an
// error that names the status, never the token or the text.
func TestSend_MapsTheAnswers(t *testing.T) {
	g, client, _ := setup(t)
	ctx := t.Context()
	if err := client.Send(ctx, push.Message{Token: "gone", Title: "T", Body: "B"}); !errors.Is(err, push.ErrTokenGone) {
		t.Errorf("Send(gone) = %v, want ErrTokenGone", err)
	}
	err := client.Send(ctx, push.Message{Token: "busy", Title: "Secret title", Body: "Secret body"})
	if !errors.Is(err, fcm.ErrService) || !strings.Contains(err.Error(), "503") {
		t.Errorf("Send(busy) = %v, want ErrService with the status", err)
	}
	for _, leak := range []string{"busy", "Secret", "synthetic-access"} {
		if err != nil && strings.Contains(err.Error(), leak) {
			t.Errorf("the error %q holds %q", err, leak)
		}
	}
	g.rejectAt = g.sends + 1
	if err := client.Send(ctx, push.Message{Token: "ok"}); !errors.Is(err, fcm.ErrService) {
		t.Errorf("Send(401) = %v, want ErrService", err)
	}
	if err := client.Send(ctx, push.Message{Token: "ok"}); err != nil || g.tokens != 2 {
		t.Errorf("after a 401: Send() = %v with %d tokens, want a new token", err, g.tokens)
	}
}

func TestNew_RefusesCredentialsThatAreNotAServiceAccountKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	good := credentials(t, key)
	var m map[string]string
	_ = json.Unmarshal(good, &m)
	for name, edit := range map[string]func(map[string]string){
		"no project": func(m map[string]string) { delete(m, "project_id") },
		"no e-mail":  func(m map[string]string) { delete(m, "client_email") },
		"no key":     func(m map[string]string) { delete(m, "private_key") },
		"not PEM":    func(m map[string]string) { m["private_key"] = "synthetic" },
		"not RSA PKCS": func(m map[string]string) {
			m["private_key"] = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("x")}))
		},
	} {
		c := map[string]string{}
		for k, v := range m {
			c[k] = v
		}
		edit(c)
		b, _ := json.Marshal(c)
		if _, err := fcm.New(b, &http.Client{}, &clk{}); !errors.Is(err, fcm.ErrConfig) || strings.Contains(err.Error(), "PRIVATE") {
			t.Errorf("%s: New() = %v, want ErrConfig without the key", name, err)
		}
	}
	if _, err := fcm.New([]byte("{"), &http.Client{}, &clk{}); !errors.Is(err, fcm.ErrConfig) {
		t.Errorf("New(not JSON) = %v", err)
	}
	if _, err := fcm.New(good, nil, &clk{}); !errors.Is(err, fcm.ErrConfig) {
		t.Errorf("New(no client) = %v", err)
	}
}
