// Package postgres connects to Postgres and runs work in transactions.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier runs queries: the pool, or the transaction in the context. sqlc's DBTX has the same methods.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Open connects a pool to url and checks the connection.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}

type txKey struct{}

// Transactor runs functions in a transaction carried by the context.
type Transactor struct {
	pool *pgxpool.Pool
}

// NewTransactor returns a Transactor on pool.
func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

// WithTx runs fn in a transaction: commit when fn returns nil, roll back when it returns an error or panics.
// Called inside another WithTx, it joins the outer transaction.
func (t *Transactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	// Roll back even when ctx is cancelled. After a commit, Rollback is a no-op.
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

// Conn returns the transaction in ctx, or db when there is none.
func Conn(ctx context.Context, db Querier) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db
}

// Tx returns the transaction of the WithTx in ctx, for a library that takes the transaction itself (River's
// InsertTx), and false outside one.
func Tx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// ErrNoTx is returned by SetLocal outside a transaction, where a local setting would not hold.
var ErrNoTx = errors.New("postgres: SetLocal needs the transaction of a WithTx")

// SetLocal sets the custom setting name (such as app.organization_id) for the rest of the transaction in
// ctx only: set_config(name, value, true). Errors name the setting, never the value.
func SetLocal(ctx context.Context, name, value string) error {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	if !ok {
		return ErrNoTx
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", name, value); err != nil {
		return fmt.Errorf("postgres: set %s: %w", name, err)
	}
	return nil
}

// RoleLock names the advisory lock SetPassword takes (its key is hashtext(RoleLock)). An advisory lock belongs
// to one database while roles belong to the cluster, so it orders only the callers on the same database; a
// caller on another database holds RoleLock on the shared one for as long as it sets passwords
// (pgtest.HoldRoleLock).
const RoleLock = "salus.postgres.SetPassword"

// SetPassword sets role's password. The statement is built by the server's format(), so the password is
// quoted as a literal, never spliced. Roles are cluster-wide, and two ALTER ROLEs on one role at once fail with
// "tuple concurrently updated" (every test package sets the app roles' passwords when it starts), so the
// statement runs in one DO block behind a transaction-scoped advisory lock on RoleLock that every SetPassword
// takes: callers on one database wait their turn instead of failing. Errors never carry the password.
func SetPassword(ctx context.Context, db Querier, role, password string) error {
	var stmt string
	err := db.QueryRow(ctx, `SELECT format('DO %L', format(
		'BEGIN PERFORM pg_advisory_xact_lock(hashtext(%L)); ALTER ROLE %I PASSWORD %L; END',
		$3::text, $1::text, $2::text))`, role, password, RoleLock).Scan(&stmt)
	if err == nil {
		_, err = db.Exec(ctx, stmt)
	}
	if err != nil {
		return fmt.Errorf("postgres: set password of %s: %w", role, err)
	}
	return nil
}
