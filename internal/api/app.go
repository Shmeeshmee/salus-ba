// Package api builds the API binary's dependencies and handler: the shared composition root.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/api/router"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
)

// RequestTimeout bounds every request's handler.
const RequestTimeout = 30 * time.Second

// App is the built API.
type App struct {
	Handler http.Handler
	pool    *pgxpool.Pool
	redis   *goredis.Client
}

// minRateLimitKey is the shortest RATE_LIMIT_KEY accepted: as long as the HMAC-SHA256 output.
const minRateLimitKey = 32

// New connects to the dependencies and builds the handler.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	box, err := auth.NewAESSecretBox(cfg.TOTPKey)
	if err != nil {
		return nil, err
	}
	if len(cfg.RateLimitKey) < minRateLimitKey {
		return nil, errors.New("api: RATE_LIMIT_KEY must be at least 32 bytes")
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, err
	}
	jobs, err := worker.NewInserter(pool) // invites enqueue their e-mail in their transaction
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	hasher := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), auth.MaxConcurrentHashes)
	accounts := auth.NewService(auth.Ports{
		Repo: auth.NewPostgresRepository(pool), Hasher: hasher, Audit: audit, Tx: tx, Clock: clock.System{},
		Challenges: auth.NewRedisChallengeStore(rdb), Counters: auth.NewRedisCounters(rdb), Box: box,
		Jobs:              auth.NewRiverJobs(jobs),
		Practitioners:     practitioner.NewService(practitioner.NewPostgresRepository(pool), audit, tx),
		PractitionerRoles: practitionerrole.NewService(practitionerrole.NewPostgresRepository(pool), audit, tx),
		RateLimitKey:      cfg.RateLimitKey,
	})
	authHandler := auth.NewHandler(accounts, log, cfg.TrustedProxies)

	h := router.New(router.Config{
		Logger: log,
		Checks: []router.Check{
			{Name: "postgres", Ping: pool.Ping},
			{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
		},
		Session: authHandler.Session,
		Routes:  []func(*http.ServeMux){authHandler.Routes},
		Timeout: RequestTimeout,
	})
	return &App{Handler: h, pool: pool, redis: rdb}, nil
}

// Close releases the dependencies.
func (a *App) Close() {
	_ = a.redis.Close()
	a.pool.Close()
}
