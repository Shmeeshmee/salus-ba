package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal/fiscaltest"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The invoice_fiscalize job against the database: run twice it reports the receipt once and stores the answer with
// an audit event of the job; while the authority is down the receipt stays pending, the last failed attempt marks
// it failed, and a later run still fiscalizes it.
func TestInvoiceFiscalizeWorker_ReportsOnceAndGivesUpOnTheLastAttempt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	org, account := seedReceiptClinic(t, owner)
	at := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	clk := clock.Fixed{T: at}
	issue := func() string { return issueReceipt(t, app, org, account, at) }
	status := func(id string) (s string, qr *string) {
		if err := owner.QueryRow(ctx, `SELECT fiscal_status, fiscal_qr FROM invoice WHERE id = $1`, id).Scan(&s, &qr); err != nil {
			t.Fatal(err)
		}
		return s, qr
	}
	reporter := &fiscaltest.Reporter{Clock: clk}
	w := &invoiceFiscalizeWorker{receipts: newReceiptsOn(app, reporter, clk)}
	job := func(id string, attempt int) *river.Job[invoice.FiscalizeArgs] {
		return &river.Job[invoice.FiscalizeArgs]{
			JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: 3},
			Args:   invoice.FiscalizeArgs{InvoiceID: id, OrganizationID: org},
		}
	}

	first := issue()
	for attempt := 1; attempt <= 2; attempt++ {
		if err := w.Work(ctx, job(first, attempt)); err != nil {
			t.Fatalf("Work() = %v", err)
		}
	}
	if s, qr := status(first); s != "fiscalized" || qr == nil || reporter.Reported(first) != 1 {
		t.Errorf("after two runs: %s, reported %d times; want fiscalized, once", s, reporter.Reported(first))
	}
	var events int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_name = $1 AND entity_reference = $2`,
		invoice.AgentFiscal, "Invoice/"+first).Scan(&events); err != nil || events != 1 {
		t.Errorf("audit events of the job = %d (%v), want 1", events, err)
	}

	second := issue()
	down := errors.New("synthetic outage")
	reporter.Err = down
	if err := w.Work(ctx, job(second, 2)); !errors.Is(err, down) {
		t.Fatalf("Work() while down = %v, want the outage", err)
	}
	if s, _ := status(second); s != "pending" {
		t.Errorf("while the job retries: %s, want pending", s)
	}
	if err := w.Work(ctx, job(second, 3)); !errors.Is(err, down) {
		t.Fatalf("the last Work() = %v, want the outage", err)
	}
	if s, _ := status(second); s != "failed" {
		t.Errorf("after the last attempt: %s, want failed", s)
	}
	reporter.Err = nil
	if err := w.Work(ctx, job(second, 1)); err != nil {
		t.Fatal(err)
	}
	if s, qr := status(second); s != "fiscalized" || qr == nil {
		t.Errorf("a later report of a failed receipt: %s, want fiscalized with its QR", s)
	}
}

// seedReceiptClinic creates, as the owner, a synthetic clinic and its admin's account.
func seedReceiptClinic(t *testing.T, owner *pgxpool.Pool) (org, account string) {
	t.Helper()
	ctx := t.Context()
	if err := owner.QueryRow(ctx, `INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id, mfa_enrolled_at)
		VALUES ('admin@example.test', 'h', 'admin', $1, now()) RETURNING id::text`, org).Scan(&account); err != nil {
		t.Fatal(err)
	}
	return org, account
}

// issueReceipt stores a synthetic cash receipt of the clinic with its next number and returns its id.
func issueReceipt(t *testing.T, app *pgxpool.Pool, org, account string, at time.Time) string {
	t.Helper()
	repo, tr := invoice.NewPostgresRepository(app), postgres.NewTransactor(app)
	var id string
	err := tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		n, err := repo.NextNumber(ctx, org)
		if err != nil {
			return err
		}
		inv, err := repo.Insert(ctx, invoice.Invoice{
			IssuedAt: at, OrganizationID: org, SellerLegalName: "Synthetic clinic d.o.o.", SellerTaxID: "4200000000000",
			SellerAddress: "Placeholder street 1", IssuedByAccountID: account, Currency: "BAM", Kind: invoice.KindReceipt,
			PaymentMethod: invoice.PaymentCash, Number: n, TotalMinor: 5000,
			Lines: []invoice.Line{{Name: "Placeholder item", Quantity: 1, UnitPriceMinor: 5000, TotalMinor: 5000}},
		})
		id = inv.ID
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
