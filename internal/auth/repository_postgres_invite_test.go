package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const otherClinicID = "d0000000-0000-4000-8000-00000000000d"

// inviteDB is a test database with clinics clinicID and otherClinicID, and the repository as salus_app.
type inviteDB struct {
	owner *pgxpool.Pool
	repo  auth.Repository
	tx    *postgres.Transactor
}

func newInviteDB(t *testing.T) inviteDB {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), "INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic'), ($2, 'Other clinic')",
		clinicID, otherClinicID); err != nil {
		t.Fatal(err)
	}
	return inviteDB{owner: owner, repo: auth.NewPostgresRepository(app), tx: postgres.NewTransactor(app)}
}

// in runs fn in a transaction scoped to org.
func (d inviteDB) in(t *testing.T, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return d.tx.WithTx(t.Context(), func(ctx context.Context) error {
		if err := d.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func (d inviteDB) put(t *testing.T, inv auth.Invite) string {
	t.Helper()
	var id string
	err := d.in(t, inv.OrganizationID, func(ctx context.Context) error {
		var err error
		id, err = d.repo.PutInvite(ctx, inv)
		return err
	})
	if err != nil {
		t.Fatalf("PutInvite() error = %v", err)
	}
	return id
}

func storedInvite(org string) auth.Invite {
	return auth.Invite{
		OrganizationID: org, Email: "cy@example.test", Role: auth.RolePractitioner, GivenName: "Cy", FamilyName: "Example",
		CreatedAt: now, ExpiresAt: now.Add(72 * time.Hour),
	}
}

func TestPutInvite_StoresAnInviteAndReplacesTheOpenOneForTheSameEmail(t *testing.T) {
	t.Parallel()
	d := newInviteDB(t)
	id := d.put(t, storedInvite(clinicID))
	hash := sha256.Sum256([]byte("synthetic token"))
	if err := d.in(t, clinicID, func(ctx context.Context) error { return d.repo.SetInviteToken(ctx, id, hash[:]) }); err != nil {
		t.Fatal(err)
	}

	again := storedInvite(clinicID)
	again.Role, again.GivenName, again.FamilyName, again.ExpiresAt = auth.RoleAdmin, "", "", now.Add(100*time.Hour)
	if got := d.put(t, again); got != id {
		t.Errorf("PutInvite() again = %s, want the open invite %s replaced", got, id)
	}
	var got auth.Invite
	err := d.in(t, clinicID, func(ctx context.Context) error {
		var err error
		got, err = d.repo.InviteByID(ctx, id)
		return err
	})
	want := again
	want.ID = id
	got.CreatedAt, got.ExpiresAt = got.CreatedAt.UTC(), got.ExpiresAt.UTC()
	if err != nil || got != want {
		t.Errorf("InviteByID() = %+v, %v;\nwant %+v", got, err, want)
	}
	var tokens int
	if err := d.owner.QueryRow(t.Context(), "SELECT count(token_hash) FROM invite").Scan(&tokens); err != nil || tokens != 0 {
		t.Errorf("%d token hashes after the replacement (%v), want 0: the earlier link must stop working", tokens, err)
	}
	if other := d.put(t, storedInvite(otherClinicID)); other == id {
		t.Error("an invite for the same e-mail in another clinic replaced this one")
	}
}

func TestInvites_FindTheClinicByTokenThenTheInviteThroughItsScope(t *testing.T) {
	t.Parallel()
	d := newInviteDB(t)
	id := d.put(t, storedInvite(clinicID))
	hash := sha256.Sum256([]byte("synthetic token"))
	if err := d.in(t, clinicID, func(ctx context.Context) error { return d.repo.SetInviteToken(ctx, id, hash[:]) }); err != nil {
		t.Fatal(err)
	}

	org, err := d.repo.InviteOrganization(t.Context(), hash[:])
	if err != nil || org != clinicID {
		t.Fatalf("InviteOrganization() = %q, %v; want %s", org, err, clinicID)
	}
	unknown := sha256.Sum256([]byte("unknown"))
	if _, err := d.repo.InviteOrganization(t.Context(), unknown[:]); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Errorf("InviteOrganization(unknown) = %v, want ErrInvalidInvite", err)
	}
	for set, want := range map[string]error{clinicID: nil, otherClinicID: auth.ErrInvalidInvite} {
		err := d.in(t, set, func(ctx context.Context) error {
			got, err := d.repo.InviteByTokenHash(ctx, hash[:])
			if err == nil && got.ID != id {
				t.Errorf("InviteByTokenHash() = %+v, want invite %s", got, id)
			}
			return err
		})
		if !errors.Is(err, want) {
			t.Errorf("InviteByTokenHash() with clinic %s set = %v, want %v", set, err, want)
		}
	}
}

func TestOpenInvites_ListsOnlyOpenInvitesOfTheClinicOldestFirst(t *testing.T) {
	t.Parallel()
	d := newInviteDB(t)
	first := d.put(t, storedInvite(clinicID))
	later := storedInvite(clinicID)
	later.Email, later.CreatedAt = "dee@example.test", now.Add(time.Minute)
	second := d.put(t, later)
	expired := storedInvite(clinicID)
	expired.Email, expired.ExpiresAt = "eve@example.test", now
	d.put(t, expired)
	accepted := storedInvite(clinicID)
	accepted.Email = "fay@example.test"
	acceptedID := d.put(t, accepted)
	d.put(t, storedInvite(otherClinicID))
	if err := d.in(t, clinicID, func(ctx context.Context) error { return d.repo.AcceptInvite(ctx, acceptedID, now) }); err != nil {
		t.Fatal(err)
	}

	var got []auth.Invite
	err := d.in(t, clinicID, func(ctx context.Context) error {
		var err error
		got, err = d.repo.OpenInvites(ctx, clinicID, now)
		return err
	})
	if err != nil || len(got) != 2 || got[0].ID != first || got[1].ID != second {
		t.Errorf("OpenInvites() = %+v, %v; want %s then %s", got, err, first, second)
	}
	if got, err := d.repo.OpenInvites(t.Context(), "not-a-uuid", now); err != nil || len(got) != 0 {
		t.Errorf("OpenInvites(not a UUID) = %v, %v; want none", got, err)
	}
}

func TestAcceptInvite_AcceptsAnOpenInviteOnce(t *testing.T) {
	t.Parallel()
	d := newInviteDB(t)
	id := d.put(t, storedInvite(clinicID))
	accept := func(id string, at time.Time) error {
		return d.in(t, clinicID, func(ctx context.Context) error { return d.repo.AcceptInvite(ctx, id, at) })
	}
	if err := accept(id, now.Add(73*time.Hour)); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Errorf("AcceptInvite() after expiry = %v, want ErrInvalidInvite", err)
	}
	if err := accept(id, now); err != nil {
		t.Fatalf("AcceptInvite() = %v", err)
	}
	for _, again := range []string{id, "not-a-uuid"} {
		if err := accept(again, now); !errors.Is(err, auth.ErrInvalidInvite) {
			t.Errorf("AcceptInvite(%s) again = %v, want ErrInvalidInvite", again, err)
		}
	}
	err := d.in(t, clinicID, func(ctx context.Context) error {
		if _, err := d.repo.InviteByID(ctx, "not-a-uuid"); !errors.Is(err, auth.ErrInvalidInvite) {
			t.Errorf("InviteByID(not a UUID) = %v, want ErrInvalidInvite", err)
		}
		return d.repo.SetInviteToken(ctx, "1a000000-0000-4000-8000-000000000000", []byte("x"))
	})
	if !errors.Is(err, auth.ErrInvalidInvite) {
		t.Errorf("SetInviteToken(unknown) = %v, want ErrInvalidInvite", err)
	}
}

func TestStaffAccount_IsCreatedInItsClinicLinkedResetAndSignedOutEverywhere(t *testing.T) {
	t.Parallel()
	d := newInviteDB(t)
	ctx := t.Context()
	id, err := d.repo.Create(ctx, auth.Account{Email: "cy@example.test", Role: auth.RolePractitioner, OrganizationID: clinicID, PasswordHash: "h"})
	if err != nil {
		t.Fatal(err)
	}
	var practitionerID string
	if err := d.owner.QueryRow(ctx, "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example') RETURNING id",
		clinicID).Scan(&practitionerID); err != nil {
		t.Fatal(err)
	}
	if err := d.repo.SetPractitioner(ctx, id, practitionerID); err != nil {
		t.Fatalf("SetPractitioner() = %v", err)
	}
	if err := d.repo.SetTOTPSecret(ctx, id, []byte("sealed")); err != nil {
		t.Fatal(err)
	}
	if err := d.repo.ConfirmMFA(ctx, id, 7, now); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if _, err := d.repo.CreateSession(ctx, auth.Session{AccountID: id, OrganizationID: clinicID, MFAVerified: true, ExpiresAt: now.Add(time.Hour)},
			[]byte{byte(i)}, now); err != nil {
			t.Fatal(err)
		}
	}

	if err := d.repo.ResetMFA(ctx, id); err != nil {
		t.Fatalf("ResetMFA() = %v", err)
	}
	if err := d.repo.RevokeSessions(ctx, id, now); err != nil {
		t.Fatalf("RevokeSessions() = %v", err)
	}
	var org, linked string
	var cleared bool
	var open int
	err = d.owner.QueryRow(ctx, `SELECT organization_id::text, practitioner_id::text,
		totp_secret IS NULL AND totp_last_step IS NULL AND mfa_enrolled_at IS NULL,
		(SELECT count(*) FROM session WHERE account_id = account.id AND revoked_at IS NULL)
		FROM account WHERE id = $1`, id).Scan(&org, &linked, &cleared, &open)
	if err != nil || org != clinicID || linked != practitionerID || !cleared || open != 0 {
		t.Errorf("account clinic %s, practitioner %s, MFA cleared %v, %d open sessions (%v); want the clinic, the link, cleared, none",
			org, linked, cleared, open, err)
	}
	for name, err := range map[string]error{
		"ResetMFA(unknown)":       d.repo.ResetMFA(ctx, "0b6c8a52-0000-4000-8000-000000000000"),
		"ResetMFA(not a UUID)":    d.repo.ResetMFA(ctx, "x"),
		"SetPractitioner(x)":      d.repo.SetPractitioner(ctx, id, "x"),
		"SetPractitioner(x, ...)": d.repo.SetPractitioner(ctx, "x", practitionerID),
		"RevokeSessions(x)":       d.repo.RevokeSessions(ctx, "x", now),
	} {
		if !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("%s = %v, want ErrNotFound", name, err)
		}
	}
}
