package postgres_test

import (
	"context"
	"errors"
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
