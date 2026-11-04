// Package postgres connects to Postgres and runs work in transactions.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

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

// passwordAttempts bounds the retries of SetPassword; each waits up to passwordBackoff times its number,
// with jitter so racing callers spread out.
const (
	passwordAttempts = 10
	passwordBackoff  = 10 * time.Millisecond
)

// SetPassword sets role's password. The statement is built by the server's format(), so the password is
// quoted as a literal, never spliced. Roles are cluster-wide: an ALTER ROLE that races another one on the
// same role fails with "tuple concurrently updated", so it is retried. Errors never carry the password.
func SetPassword(ctx context.Context, db Querier, role, password string) error {
	var stmt string
	err := db.QueryRow(ctx, "SELECT format('ALTER ROLE %I PASSWORD %L', $1::text, $2::text)", role, password).Scan(&stmt)
	if err != nil {
		return fmt.Errorf("postgres: set password of %s: %w", role, err)
	}
	for attempt := range passwordAttempts {
		if _, err = db.Exec(ctx, stmt); !concurrentlyUpdated(err) {
			break
		}
		time.Sleep(rand.N(time.Duration(attempt+1) * passwordBackoff))
	}
	if err != nil {
		return fmt.Errorf("postgres: set password of %s: %w", role, err)
	}
	return nil
}

func concurrentlyUpdated(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "XX000" && pgErr.Message == "tuple concurrently updated"
}
