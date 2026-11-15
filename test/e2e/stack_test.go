//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The synthetic password every journey's accounts use (12 to 128 characters, SPEC.md "Passwords").
const password = "e2e-synthetic-password"

// env returns the variable `make e2e` sets, failing the test without it: the suite never skips.
func env(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set; run the end-to-end tests with `make e2e`", name)
	}
	return v
}

// email returns a fresh synthetic address for who, so reruns against the same stack never collide.
func email(t *testing.T, who string) string {
	t.Helper()
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "e2e-" + who + "-" + hex.EncodeToString(b) + "@example.test"
}

// waitReady waits up to 30 s for the API's /readyz to answer 200: `make up` does not wait for the API.
func waitReady(t *testing.T) string {
	t.Helper()
	base := env(t, "E2E_BASE_URL")
	client := &http.Client{Timeout: 2 * time.Second}
	timeout := time.After(30 * time.Second)
	for {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+"/readyz", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		status := 0
		if err == nil {
			status = resp.StatusCode
			_ = resp.Body.Close()
			if status == http.StatusOK {
				return base
			}
		}
		select {
		case <-timeout:
			t.Fatalf("GET %s/readyz did not return 200 within 30s (last: status %d, %v)", base, status, err)
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// api is a client of the JSON and FHIR APIs, signed in with a bearer token as the mobile app is.
type api struct {
	t      *testing.T
	http   *http.Client
	base   string
	bearer string
}

func newAPI(t *testing.T) *api {
	t.Helper()
	return &api{t: t, http: &http.Client{Timeout: 10 * time.Second}, base: waitReady(t)}
}

// do sends body (JSON, or none when nil) and decodes a JSON answer into out when out is not nil. It returns
// the status.
func (a *api) do(method, path string, body, out any) int {
	a.t.Helper()
	status, raw := a.send(nil, method, path, body)
	if out != nil && len(raw) > 0 && status < 300 {
		if err := json.Unmarshal(raw, out); err != nil {
			a.t.Fatalf("%s %s = %d: decoding %s: %v", method, path, status, raw, err)
		}
	}
	return status
}

// refusal sends body and returns the status with the error code of a JSON error answer ({"error": "<code>"}).
func (a *api) refusal(method, path string, body any) (int, string) {
	a.t.Helper()
	status, raw := a.send(nil, method, path, body)
	var e struct{ Error string }
	_ = json.Unmarshal(raw, &e)
	return status, e.Error
}

// mustWith is must with extra request headers, such as If-Match.
func (a *api) mustWith(header http.Header, want int, method, path string, body, out any) {
	a.t.Helper()
	status, raw := a.send(header, method, path, body)
	if status != want {
		a.t.Fatalf("%s %s = %d, want %d", method, path, status, want)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			a.t.Fatalf("%s %s = %d: decoding %s: %v", method, path, status, raw, err)
		}
	}
}

// send sends body (JSON, or none when nil) with the extra headers and returns the status and the raw answer.
func (a *api) send(header http.Header, method, path string, body any) (int, []byte) {
	a.t.Helper()
	status, raw, _ := a.exchange(header, method, path, body)
	return status, raw
}

// changeAppointmentSettings reads the clinic's appointment settings, applies change and saves them at the version
// read (If-Match with the GET's ETag).
func (a *api) changeAppointmentSettings(change func(settings map[string]any)) {
	a.t.Helper()
	status, raw, header := a.exchange(nil, http.MethodGet, "/clinic/appointment-settings", nil)
	var settings map[string]any
	if status != http.StatusOK || json.Unmarshal(raw, &settings) != nil {
		a.t.Fatalf("GET /clinic/appointment-settings = %d %s", status, raw)
	}
	change(settings)
	a.mustWith(http.Header{"If-Match": {header.Get("ETag")}}, http.StatusOK, http.MethodPut, "/clinic/appointment-settings", settings, nil)
}

// exchange is send that also returns the answer's headers.
func (a *api) exchange(header http.Header, method, path string, body any) (int, []byte, http.Header) {
	a.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(a.t.Context(), method, a.base+path, r)
	if err != nil {
		a.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.HasPrefix(path, "/fhir/") {
		req.Header.Set("Accept", "application/fhir+json")
	}
	if a.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+a.bearer)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := a.http.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	return resp.StatusCode, raw, resp.Header
}

// must is do that fails the test unless the status is want.
func (a *api) must(want int, method, path string, body, out any) {
	a.t.Helper()
	if got := a.do(method, path, body, out); got != want {
		a.t.Fatalf("%s %s = %d, want %d", method, path, got, want)
	}
}

// login is the LoginResponse schema, and the challenge's next step.
type login struct {
	Next      string `json:"next"`
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
}

func (a *api) login(addr string) login {
	a.t.Helper()
	clearLoginLimits(a.t)
	var l login
	a.must(http.StatusOK, http.MethodPost, "/auth/login", map[string]string{"email": addr, "password": password}, &l)
	return l
}

// clearLoginLimits deletes the login rate-limit counters (ratelimit:*) in the compose Redis before a journey signs
// in, as make e2e does before the run: every sign-in of the suite comes from the playwright container's one IP, and
// the suite now signs in more often than the production limit of 30 per IP in 15 minutes allows. The limit itself is
// unchanged, and tested in internal/auth.
func clearLoginLimits(t *testing.T) {
	t.Helper()
	opts, err := goredis.ParseURL(env(t, "E2E_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	rdb := goredis.NewClient(opts)
	defer func() { _ = rdb.Close() }()
	keys, err := rdb.Keys(t.Context(), "ratelimit:*").Result()
	if err == nil && len(keys) > 0 {
		err = rdb.Del(t.Context(), keys...).Err()
	}
	if err != nil {
		t.Fatalf("clear the login rate limits: %v", err)
	}
}

// account is the Account schema.
type account struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	OrganizationID string `json:"organizationId"`
	PractitionerID string `json:"practitionerId"`
	MFAEnrolled    bool   `json:"mfaEnrolled"`
}

func (a *api) me() account {
	a.t.Helper()
	var acc account
	a.must(http.StatusOK, http.MethodGet, "/auth/me", nil, &acc)
	return acc
}

// authenticator computes the TOTP codes an authenticator app would show for an enrolment's secret.
type authenticator struct {
	secret []byte
	last   int64 // the step of the last code used; the API rejects a step already used
}

func newAuthenticator(t *testing.T, secret string) *authenticator {
	t.Helper()
	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil || len(b) == 0 {
		t.Fatalf("the enrolment's secret is not base32: %v", err)
	}
	return &authenticator{secret: b}
}

// code returns a code of a step not used yet: the current one, or the one after the last used, which the API
// accepts as one step of drift. It waits when that would be more than one step ahead.
func (a *authenticator) code() string {
	now := func() int64 { return clock.System{}.Now().Unix() / 30 }
	step := max(now(), a.last+1)
	for step > now()+1 {
		time.Sleep(time.Second)
	}
	a.last = step
	return auth.TOTPCode(a.secret, time.Unix(step*30, 0))
}

// salusctl runs the operator command (built by `make e2e`) on the owner's connection and returns its output.
func salusctl(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), env(t, "E2E_SALUSCTL"), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("salusctl %s: %v: %s", args[0], err, stderr.String())
	}
	return string(out)
}

// field returns the value after "<name> " on a line of salusctl's output.
func field(t *testing.T, out, name string) string {
	t.Helper()
	for line := range strings.Lines(out) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), name+" "); ok {
			return v
		}
	}
	t.Fatalf("salusctl printed no %q line: %q", name, out)
	return ""
}

var inviteLink = regexp.MustCompile(`/invite#([A-Za-z0-9_-]+)`)

// inviteToken waits for the invite e-mail to addr in Mailpit and returns the token from its link.
func inviteToken(t *testing.T, addr string) string {
	t.Helper()
	m := inviteLink.FindStringSubmatch(mailText(t, addr, ""))
	if m == nil {
		t.Fatal("the invite e-mail holds no invite link")
	}
	return m[1]
}

// mailText waits up to 30 s for the one e-mail to addr in Mailpit (with the subject, when not empty), deletes it
// so Mailpit is left as it was, and returns its plain text.
func mailText(t *testing.T, addr, subject string) string {
	t.Helper()
	return mailTextWithin(t, addr, subject, 30*time.Second)
}

// mailTextWithin is mailText waiting up to within: for an e-mail a job sends at a time of its own (a reminder).
func mailTextWithin(t *testing.T, addr, subject string, within time.Duration) string {
	t.Helper()
	mailpit := env(t, "E2E_MAILPIT_URL")
	client := &http.Client{Timeout: 5 * time.Second}
	query := `to:"` + addr + `"`
	if subject != "" {
		query += ` subject:"` + subject + `"`
	}
	timeout := time.After(within)
	for {
		var found struct{ Messages []struct{ ID string } }
		mailpitDo(t, client, http.MethodGet, mailpit+"/api/v1/search?query="+url.QueryEscape(query), nil, &found)
		if len(found.Messages) == 1 {
			id := found.Messages[0].ID
			var msg struct{ Text string }
			mailpitDo(t, client, http.MethodGet, mailpit+"/api/v1/message/"+id, nil, &msg)
			mailpitDo(t, client, http.MethodDelete, mailpit+"/api/v1/messages", map[string][]string{"IDs": {id}}, nil)
			return msg.Text
		}
		if len(found.Messages) > 1 {
			t.Fatalf("%d e-mails %q to the address in Mailpit, want 1", len(found.Messages), subject)
		}
		select {
		case <-timeout:
			t.Fatalf("no e-mail %q to the address in Mailpit within %v", subject, within)
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func mailpitDo(t *testing.T, client *http.Client, method, u string, body, out any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), method, u, r)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Mailpit %s: %v", method, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit %s %s = %d", method, req.URL.Path, resp.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(fmt.Errorf("Mailpit %s: %w", req.URL.Path, err))
		}
	}
}
