//go:build loadtest

package load_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// TestLoad is WAVE7.md RD2's load run (make load, scripts/load.sh): on the load stack's own database, refused
// anywhere else (guard), it seeds the busy year once, then drives every hot path through the api and the workers and
// prints p50, p95 and p99 of each with the slowest statements of pg_stat_statements. LOAD_REQUESTS (500) requests a
// path, LOAD_CONCURRENCY (8) at a time; LOAD_OUT names a file the readings are also written to.
func TestLoad(t *testing.T) {
	ctx := t.Context()
	owner := openPool(t, mustEnv(t, "LOAD_DATABASE_URL"))
	if err := guard(ctx, owner); err != nil {
		t.Fatal(err)
	}
	// Only the load stack preloads it (docker-compose.load.yml); no migration adds it.
	if _, err := owner.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_stat_statements`); err != nil {
		t.Fatal(err)
	}
	tg := target{
		owner: owner, app: openPool(t, mustEnv(t, "LOAD_APP_DATABASE_URL")), stats: openPool(t, mustEnv(t, "LOAD_STATS_DATABASE_URL")),
		appURL: mustEnv(t, "LOAD_APP_DATABASE_URL"), redisURL: mustEnv(t, "LOAD_REDIS_URL"),
	}
	var clinics int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM organization`).Scan(&clinics); err != nil {
		t.Fatal(err)
	}
	if clinics == 0 {
		began := wall.Now()
		if err := seed(ctx, owner, tg.stats, busyYear); err != nil {
			t.Fatal(err)
		}
		t.Logf("seeded the busy year in %s", wall.Now().Sub(began).Round(time.Second))
	}
	if os.Getenv("LOAD_SEED_ONLY") != "" {
		return
	}
	lines, err := drive(ctx, tg, envInt("LOAD_REQUESTS", 500), envInt("LOAD_CONCURRENCY", 8), 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.Join(lines, "\n") + "\n"
	t.Log("\n" + out)
	if f := os.Getenv("LOAD_OUT"); f != "" {
		if err := os.WriteFile(f, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// drive measures every path on tg: the api's paths n requests each, conc at a time, after a warm-up of each; the slot
// generation three runs; the reminders' sweep and dispatch of reminders reminders. With statements, it resets
// pg_stat_statements after the warm-up and appends the slowest statements. It returns the readings as Markdown
// lines.
func drive(ctx context.Context, tg target, n, conc, reminders int, statements bool) ([]string, error) {
	s, err := loadSeeded(ctx, tg.owner, 200)
	if err != nil {
		return nil, err
	}
	if err := openSessions(ctx, tg.owner, s); err != nil {
		return nil, fmt.Errorf("sessions: %w", err)
	}
	srv, stop, err := newServer(ctx, tg)
	if err != nil {
		return nil, err
	}
	defer stop()
	client := &http.Client{Timeout: time.Minute, Transport: &http.Transport{MaxIdleConnsPerHost: conc}}
	slots, err := freeSlots(ctx, tg.owner, s, n+n/10)
	if err != nil || len(slots) < n+n/10 {
		return nil, fmt.Errorf("free slots: %d, %w", len(slots), err)
	}
	now := wall.Now()
	paths := httpPaths(s, slots, now)
	for _, p := range paths {
		warm := p
		if strings.HasPrefix(p.name, "booking: book") { // the warm-up books the slots after the measured ones
			warm.next = func(i int) request { return p.next(n + i) }
		}
		measureHTTP(ctx, client, srv.URL, warm, max(n/10, 5), conc)
	}
	if statements {
		if _, err := tg.owner.Exec(ctx, `SELECT pg_stat_statements_reset()`); err != nil {
			return nil, err
		}
	}
	lines := []string{fmt.Sprintf("Readings at %s, %d requests a path, %d at a time (ms):", now.UTC().Format(time.RFC3339), n, conc), "",
		"| Path | Requests | p50 | p95 | p99 | max | not as wanted | statements a request | database ms a request |",
		"| --- | --- | --- | --- | --- | --- | --- | --- | --- |"}
	for _, p := range paths {
		calls, ms, err := totals(ctx, tg.owner, statements)
		if err != nil {
			return nil, err
		}
		r := measureHTTP(ctx, client, srv.URL, p, n, conc)
		calls2, ms2, err := totals(ctx, tg.owner, statements)
		if err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s %.1f | %.2f |", r, (calls2-calls)/float64(n), (ms2-ms)/float64(n)))
	}
	sl, err := measureSlots(ctx, tg, 3)
	if err != nil {
		return nil, fmt.Errorf("slots: %w", err)
	}
	lines = append(lines, sl.String())
	rl, _, err := reminderReadings(ctx, tg, reminders)
	if err != nil {
		return nil, err
	}
	lines = append(lines, rl[1:]...)
	lines = append(lines, "", rl[0])
	if statements {
		st, err := slowest(ctx, tg.owner)
		if err != nil {
			return nil, err
		}
		lines = append(lines, st...)
	}
	return lines, nil
}

// totals are pg_stat_statements' calls and execution time so far in the database (0 without statements); the
// driver's own reads of it are left out.
func totals(ctx context.Context, owner *pgxpool.Pool, statements bool) (calls, ms float64, err error) {
	if !statements {
		return 0, 0, nil
	}
	err = owner.QueryRow(ctx, `SELECT coalesce(sum(calls), 0)::float8, coalesce(sum(total_exec_time), 0)::float8 FROM pg_stat_statements
		WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database()) AND query NOT LIKE '%pg_stat_statements%'`).Scan(&calls, &ms)
	return calls, ms, err
}

// slowest is pg_stat_statements' statements of the run: the 25 with the most time in all, and the 15 slowest by
// mean of those called at least 10 times. Its texts are normalised (every value a $n): each is shown by its start
// and its FROM clause.
func slowest(ctx context.Context, owner *pgxpool.Pool) ([]string, error) {
	out := []string{}
	for _, q := range []struct{ title, order string }{
		{"Statements by total time", "total_exec_time DESC LIMIT 25"},
		{"Statements by mean time (10 calls or more)", "mean_exec_time DESC LIMIT 15"},
	} {
		rows, err := owner.Query(ctx, `SELECT calls, round(total_exec_time::numeric, 1), round(mean_exec_time::numeric, 2),
				round(rows::numeric / greatest(calls, 1), 1), left(regexp_replace(query, '\s+', ' ', 'g'), 60) || coalesce(' … ' || substring(regexp_replace(query, '\s+', ' ', 'g') FROM ' (FROM [a-z_]+ .{0,90})'), '')
			FROM pg_stat_statements WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database()) AND calls >= 10
				AND query NOT LIKE '%pg_stat_statements%'
			ORDER BY `+q.order)
		if err != nil {
			return nil, err
		}
		out = append(out, "", q.title+":", "", "| Calls | Total ms | Mean ms | Rows a call | Statement |", "| --- | --- | --- | --- | --- |")
		for rows.Next() {
			var calls int64
			var total, mean, perCall float64
			var text string
			if err := rows.Scan(&calls, &total, &mean, &perCall, &text); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, fmt.Sprintf("| %d | %.1f | %.2f | %.1f | `%s` |", calls, total, mean, perCall, strings.ReplaceAll(text, "|", "/")))
		}
		rows.Close()
	}
	return out, nil
}

func mustEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set; run the load with make load", name)
	}
	return v
}

func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return def
}

func openPool(t testing.TB, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
