package pgtest

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// LastQuery is a pgx tracer that keeps the last statement sent through its pool whose text contains Match, with its
// arguments: a test runs a repository's call and then asks how Postgres planned the statement it sent (RowsDropped).
type LastQuery struct {
	Match string
	sql   string
	args  []any
	mu    sync.Mutex
}

// TraceQueryStart keeps the statement when it matches.
func (q *LastQuery) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if strings.Contains(d.SQL, q.Match) && !strings.HasPrefix(d.SQL, "EXPLAIN") {
		q.mu.Lock()
		q.sql, q.args = d.SQL, d.Args
		q.mu.Unlock()
	}
	return ctx
}

// TraceQueryEnd does nothing.
func (*LastQuery) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TracedPool connects to url with tracer, closed when the test ends.
func TracedPool(t testing.TB, url string, tracer pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Plan is what EXPLAIN ANALYZE of the statement q kept tells: the rows its plan read and dropped (every node's "Rows
// Removed by Filter" and "Rows Removed by Index Recheck") and the rows an INSERT … ON CONFLICT met as conflicts. It runs
// in the transaction and scope of ctx on db; an INSERT or DELETE is carried out, as EXPLAIN ANALYZE does.
type Plan struct{ Dropped, Conflicts float64 }

// Explain runs EXPLAIN ANALYZE of the statement q kept and returns its Plan.
func Explain(t testing.TB, ctx context.Context, db postgres.Querier, q *LastQuery) Plan {
	t.Helper()
	q.mu.Lock()
	sql, args := q.sql, q.args
	q.mu.Unlock()
	if sql == "" {
		t.Fatalf("pgtest: no statement matching %q was sent", q.Match)
	}
	var raw []byte
	if err := postgres.Conn(ctx, db).QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+sql, args...).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &plans); err != nil || len(plans) != 1 {
		t.Fatalf("pgtest: EXPLAIN gave %s, %v", raw, err)
	}
	var p Plan
	var walk func(node map[string]any)
	walk = func(node map[string]any) {
		for _, key := range []string{"Rows Removed by Filter", "Rows Removed by Index Recheck"} {
			n, _ := node[key].(float64)
			p.Dropped += n
		}
		c, _ := node["Conflicting Tuples"].(float64)
		p.Conflicts += c
		children, _ := node["Plans"].([]any)
		for _, child := range children {
			if m, ok := child.(map[string]any); ok {
				walk(m)
			}
		}
	}
	walk(plans[0].Plan)
	return p
}
