package worker

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
	"github.com/Shmeeshmee/salus/pkg/fiscal/fiscaltest"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// invoiceFiscalizeWorker reports a receipt to the tax authority (invoice_fiscalize). A failure is retried by River
// with its error; when the last attempt fails the receipt is marked failed, which the clinic sees. It logs nothing.
type invoiceFiscalizeWorker struct {
	river.WorkerDefaults[invoice.FiscalizeArgs]
	receipts *invoice.Service
}

func (w *invoiceFiscalizeWorker) Work(ctx context.Context, job *river.Job[invoice.FiscalizeArgs]) error {
	err := w.receipts.Fiscalize(ctx, job.Args.OrganizationID, job.Args.InvoiceID)
	if err != nil && job.Attempt >= job.MaxAttempts {
		return errors.Join(err, w.receipts.GiveUp(ctx, job.Args.OrganizationID, job.Args.InvoiceID))
	}
	return err
}

// newReceipts builds the receipts' service for the fiscal job on pool (salus_app), reporting through the fake: the
// only engine until the ESET adapter exists (config.LoadFiscal; docs/plans/clinic-cms.md, package G). The fake keeps
// what it fiscalized in memory, for the life of the process.
func newReceipts(pool *pgxpool.Pool, clk clock.Clock) *invoice.Service {
	return newReceiptsOn(pool, &fiscaltest.Reporter{Clock: clk}, clk)
}

func newReceiptsOn(pool *pgxpool.Pool, reporter fiscal.Reporter, clk clock.Clock) *invoice.Service {
	tx := postgres.NewTransactor(pool)
	return invoice.NewService(invoice.Ports{
		Repo: invoice.NewPostgresRepository(pool), Reporter: reporter, Tx: tx, Clock: clk,
		Audit: auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk),
	})
}
