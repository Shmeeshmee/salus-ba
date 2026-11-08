package auth_test

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// nopAuditor records nothing; safe for concurrent use.
type nopAuditor struct{}

func (nopAuditor) Record(context.Context, auditevent.AuditEvent) error { return nil }

// nopCounters count nothing, so no login is limited; safe for concurrent use.
type nopCounters struct{}

func (nopCounters) Incr(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, nil
}

func (nopCounters) Get(context.Context, string) (int64, time.Duration, error) { return 0, 0, nil }

// openPool opens a pool on a fresh test database with at most conns connections.
func openPool(b *testing.B, conns int) *pgxpool.Pool {
	b.Helper()
	u, err := url.Parse(pgtest.NewDatabaseURL(b))
	if err != nil {
		b.Fatal(err)
	}
	q := u.Query()
	q.Set("pool_max_conns", fmt.Sprint(conns))
	u.RawQuery = q.Encode()
	pool, err := postgres.Open(b.Context(), u.String())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(pool.Close)
	return pool
}

// BenchmarkLoginBurst sends a burst of 32 patient logins (each opens a session) (real argon2id, MaxConcurrentHashes slots) at a pool of 4
// connections (the pgxpool default on a 4-vCPU host) and of 16, while a probe makes one indexed read after another, as
// every other request does. It reports how long the probe waited (p50, p99, max) and how long the burst
// took (docs/accounts/PROGRESS.md, package 3).
func BenchmarkLoginBurst(b *testing.B) {
	for _, conns := range []int{4, 16} {
		b.Run(fmt.Sprintf("conns=%d", conns), func(b *testing.B) { benchmarkLoginBurst(b, conns) })
	}
}

func benchmarkLoginBurst(b *testing.B, conns int) {
	const logins = 32
	pool := openPool(b, conns)
	hasher := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), auth.MaxConcurrentHashes)
	hash, err := hasher.Hash(b.Context(), password)
	if err != nil {
		b.Fatal(err)
	}
	var probeID string
	for i := range logins {
		err := pool.QueryRow(b.Context(), `INSERT INTO account (email, password_hash, role) VALUES ($1, $2, 'patient') RETURNING id`,
			fmt.Sprintf("user%d@example.test", i), hash).Scan(&probeID)
		if err != nil {
			b.Fatal(err)
		}
	}
	repo := auth.NewPostgresRepository(pool)
	svc := auth.NewService(auth.Ports{
		Repo: repo, Hasher: hasher, Audit: nopAuditor{}, Tx: postgres.NewTransactor(pool), Clock: clock.System{},
		Counters: nopCounters{}, RateLimitKey: []byte("k"),
	})

	var waits []time.Duration
	var burst time.Duration
	for b.Loop() {
		ctx, stop := context.WithCancel(b.Context())
		var mu sync.Mutex
		var probes sync.WaitGroup
		ticking := make(chan struct{})
		go func() { // one read arrives every 5 ms while the burst runs, as other requests would
			defer close(ticking)
			tick := time.NewTicker(5 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
				probes.Go(func() {
					start := clock.System{}.Now()
					if _, err := repo.ByID(b.Context(), probeID); err != nil {
						b.Error(err)
					}
					mu.Lock()
					waits = append(waits, time.Since(start))
					mu.Unlock()
				})
			}
		}()
		start := clock.System{}.Now()
		var wg sync.WaitGroup
		for i := range logins {
			wg.Go(func() {
				if _, err := svc.Login(ctx, fmt.Sprintf("user%d@example.test", i), password, clientIP, "", ""); err != nil {
					b.Error(err)
				}
			})
		}
		wg.Wait()
		burst += time.Since(start)
		stop()
		<-ticking
		probes.Wait()
	}
	slices.Sort(waits)
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	b.ReportMetric(ms(waits[len(waits)/2]), "probe-p50-ms")
	b.ReportMetric(ms(waits[len(waits)*99/100]), "probe-p99-ms")
	b.ReportMetric(ms(waits[len(waits)-1]), "probe-max-ms")
	b.ReportMetric(ms(burst)/float64(b.N), "burst-ms")
	b.ReportMetric(0, "ns/op")
}

// BenchmarkAuthenticate is the per-request session check: one read of the session row by token hash on the
// unique index, as salus_app, among 10 000 sessions (docs/accounts/PROGRESS.md, package 3).
func BenchmarkAuthenticate(b *testing.B) {
	_, app := pgtest.NewAppDatabase(b)
	svc := auth.NewService(auth.Ports{
		Repo: auth.NewPostgresRepository(app), Hasher: auth.NewArgon2Hasher(auth.DefaultArgon2Params(), 1), Audit: nopAuditor{},
		Tx: postgres.NewTransactor(app), Clock: clock.System{}, Counters: nopCounters{}, RateLimitKey: []byte("k"),
	})
	if _, err := svc.Register(b.Context(), "ana@example.test", password, "", ""); err != nil {
		b.Fatal(err)
	}
	_, err := app.Exec(b.Context(), `INSERT INTO session (account_id, mfa_verified, token_hash, expires_at)
		SELECT id, false, sha256(i::text::bytea), now() + interval '1 day' FROM account, generate_series(1, 10000) i`)
	if err != nil {
		b.Fatal(err)
	}
	res, err := svc.Login(b.Context(), "ana@example.test", password, clientIP, "", "")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("serial", func(b *testing.B) {
		for b.Loop() {
			if _, err := svc.Authenticate(b.Context(), res.Token); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := svc.Authenticate(b.Context(), res.Token); err != nil {
					b.Error(err)
				}
			}
		})
	})
}
