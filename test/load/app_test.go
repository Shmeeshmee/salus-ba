//go:build loadtest

package load_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
)

// target is the database and Redis the driver runs on: owner (the schema's owner, for the seed and the readings),
// app (salus_app, as the binaries log in), stats (salus_stats, the aggregate builds' login).
type target struct {
	owner, app, stats *pgxpool.Pool
	appURL, redisURL  string
}

// newServer is the api as cmd/api builds it (api.New, every domain real) on t, behind a real HTTP server on the
// loopback, with synthetic keys and the fake language models. The object store and LiveKit are never reached by the
// paths measured.
func newServer(ctx context.Context, t target) (*httptest.Server, func(), error) {
	cfg := config.Config{
		DatabaseURL: t.appURL, RedisURL: t.redisURL,
		TOTPKey:      []byte("synthetic 32-byte TOTP key, load"),
		RateLimitKey: []byte("synthetic 32-byte rate-limit key"),
		WebhookKey:   []byte("synthetic 32-byte webhook key!!!"),
		FHIRBaseURL:  "https://api.example.test/fhir/R4",
	}
	lk := config.LiveKit{URL: "http://livekit.invalid:7880", PublicURL: "ws://localhost:7880", APIKey: "synthetic-key", APISecret: "synthetic-webhook-secret-of-enough-length"}
	files := config.ObjectStore{
		Endpoint: "http://objectstore.invalid:3900", Region: "garage", Bucket: "salus-binaries", AccessKeyID: "synthetic", SecretAccessKey: "synthetic",
		KEKID: "load-1", KEK: []byte("synthetic 32-byte KEK for loads!"),
	}
	app, err := api.New(ctx, cfg, lk, files, &llmtest.Drafter{DiscardCalls: true}, &llmtest.Summarizer{DiscardCalls: true}, slog.New(slog.DiscardHandler))
	if err != nil {
		return nil, nil, err
	}
	srv := httptest.NewServer(app.Handler)
	return srv, func() { srv.Close(); app.Close() }, nil
}

// token is the bearer token of account's load session: 32 bytes derived from its id, so the driver and the database
// agree without storing it. Its SHA-256 is the session's token_hash, as auth stores it.
func token(account string) string {
	raw := sha256.Sum256([]byte("load-session:" + account))
	return base64.RawURLEncoding.EncodeToString(raw[:])
}

// openSessions gives every seeded staff account and the sampled patients an open session of token(account), as a
// sign-in would (a staff session is MFA-verified and of the account's clinic).
func openSessions(ctx context.Context, owner *pgxpool.Pool, s seeded) error {
	var accounts []string
	for i, c := range s.clinics {
		accounts = append(accounts, c.admin, c.reception)
		for _, r := range c.roles {
			accounts = append(accounts, r.account)
		}
		for _, p := range s.patients[i] {
			accounts = append(accounts, p.account)
		}
	}
	_, err := owner.Exec(ctx, `INSERT INTO session (account_id, organization_id, mfa_verified, token_hash, expires_at)
		SELECT a.id, a.organization_id, a.organization_id IS NOT NULL,
			sha256(sha256(convert_to('load-session:' || a.id::text, 'UTF8'))), now() + interval '12 hours'
		FROM account a WHERE a.id = ANY($1::uuid[])
		ON CONFLICT (token_hash) DO UPDATE SET expires_at = excluded.expires_at, revoked_at = NULL`, accounts)
	return err
}

// loadSeeded reads back the seeded clinics, their staff and roles, and up to perClinic patients of each clinic with
// their consent there.
func loadSeeded(ctx context.Context, owner *pgxpool.Pool, perClinic int) (seeded, error) {
	var s seeded
	rows, err := owner.Query(ctx, `SELECT o.id::text,
			(SELECT a.id::text FROM account a WHERE a.organization_id = o.id AND a.role = 'admin' ORDER BY a.email LIMIT 1),
			(SELECT a.id::text FROM account a WHERE a.organization_id = o.id AND a.role = 'reception' ORDER BY a.email LIMIT 1)
		FROM organization o WHERE o.name LIKE $1 || '%' ORDER BY o.name`, seedPrefix)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var c seededClinic
		if err := rows.Scan(&c.id, &c.admin, &c.reception); err != nil {
			rows.Close()
			return s, err
		}
		s.clinics = append(s.clinics, c)
	}
	rows.Close()
	if len(s.clinics) == 0 {
		return s, fmt.Errorf("load: no seeded clinic")
	}
	for i := range s.clinics {
		c := &s.clinics[i]
		rows, err := owner.Query(ctx, `SELECT r.id::text, r.practitioner_id::text, a.id::text, s.id::text
			FROM practitioner_role r JOIN account a ON a.practitioner_id = r.practitioner_id JOIN schedule s ON s.practitioner_role_id = r.id
			WHERE r.organization_id = $1 ORDER BY r.id`, c.id)
		if err != nil {
			return s, err
		}
		for rows.Next() {
			var r seededRole
			if err := rows.Scan(&r.id, &r.practitioner, &r.account, &r.schedule); err != nil {
				rows.Close()
				return s, err
			}
			c.roles = append(c.roles, r)
		}
		rows.Close()
		rows, err = owner.Query(ctx, `SELECT p.id::text, p.account_id::text FROM consent k JOIN patient p ON p.id = k.patient_id
			WHERE k.organization_id = $1 AND k.status = 'active' AND p.account_id IS NOT NULL ORDER BY p.id LIMIT $2`, c.id, perClinic)
		if err != nil {
			return s, err
		}
		var ps []seededPatient
		for rows.Next() {
			var p seededPatient
			if err := rows.Scan(&p.id, &p.account); err != nil {
				rows.Close()
				return s, err
			}
			ps = append(ps, p)
		}
		rows.Close()
		s.patients = append(s.patients, ps)
	}
	return s, nil
}

// request is one HTTP request of a path, signed in as account ("" for none).
type request struct {
	method, path, account, body string
	want                        int
}

// do sends r to base and returns its status; the body is read to its end and dropped, as a client reads it.
func do(ctx context.Context, client *http.Client, base string, r request) (int, error) {
	var body io.Reader
	if r.body != "" {
		body = strings.NewReader(r.body)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, base+r.path, body)
	if err != nil {
		return 0, err
	}
	if r.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.account != "" {
		req.Header.Set("Authorization", "Bearer "+token(r.account))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, err
}
