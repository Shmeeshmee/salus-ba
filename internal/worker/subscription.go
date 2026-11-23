package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// subscriptionHour is the UTC hour of the daily subscription_monthly job: after midnight in the clinics' zone, so the
// run of the 1st issues the month's invoices (docs/plans/clinic-cms.md, package D).
const subscriptionHour = 1

// subscriptionWorker issues the clinics' monthly invoices and moves a clinic that does not pay: every clinic's (the
// scheduler's daily run; an invoice is issued once per clinic and month), or the job's clinic. It logs counts only.
type subscriptionWorker struct {
	river.WorkerDefaults[subscription.MonthlyArgs]
	memberships *subscription.Service
	clock       clock.Clock
	log         *slog.Logger
}

func (w *subscriptionWorker) Work(ctx context.Context, job *river.Job[subscription.MonthlyArgs]) error {
	month := job.Args.Month
	if month.IsZero() {
		month = w.clock.Now()
	}
	if job.Args.OrganizationID == "" {
		issued, err := w.memberships.RunAll(ctx, month)
		w.log.InfoContext(ctx, "subscription invoices issued", "invoices", issued)
		return err
	}
	_, err := w.memberships.IssueMonthly(ctx, job.Args.OrganizationID, month)
	switch {
	case errors.Is(err, subscription.ErrNotFound):
		return river.JobCancel(err) // no membership: a retry would find none either
	case err != nil && !errors.Is(err, subscription.ErrNotBilled):
		return err
	}
	_, err = w.memberships.Review(ctx, job.Args.OrganizationID)
	return err
}

// newMemberships builds the membership service on pool (salus_app) at clk for the monthly job: it reads the clinic's
// time zone and counts its active roles through their services, and enqueues each invoice's e-mail with inserter.
func newMemberships(pool *pgxpool.Pool, inserter *river.Client[pgx.Tx], clk clock.Clock) *subscription.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk, nil)
	roles := practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	return subscription.NewService(subscription.Ports{
		Repo: subscription.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Clinics: clinics, Seats: roles,
		Jobs: subscription.NewRiverJobs(inserter),
	})
}

// invoiceEmailWorker sends a membership invoice to its clinic's billing e-mail with the PDF (package I). An invoice
// that is not there, or an address SMTP cannot carry, cancels the job: a retry fails the same way.
type invoiceEmailWorker struct {
	river.WorkerDefaults[subscription.InvoiceEmailArgs]
	memberships *subscription.Service
}

func (w *invoiceEmailWorker) Work(ctx context.Context, job *river.Job[subscription.InvoiceEmailArgs]) error {
	err := w.memberships.SendInvoice(ctx, job.Args.OrganizationID, job.Args.InvoiceID)
	if errors.Is(err, subscription.ErrNotFound) || errors.Is(err, notification.ErrRecipientNotASCII) {
		return river.JobCancel(err)
	}
	return err
}

// newInvoiceMail builds the membership service the notifier sends the invoices with: the clinic, its billing details,
// the seller and the mail.
func newInvoiceMail(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, clinics *organization.Service, mail *notification.Service, seller subscription.Seller) *subscription.Service {
	billing := organization.NewBilling(organization.BillingPorts{Repo: organization.NewPostgresBillingRepository(pool), Audit: audit, Tx: tx})
	return subscription.NewService(subscription.Ports{
		Repo: subscription.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics, Billing: billing, Mailer: mail, Seller: seller,
	})
}
