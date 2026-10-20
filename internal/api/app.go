// Package api builds the API binary's dependencies and handler: the shared composition root.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/api/router"
	"github.com/Shmeeshmee/salus/internal/config"
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

// New connects to the dependencies and builds the handler.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, err
	}
	h := router.New(router.Config{
		Logger: log,
		Checks: []router.Check{
			{Name: "postgres", Ping: pool.Ping},
			{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
		},
		Timeout: RequestTimeout,
	})
	return &App{Handler: h, pool: pool, redis: rdb}, nil
}

// Close releases the dependencies.
func (a *App) Close() {
	_ = a.redis.Close()
	a.pool.Close()
}
