package api_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
)

// testDrafter is the api's default drafting engine: the fake, keeping no calls.
func testDrafter() llm.Drafter { return &llmtest.Drafter{DiscardCalls: true} }

func testSummarizer() llm.Summarizer { return &llmtest.Summarizer{DiscardCalls: true} }

// testLiveKit is a synthetic LiveKit configuration; the server URL resolves nowhere, so a test that needs
// LiveKit itself uses the compose one through TEST_LIVEKIT_*.
func testLiveKit() config.LiveKit {
	return config.LiveKit{
		URL: "http://livekit.invalid:7880", PublicURL: "ws://localhost:7880",
		APIKey: "synthetic-key", APISecret: "synthetic-webhook-secret-of-enough-length",
	}
}

// livekitSignature is the Authorization header LiveKit sends with body: an HS256 JWT issued by the key,
// unexpired, carrying the body's SHA-256 (written here by hand, as only the provider may use LiveKit's SDK).
func livekitSignature(t *testing.T, lk config.LiveKit, body []byte) string {
	t.Helper()
	sum := sha256.Sum256(body)
	now := clock.System{}.Now()
	claims, err := json.Marshal(map[string]any{
		"iss": lk.APIKey, "nbf": now.Add(-time.Minute).Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		"sha256": base64.StdEncoding.EncodeToString(sum[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(lk.APISecret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil))
}

// The api serves LiveKit's webhook without a session: an unsigned one is 401, a signed one for a room we do
// not know is 200 with no effect, and the log holds no room or identity from the body.
func TestNew_ServesTheLiveKitWebhook(t *testing.T) {
	var log bytes.Buffer
	lk := testLiveKit()
	app, err := api.New(t.Context(), testConfig(t), lk, testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewJSONHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()
	body := []byte(`{"event":"participant_joined","room":{"name":"r_secretroom"},"participant":{"identity":"p_secretidentity"}}`)
	for _, tc := range []struct {
		authorization string
		want          int
	}{
		{authorization: "", want: http.StatusUnauthorized},
		{authorization: "Bearer not-a-session", want: http.StatusUnauthorized},
		{authorization: livekitSignature(t, lk, body), want: http.StatusOK},
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/livekit", bytes.NewReader(body))
		req.Header.Set("Authorization", tc.authorization)
		rec := httptest.NewRecorder()
		app.Handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("POST /webhooks/livekit with %q = %d %s, want %d", tc.authorization, rec.Code, rec.Body, tc.want)
		}
	}
	if !strings.Contains(log.String(), `"msg":"unknown room"`) || strings.Contains(log.String(), "secret") {
		t.Errorf("log %s; want the unknown room and nothing from the body", log.String())
	}
}
