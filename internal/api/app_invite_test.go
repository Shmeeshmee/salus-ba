package api_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const inviteClinic = "c0000000-0000-4000-8000-00000000000c"

// issueToken does what the invite_email job does for the invite of email: stores the SHA-256 of a new token,
// and returns the token.
func issueToken(t *testing.T, exec func(string, ...any), email string) string {
	t.Helper()
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	sum := sha256.Sum256(raw)
	exec("UPDATE invite SET token_hash = $1 WHERE email = $2 AND accepted_at IS NULL", sum[:], email)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// accept accepts the invite of token and confirms the first code, returning the session token.
func accept(t *testing.T, c *client, token string) string {
	t.Helper()
	rec := c.do(http.MethodPost, "/invites/accept", `{"token":"`+token+`","password":"`+testPassword+`"}`)
	var acc struct{ Challenge, Secret, URI string }
	decode(t, rec, &acc)
	if rec.Code != http.StatusCreated || acc.Challenge == "" || acc.Secret == "" {
		t.Fatalf("POST /invites/accept = %d %s, want the enrolment", rec.Code, rec.Body)
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(acc.Secret)
	if err != nil {
		t.Fatal(err)
	}
	code := auth.TOTPCode(secret, clock.System{}.Now())
	rec = c.do(http.MethodPost, "/auth/mfa/confirm", `{"challenge":"`+acc.Challenge+`","code":"`+code+`"}`)
	var login struct{ Next, Token string }
	decode(t, rec, &login)
	if rec.Code != http.StatusOK || login.Next != "done" || login.Token == "" {
		t.Fatalf("POST /auth/mfa/confirm after acceptance = %d %s, want the session", rec.Code, rec.Body)
	}
	return login.Token
}

func TestNew_AnAdminInvitesAPractitionerWhoAcceptsAndIsInTheDirectory(t *testing.T) {
	app, cfg := newApp(t)
	count := openDB(t, cfg)
	owner, err := postgres.Open(t.Context(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Close)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	// What salusctl create-clinic leaves: the clinic and its first admin's open invite.
	exec("INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic')", inviteClinic)
	exec("INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now() + interval '72 hours', 'admin', 'dee@example.test')", inviteClinic)

	admin := &client{t: t, h: app.Handler}
	admin.bearer = accept(t, admin, issueToken(t, exec, "dee@example.test"))

	rec := admin.do(http.MethodPost, "/invites", `{"email":"Cy@example.test","role":"practitioner","givenName":"Cy","familyName":"Example"}`)
	var inv struct{ ID, Email string }
	decode(t, rec, &inv)
	if rec.Code != http.StatusCreated || inv.Email != "cy@example.test" {
		t.Fatalf("POST /invites = %d %s, want the invite", rec.Code, rec.Body)
	}
	if n := count(`SELECT count(*) FROM river_job WHERE kind = 'invite_email' AND queue = 'email'
		AND args = jsonb_build_object('invite_id', $1::text, 'organization_id', $2::text)`, inv.ID, inviteClinic); n != 1 {
		t.Errorf("%d invite_email jobs with the two ids, want 1", n)
	}
	if rec := admin.do(http.MethodGet, "/invites", ""); rec.Code != http.StatusOK {
		t.Errorf("GET /invites = %d %s", rec.Code, rec.Body)
	}

	practitioner := &client{t: t, h: app.Handler}
	practitioner.bearer = accept(t, practitioner, issueToken(t, exec, "cy@example.test"))
	if rec := practitioner.do(http.MethodPost, "/invites", `{"email":"eve@example.test","role":"admin"}`); rec.Code != http.StatusForbidden {
		t.Errorf("POST /invites by a practitioner = %d %s, want 403", rec.Code, rec.Body)
	}

	if n := count(`SELECT count(*) FROM account a JOIN practitioner p ON p.id = a.practitioner_id
		JOIN practitioner_role r ON r.practitioner_id = p.id
		WHERE a.email = 'cy@example.test' AND a.role = 'practitioner' AND a.organization_id = $1 AND a.mfa_enrolled_at IS NOT NULL
		AND p.given_name = 'Cy' AND p.family_name = 'Example' AND r.organization_id = $1`, inviteClinic); n != 1 {
		t.Errorf("%d linked, enrolled practitioner accounts with a Practitioner and a role, want 1", n)
	}
	if n := count(`SELECT count(*) FROM invite WHERE accepted_at IS NOT NULL`); n != 2 {
		t.Errorf("%d accepted invites, want 2", n)
	}
	// One audited create per invite, account (two), Practitioner and role, and the practitioner's refusal.
	for entity, want := range map[string]int{"Invite/%": 1, "Account/%": 2, "Practitioner/%": 1, "PractitionerRole/%": 1} {
		if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = 'create' AND outcome = 0 AND entity_reference LIKE $1`, entity); n != want {
			t.Errorf("%d audited creates of %s, want %d", n, entity, want)
		}
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = 'create' AND outcome = 4 AND entity_reference IS NULL`); n != 1 {
		t.Errorf("%d audited refused invites, want 1", n)
	}
}
