package postgres_test

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, nil) }

// newItemsDB returns a fresh database with one table to write to.
func newItemsDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := pgtest.NewDatabase(t)
	if _, err := pool.Exec(t.Context(), "CREATE TABLE item (name text NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func insertItem(ctx context.Context, db postgres.Querier, name string) error {
	_, err := postgres.Conn(ctx, db).Exec(ctx, "INSERT INTO item (name) VALUES ($1)", name)
	return err
}

func countItems(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM item").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestOpen_ConnectsAndPings(t *testing.T) {
	t.Parallel()
	url := pgtest.NewDatabaseURL(t)

	pool, err := postgres.Open(t.Context(), url)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	pool.Close()
}

func TestOpen_FailsWhenServerIsUnreachable(t *testing.T) {
	t.Parallel()

	_, err := postgres.Open(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err == nil {
		t.Fatal("Open() error = nil, want a connection error")
	}
}

func TestOpen_RejectsAMalformedURL(t *testing.T) {
	t.Parallel()

	if _, err := postgres.Open(t.Context(), "postgres://%zz"); err == nil {
		t.Fatal("Open() error = nil, want a parse error")
	}
}

func TestWithTx_CommitsWhenFnSucceeds(t *testing.T) {
	t.Parallel()
	pool := newItemsDB(t)

	err := postgres.NewTransactor(pool).WithTx(t.Context(), func(ctx context.Context) error {
		return insertItem(ctx, pool, "a")
	})
	if err != nil {
		t.Fatalf("WithTx() error = %v", err)
	}
	if n := countItems(t, pool); n != 1 {
		t.Fatalf("items = %d, want 1", n)
	}
}

func TestWithTx_RollsBackAndReturnsFnError(t *testing.T) {
	t.Parallel()
	pool := newItemsDB(t)
	errStop := errors.New("stop")

	err := postgres.NewTransactor(pool).WithTx(t.Context(), func(ctx context.Context) error {
		if err := insertItem(ctx, pool, "a"); err != nil {
			return err
		}
		return errStop
	})
	if !errors.Is(err, errStop) {
		t.Fatalf("WithTx() error = %v, want %v", err, errStop)
	}
	if n := countItems(t, pool); n != 0 {
		t.Fatalf("items = %d, want 0 after rollback", n)
	}
}

func TestWithTx_RollsBackAndRepanicsOnPanic(t *testing.T) {
	t.Parallel()
	pool := newItemsDB(t)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("WithTx swallowed the panic")
			}
		}()
		_ = postgres.NewTransactor(pool).WithTx(t.Context(), func(ctx context.Context) error {
			if err := insertItem(ctx, pool, "a"); err != nil {
				return err
			}
			panic("boom")
		})
	}()
	if n := countItems(t, pool); n != 0 {
		t.Fatalf("items = %d, want 0 after rollback", n)
	}
}

func TestWithTx_NestedCallJoinsOuterTransaction(t *testing.T) {
	t.Parallel()
	pool := newItemsDB(t)
	tr := postgres.NewTransactor(pool)
	errStop := errors.New("stop")

	err := tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := insertItem(ctx, pool, "outer"); err != nil {
			return err
		}
		if err := tr.WithTx(ctx, func(ctx context.Context) error { return insertItem(ctx, pool, "inner") }); err != nil {
			return err
		}
		return errStop
	})
	if !errors.Is(err, errStop) {
		t.Fatalf("WithTx() error = %v, want %v", err, errStop)
	}
	if n := countItems(t, pool); n != 0 {
		t.Fatalf("items = %d, want 0: the inner write must roll back with the outer transaction", n)
	}
}

func TestConn_UsesThePoolOutsideATransaction(t *testing.T) {
	t.Parallel()
	pool := newItemsDB(t)

	if err := insertItem(t.Context(), pool, "a"); err != nil {
		t.Fatalf("insert outside a transaction: %v", err)
	}
	if n := countItems(t, pool); n != 1 {
		t.Fatalf("items = %d, want 1", n)
	}
}

// setting reads a custom setting in the transaction in ctx, or on the pool outside one; "" when unset.
func setting(ctx context.Context, t *testing.T, db postgres.Querier, name string) string {
	t.Helper()
	var v string
	if err := postgres.Conn(ctx, db).QueryRow(ctx, "SELECT coalesce(current_setting($1, true), '')", name).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSetLocal_IsVisibleOnlyInsideItsTransaction(t *testing.T) {
	t.Parallel()
	// One connection, so the read after the transaction lands on the connection that ran it.
	pool, err := postgres.Open(t.Context(), pgtest.NewDatabaseURL(t)+"&pool_max_conns=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tr := postgres.NewTransactor(pool)

	err = tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", "4b1f7c1e-0000-4000-8000-000000000001"); err != nil {
			return err
		}
		if got := setting(ctx, t, pool, "app.organization_id"); got != "4b1f7c1e-0000-4000-8000-000000000001" {
			t.Errorf("inside the transaction: setting = %q", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx() error = %v", err)
	}
	if got := setting(t.Context(), t, pool, "app.organization_id"); got != "" {
		t.Errorf("after the transaction: setting = %q, want it unset", got)
	}
}

func TestSetLocal_FailsOutsideATransaction(t *testing.T) {
	t.Parallel()

	if err := postgres.SetLocal(t.Context(), "app.organization_id", "x"); !errors.Is(err, postgres.ErrNoTx) {
		t.Fatalf("SetLocal() error = %v, want ErrNoTx", err)
	}
}

func TestSetLocal_ReturnsTheDatabaseError(t *testing.T) {
	t.Parallel()
	pool := pgtest.NewDatabase(t)

	err := postgres.NewTransactor(pool).WithTx(t.Context(), func(ctx context.Context) error {
		return postgres.SetLocal(ctx, "no dot is not a custom setting", "x")
	})
	if err == nil {
		t.Fatal("SetLocal() error = nil, want the database's error for an invalid name")
	}
}

// newRole creates a LOGIN role without a password, dropped when the test ends. Roles are cluster-wide, so
// the name is unique.
func newRole(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	role := "pgtest_role_" + strings.ToLower(rand.Text())
	if _, err := pool.Exec(t.Context(), "CREATE ROLE "+role+" LOGIN"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), "DROP ROLE "+role); err != nil {
			t.Errorf("drop role: %v", err)
		}
	})
	return role
}

// loginAs connects to dbURL as role with password and pings.
func loginAs(t *testing.T, dbURL, role, password string) error {
	t.Helper()
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(role, password)
	pool, err := postgres.Open(t.Context(), u.String())
	if err == nil {
		pool.Close()
	}
	return err
}

func TestSetPassword_LetsTheRoleLogIn(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)
	pool, err := postgres.Open(t.Context(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close) // after newRole's cleanup drops the role
	role := newRole(t, pool)
	const password = "synthetic-pa$$'word" // a quote and a dollar sign: the value is quoted, never spliced

	if err := postgres.SetPassword(t.Context(), pool, role, password); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	if err := loginAs(t, dbURL, role, password); err != nil {
		t.Fatalf("login with the new password: %v", err)
	}
	if err := loginAs(t, dbURL, role, "wrong"); err == nil {
		t.Fatal("login with a wrong password succeeded")
	}
}

// Roles are cluster-wide, and parallel test packages set the same role's password at once.
func TestSetPassword_ConcurrentCallsAllSucceed(t *testing.T) {
	t.Parallel()
	pool := pgtest.NewDatabase(t)
	role := newRole(t, pool)
	errs := make(chan error, 8)

	for range cap(errs) {
		go func() { errs <- postgres.SetPassword(t.Context(), pool, role, "synthetic") }()
	}
	for range cap(errs) {
		if err := <-errs; err != nil {
			t.Errorf("SetPassword() error = %v", err)
		}
	}
}

// Every test package sets the app roles' passwords when it starts, so on a large host forty callers can meet
// on one role at once, each on its own connection; none may fail (seen: "tuple concurrently updated" after the
// retries ran out, in the terminology 3.1 integration step).
func TestSetPassword_ManyCallersOnTheirOwnConnectionsAllSucceed(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewDatabaseURL(t)
	const callers = 48
	pool, err := postgres.Open(t.Context(), dbURL+"&pool_max_conns="+strconv.Itoa(callers))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	role := newRole(t, pool)
	for round := range 5 {
		start := make(chan struct{})
		errs := make(chan error, callers)
		for range callers {
			go func() {
				<-start
				errs <- postgres.SetPassword(t.Context(), pool, role, "synthetic")
			}()
		}
		close(start)
		for range callers {
			if err := <-errs; err != nil {
				t.Fatalf("round %d: SetPassword() error = %v", round, err)
			}
		}
	}
}

// An advisory lock belongs to one database and a role to the cluster, so callers on different databases (each
// test package's, cmd/migrate's own) are not ordered by the lock and meet on the role's row; none may fail (seen:
// "tuple concurrently updated", SQLSTATE XX000, in cmd/migrate, cmd/notifier and internal/schedule).
func TestSetPassword_CallersOnDifferentDatabasesAllSucceed(t *testing.T) {
	t.Parallel()
	const databases = 8
	pools := make([]*pgxpool.Pool, databases)
	for i := range pools {
		pools[i] = pgtest.NewDatabase(t)
	}
	role := newRole(t, pools[0])
	for round := range 20 {
		start := make(chan struct{})
		errs := make(chan error, databases)
		for _, pool := range pools {
			go func() {
				<-start
				errs <- postgres.SetPassword(t.Context(), pool, role, "synthetic")
			}()
		}
		close(start)
		for range databases {
			if err := <-errs; err != nil {
				t.Fatalf("round %d: SetPassword() error = %v", round, err)
			}
		}
	}
}

func TestSetPassword_ErrorDoesNotEchoThePassword(t *testing.T) {
	t.Parallel()
	pool := pgtest.NewDatabase(t)

	err := postgres.SetPassword(t.Context(), pool, "no_such_role_anywhere", "secret-looking-value")
	if err == nil {
		t.Fatal("SetPassword() error = nil for an unknown role")
	}
	if strings.Contains(err.Error(), "secret-looking-value") {
		t.Errorf("error %q leaks the password", err)
	}
}

// A library that takes the transaction itself (River's InsertTx) gets the one WithTx carries.
func TestTx_ReturnsTheTransactionOfWithTxOnly(t *testing.T) {
	pool := newItemsDB(t)
	if _, ok := postgres.Tx(t.Context()); ok {
		t.Error("Tx() outside a transaction: ok = true")
	}
	err := postgres.NewTransactor(pool).WithTx(t.Context(), func(ctx context.Context) error {
		tx, ok := postgres.Tx(ctx)
		if !ok {
			t.Fatal("Tx() inside WithTx: ok = false")
		}
		if _, err := tx.Exec(ctx, "INSERT INTO item (name) VALUES ('through the tx')"); err != nil {
			return err
		}
		return errors.New("roll back")
	})
	if err == nil {
		t.Fatal("WithTx() error = nil")
	}
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM item").Scan(&n); err != nil || n != 0 {
		t.Errorf("%d items after the rollback (%v), want 0: Tx must be the WithTx transaction", n, err)
	}
}
