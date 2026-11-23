package subscription_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed creates clinics A and B and the basic plan as the owner.
func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, updated_at, contributes_since) VALUES
		($1, 'Synthetic clinic A', '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z'),
		($2, 'Synthetic clinic B', '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z')`, orgA, orgB); err != nil {
		t.Fatal(err)
	}
	err := subscription.NewPostgresRepository(owner).PutPlan(t.Context(), subscription.Plan{Code: "basic", Name: "Basic", Currency: "BAM", PricePerSeatMinor: 10000, Active: true})
	if err != nil {
		t.Fatal(err)
	}
}

// in runs fn in a transaction on db scoped to org ("" for no scope).
func in(t *testing.T, db *pgxpool.Pool, repo subscription.Repository, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(db).WithTx(t.Context(), func(ctx context.Context) error {
		if org != "" {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
		}
		return fn(ctx)
	})
}

// A clinic's membership and invoices are stored and read in its own scope only; the run's scope lists the clinics.
func TestRepository_StoresAMembershipInItsClinicsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := subscription.NewPostgresRepository(app)
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	for _, org := range []string{orgA, orgB, orgA} {
		created := false
		if err := in(t, app, repo, org, func(ctx context.Context) (err error) {
			created, err = repo.Insert(ctx, org, "basic", at, at.Add(subscription.TrialLength))
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if again := org == orgA && created; again {
			sub := subscription.Subscription{}
			_ = in(t, app, repo, org, func(ctx context.Context) (err error) { sub, err = repo.Subscription(ctx, org); return err })
			if sub.Status != subscription.StatusTrial || !sub.TrialEndsAt.Equal(at.Add(subscription.TrialLength)) || sub.PlanCode != "basic" {
				t.Errorf("Subscription() = %+v, want the trial", sub)
			}
		}
	}
	// Another clinic's scope, and no scope, see nothing.
	for _, scope := range []string{orgB, ""} {
		err := in(t, app, repo, scope, func(ctx context.Context) error { _, err := repo.Subscription(ctx, orgA); return err })
		if !errors.Is(err, subscription.ErrNotFound) {
			t.Errorf("scope %q: Subscription(A) = %v, want ErrNotFound", scope, err)
		}
	}
	var orgs []string
	if err := in(t, app, repo, "", func(ctx context.Context) (err error) {
		if err := repo.SetRun(ctx); err != nil {
			return err
		}
		orgs, err = repo.Subscribed(ctx)
		return err
	}); err != nil || len(orgs) != 2 {
		t.Errorf("Subscribed() = %v, %v, want both clinics", orgs, err)
	}
	if err := in(t, app, repo, orgA, func(ctx context.Context) error {
		if err := repo.SetStatus(ctx, orgA, subscription.StatusActive, at); err != nil {
			return err
		}
		return repo.SetPeriod(ctx, orgA, at, at.AddDate(0, 1, 0), at)
	}); err != nil {
		t.Fatal(err)
	}
	if err := in(t, app, repo, orgB, func(ctx context.Context) error { return repo.SetStatus(ctx, orgA, subscription.StatusCancelled, at) }); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("clinic B moving clinic A's status: %v, want ErrNotFound", err)
	}
	var sub subscription.Subscription
	_ = in(t, app, repo, orgA, func(ctx context.Context) (err error) { sub, err = repo.Lock(ctx, orgA); return err })
	if sub.Status != subscription.StatusActive || !sub.CurrentPeriodStart.Equal(at) {
		t.Errorf("Lock() = %+v, want active with its period", sub)
	}
	if _, err := repo.Plan(t.Context(), "pro"); !errors.Is(err, subscription.ErrNoPlan) {
		t.Errorf("Plan(pro) = %v, want ErrNoPlan", err)
	}
	if plans, err := repo.Plans(t.Context()); err != nil || len(plans) != 1 || plans[0].PricePerSeatMinor != 10000 {
		t.Errorf("Plans() = %+v, %v", plans, err)
	}
	if err := repo.PutPlan(t.Context(), subscription.Plan{Code: "pro", Name: "Pro", Currency: "BAM", Active: true}); err == nil {
		t.Error("the app stored a plan, want only the owner to")
	}
}

// issue stores an invoice of org for the month from start with the next number, in one transaction.
func issue(t *testing.T, db *pgxpool.Pool, repo subscription.Repository, org string, start time.Time) (subscription.Invoice, error) {
	t.Helper()
	var out subscription.Invoice
	err := in(t, db, repo, org, func(ctx context.Context) error {
		n, err := repo.NextNumber(ctx)
		if err != nil {
			return err
		}
		out, err = repo.InsertInvoice(ctx, subscription.Invoice{
			OrganizationID: org, Number: n, PeriodStart: start, PeriodEnd: start.AddDate(0, 1, 0), Seats: 2, AmountMinor: 20000, Currency: "BAM",
			IssuedAt: start, DueAt: start.Add(subscription.PaymentTerm),
		})
		return err
	})
	return out, err
}

// Invoices take gapless numbers (a refused one leaves no gap, concurrent ones never share one), are one per clinic and
// month, and are marked paid by the owner only.
func TestRepository_NumbersInvoicesWithoutGapsAndOnlyTheOwnerMarksPaid(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := subscription.NewPostgresRepository(app)
	dec := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	first, err := issue(t, app, repo, orgA, dec)
	if err != nil || first.Number != 1 || first.Status != subscription.InvoiceOpen || first.ID == "" {
		t.Fatalf("first invoice = %+v, %v", first, err)
	}
	if _, err := issue(t, app, repo, orgA, dec); err == nil {
		t.Fatal("a second invoice for the same month was stored")
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, org := range []string{orgA, orgB} {
		wg.Go(func() { _, errs[i] = issue(t, app, repo, org, dec.AddDate(0, 1+i, 0)) })
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	var numbers []int64
	rows, err := owner.Query(t.Context(), `SELECT number FROM subscription_invoice ORDER BY number`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		numbers = append(numbers, n)
	}
	if len(numbers) != 3 || numbers[0] != 1 || numbers[1] != 2 || numbers[2] != 3 {
		t.Errorf("numbers %v, want 1 2 3: the refused invoice left no gap", numbers)
	}
	var list []subscription.Invoice
	_ = in(t, app, repo, orgA, func(ctx context.Context) (err error) { list, err = repo.Invoices(ctx, orgA); return err })
	if len(list) != 2 || !list[0].PeriodStart.After(list[1].PeriodStart) {
		t.Errorf("Invoices(A) = %+v, want its two, newest first", list)
	}
	_ = in(t, app, repo, orgB, func(ctx context.Context) (err error) { list, err = repo.Invoices(ctx, orgA); return err })
	if len(list) != 0 {
		t.Errorf("clinic B sees %d of clinic A's invoices", len(list))
	}
	now := first.DueAt.Add(time.Hour)
	var since time.Time
	var found bool
	_ = in(t, app, repo, orgA, func(ctx context.Context) (err error) {
		if _, found, err = repo.InvoiceOf(ctx, orgA, dec); err != nil {
			return err
		}
		since, err = repo.OldestPastDue(ctx, orgA, now)
		return err
	})
	if !found || !since.Equal(first.DueAt) {
		t.Errorf("InvoiceOf() found %v, OldestPastDue() = %v, want the first invoice's due date", found, since)
	}
	if _, err := repo.MarkPaid(t.Context(), 1, now); err == nil {
		t.Error("the app marked an invoice paid, want only the owner to")
	}
	asOwner := subscription.NewPostgresRepository(owner)
	paid, err := asOwner.MarkPaid(t.Context(), 1, now)
	if err != nil || paid.Status != subscription.InvoicePaid || !paid.PaidAt.Equal(now) || paid.OrganizationID != orgA {
		t.Errorf("MarkPaid() = %+v, %v", paid, err)
	}
	if _, err := asOwner.MarkPaid(t.Context(), 1, now); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("MarkPaid(paid) = %v, want ErrNotFound", err)
	}
	_ = in(t, app, repo, orgA, func(ctx context.Context) (err error) { since, err = repo.OldestPastDue(ctx, orgA, now); return err })
	if !since.IsZero() {
		t.Errorf("OldestPastDue() after the payment = %v, want none", since)
	}
}
