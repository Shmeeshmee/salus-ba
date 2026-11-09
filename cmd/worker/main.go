// Command worker runs the River background jobs of the default queue and the scan queue, and the consultations'
// stream consumer (worker.ConsultStream, on REDIS_URL), which stores what the encounters' Redis Streams carry. It
// logs in as salus_app (DATABASE_URL), and as salus_retention (RETENTION_DATABASE_URL) for the retention_apply
// job alone; the binary jobs reach the object store and clamd (config.LoadObjectStore, config.LoadClamd).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the clinics' time zones, independent of the runtime image

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
)

// stopTimeout is how long running jobs get to finish after a stop signal.
const stopTimeout = 30 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logOut io.Writer) error {
	log := slog.New(slog.NewJSONHandler(logOut, nil))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Required, like the api (docs/consultation/SPEC.md §2 "Configuration"): the consultation's jobs end
	// rooms.
	lk, err := config.LoadLiveKit()
	if err != nil {
		return err
	}
	ret := config.LoadRetention()
	if ret.DatabaseURL == "" {
		return errors.New("config: RETENTION_DATABASE_URL is required")
	}
	files, err := loadFiles()
	if err != nil {
		return err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL) // the consultations' streams
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	retentionPool, err := postgres.Open(ctx, ret.DatabaseURL)
	if err != nil {
		return err
	}
	defer retentionPool.Close()
	client, err := worker.New(pool, retentionPool, videosession.NewLiveKitProvider(lk.URL, lk.APIKey, lk.APISecret), files, log)
	if err != nil {
		return err
	}
	// River stops hard when its start context ends; the signal triggers a soft Stop below instead.
	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	consumed := make(chan struct{})
	go func() {
		defer close(consumed)
		worker.NewConsultStream(pool, rdb, log).Run(ctx)
	}()
	log.Info("started")

	<-ctx.Done()
	log.Info("shutting down")
	<-consumed // it stops within one read; an entry it had not acknowledged is written again later
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

// s3PageSize is the object store listing's page for binary's purger, S3's largest.
const s3PageSize = 1000

// loadFiles builds the binary jobs' object store, key-encryption key and clamd client from the environment.
// The store's HTTP client has the upload's deadline as its timeout, as in the api: a scan reads a 200 MB object
// whole.
func loadFiles() (worker.Files, error) {
	store, err := config.LoadObjectStore()
	if err != nil {
		return worker.Files{}, err
	}
	scanner, err := config.LoadClamd()
	if err != nil {
		return worker.Files{}, err
	}
	kek, err := envelope.NewKEK(store.KEKID, store.KEK)
	if err != nil {
		return worker.Files{}, errors.New("worker: BINARY_KEK must be 32 bytes")
	}
	return worker.Files{
		Store: binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, store.Endpoint, store.Region, store.Bucket,
			store.AccessKeyID, store.SecretAccessKey, s3PageSize),
		KEK:     kek,
		Scanner: clamd.New(scanner.Addr),
	}, nil
}
