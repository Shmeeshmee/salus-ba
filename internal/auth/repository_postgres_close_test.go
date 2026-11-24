package auth_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestPostgres_CloseReopenAndEraseAnAccount(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	svc := auth.NewService(postgresPorts(app, auth.NewArgon2Hasher(cheap, 1)))
	ctx := t.Context()
	a, err := svc.Register(ctx, "ana@example.test", password, "bs", "de")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(ctx, "ana@example.test", password, "192.0.2.1", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(ctx, a.ID, password, eraseAfter); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if n := count(t, owner, "SELECT count(*) FROM session WHERE revoked_at IS NULL"); n != 0 {
		t.Errorf("%d open sessions after closure, want 0", n)
	}
	if _, err := svc.Login(ctx, "ana@example.test", password, "192.0.2.1", "", ""); !errors.Is(err, auth.ErrWrongCredentials) {
		t.Errorf("Login() after closure error = %v, want ErrWrongCredentials", err)
	}
	if err := svc.Erase(ctx, a.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("Erase() before the date error = %v, want ErrNotFound", err)
	}
	if err := svc.Reopen(ctx, a.ID); err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}
	if err := svc.Close(ctx, a.ID, password, eraseAfter); err != nil {
		t.Fatalf("Close() again error = %v", err)
	}

	ports := postgresPorts(app, auth.NewArgon2Hasher(cheap, 1))
	ports.Clock = clock.Fixed{T: now.Add(eraseAfter)}
	later := auth.NewService(ports)
	ids, err := later.DueForErasure(ctx)
	if err != nil || len(ids) != 1 || ids[0] != a.ID {
		t.Fatalf("DueForErasure() = %v, %v; want [%s]", ids, err, a.ID)
	}
	if err := later.Erase(ctx, a.ID); err != nil {
		t.Fatalf("Erase() error = %v", err)
	}
	if n := count(t, owner, "SELECT count(*) FROM account WHERE email = id::text || '@erased.invalid' AND password_hash = '' "+
		"AND totp_secret IS NULL AND mfa_enrolled_at IS NULL AND language IS NULL AND detected_language IS NULL AND erased_at IS NOT NULL"); n != 1 {
		t.Errorf("%d erased accounts, want 1", n)
	}
	if n := count(t, owner, "SELECT count(*) FROM account WHERE email LIKE '%example.test%'"); n != 0 {
		t.Errorf("the e-mail is still stored on %d accounts", n)
	}
	if ids, _ := later.DueForErasure(ctx); len(ids) != 0 {
		t.Errorf("DueForErasure() after erasure = %v, want none", ids)
	}
	if err := later.Reopen(ctx, a.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("Reopen() after erasure error = %v, want ErrNotFound", err)
	}
}
