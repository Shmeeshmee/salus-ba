package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// scanWorkers is how many files the scan queue scans at a time: clamd holds each stream it scans in memory
// (docs/attachments/SPEC.md §2 "Scan", measured in docs/attachments/PROGRESS.md 3.1).
const scanWorkers = 2

// scanTimeout bounds one scan job: the read of a 200 MB file from the store and clamd's scan of it, with a wide
// margin over the measured time (PROGRESS.md 3.1). River's default of a minute is not enough on a slow store.
const scanTimeout = 10 * time.Minute

// Files is what the binary jobs and binary's purger work with: the object store, the key-encryption key that
// wraps the files' data keys, and clamd.
type Files struct {
	Store   binary.Store
	KEK     *envelope.KEK
	Scanner binary.Scanner
}

func (f Files) check() error {
	if f.Store == nil || f.KEK == nil || f.Scanner == nil {
		return errors.New("worker: the binary jobs need the object store, the key-encryption key and clamd")
	}
	return nil
}

// newFiles builds the binary service on pool (salus_app) for the scan and discard jobs, with the questionnaire
// response as the files' holder.
func newFiles(pool *pgxpool.Pool, files Files, responses *questionnaireresponse.Service, clk clock.Clock) *binary.Service {
	tx := postgres.NewTransactor(pool)
	return binary.NewService(binary.Ports{
		Repo: binary.NewPostgresRepository(pool), Store: files.Store, Holder: responses,
		Audit: auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk), Tx: tx,
		Scanner: files.Scanner, Clock: clk, KEK: files.KEK,
	})
}

// binaryScanWorker scans an uploaded file, a questionnaire's or a chat's (binary_scan, enqueued with the upload on
// the scan queue). It logs the file's id, the result, for a rejected file the signature, and the time taken: never
// the title, and no patient id beside the signature.
type binaryScanWorker struct {
	river.WorkerDefaults[binary.ScanArgs]
	files *binary.Service
	clock clock.Clock
	log   *slog.Logger
}

func (w *binaryScanWorker) Timeout(*river.Job[binary.ScanArgs]) time.Duration { return scanTimeout }

// Work scans the file. A chat file's last attempt that fails rejects it as unscannable (binary.Service.Unscannable),
// so its message never shows it as scanning for ever; a questionnaire file's stays quarantined.
func (w *binaryScanWorker) Work(ctx context.Context, job *river.Job[binary.ScanArgs]) error {
	start := w.clock.Now()
	res, err := w.files.Scan(ctx, job.Args)
	if err != nil && job.Attempt >= job.MaxAttempts {
		if rejected, rerr := w.files.Unscannable(ctx, job.Args); rerr == nil && rejected {
			w.log.InfoContext(ctx, "binary scan done", "binary_id", job.Args.BinaryID, "result", "unscannable",
				"duration_ms", w.clock.Now().Sub(start).Milliseconds())
			return nil
		}
	}
	if err != nil {
		return err
	}
	attrs := []any{"binary_id", job.Args.BinaryID, "result", string(res.Status)}
	switch res.Status {
	case "":
		attrs[3] = "none"
	case binary.StatusRejected:
		attrs = append(attrs, "signature", res.Signature)
	}
	attrs = append(attrs, "duration_ms", w.clock.Now().Sub(start).Milliseconds())
	w.log.InfoContext(ctx, "binary scan done", attrs...)
	return nil
}

// binaryDiscardWorker deletes the files a response no longer lists (binary_discard, enqueued by the response's
// withdrawal or submit). It logs the response's id and the count, nothing else.
type binaryDiscardWorker struct {
	river.WorkerDefaults[binary.DiscardArgs]
	files *binary.Service
	log   *slog.Logger
}

func (w *binaryDiscardWorker) Work(ctx context.Context, job *river.Job[binary.DiscardArgs]) error {
	n, err := w.files.Discard(ctx, job.Args)
	if err != nil {
		return err
	}
	w.log.InfoContext(ctx, "binary discard done", "questionnaire_response_id", job.Args.ResponseID, "deleted", n)
	return nil
}
