package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/subscription/subscriptiondb"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db, using the transaction in the context when there is one. As
// salus_app, row-level security shows only the clinic set by SetOrganization; salusctl connects as the owner.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *subscriptiondb.Queries {
	return subscriptiondb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetRun(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.subscription_run", "all")
}

func (r *postgresRepository) Plans(ctx context.Context) ([]Plan, error) {
	rows, err := r.q(ctx).ActivePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPlan(row))
	}
	return out, nil
}

func (r *postgresRepository) Plan(ctx context.Context, code string) (Plan, error) {
	row, err := r.q(ctx).PlanByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNoPlan
	}
	return toPlan(row), err
}

func (r *postgresRepository) PutPlan(ctx context.Context, p Plan) error {
	features := p.Features
	if features == nil {
		features = []string{}
	}
	return r.q(ctx).PutPlan(ctx, subscriptiondb.PutPlanParams{
		Code: p.Code, Name: p.Name, PricePerSeatMinor: p.PricePerSeatMinor, Currency: p.Currency, Features: features, Active: p.Active,
	})
}

func (r *postgresRepository) Subscription(ctx context.Context, org string) (Subscription, error) {
	id, ok := uuid(org)
	if !ok {
		return Subscription{}, ErrNotFound
	}
	row, err := r.q(ctx).SubscriptionByOrganization(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return toSubscription(subscriptiondb.LockSubscriptionRow(row)), err
}

func (r *postgresRepository) Lock(ctx context.Context, org string) (Subscription, error) {
	id, ok := uuid(org)
	if !ok {
		return Subscription{}, ErrNotFound
	}
	row, err := r.q(ctx).LockSubscription(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return toSubscription(row), err
}

func (r *postgresRepository) Insert(ctx context.Context, org, plan string, at, trialEndsAt time.Time) (bool, error) {
	id, ok := uuid(org)
	if !ok {
		return false, ErrNotFound
	}
	_, err := r.q(ctx).InsertSubscription(ctx, subscriptiondb.InsertSubscriptionParams{
		OrganizationID: id, At: instant(at), PlanCode: plan, TrialEndsAt: instant(trialEndsAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *postgresRepository) SetStatus(ctx context.Context, org string, status Status, at time.Time) error {
	id, ok := uuid(org)
	if !ok {
		return ErrNotFound
	}
	n, err := r.q(ctx).SetSubscriptionStatus(ctx, subscriptiondb.SetSubscriptionStatusParams{Status: string(status), At: instant(at), OrganizationID: id})
	return one(n, err)
}

func (r *postgresRepository) SetPeriod(ctx context.Context, org string, start, end, at time.Time) error {
	id, ok := uuid(org)
	if !ok {
		return ErrNotFound
	}
	n, err := r.q(ctx).SetSubscriptionPeriod(ctx, subscriptiondb.SetSubscriptionPeriodParams{
		PeriodStart: instant(start), PeriodEnd: instant(end), At: instant(at), OrganizationID: id,
	})
	return one(n, err)
}

func (r *postgresRepository) Subscribed(ctx context.Context) ([]string, error) {
	rows, err := r.q(ctx).SubscribedOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, id := range rows {
		out = append(out, id.String())
	}
	return out, nil
}

func (r *postgresRepository) Invoices(ctx context.Context, org string) ([]Invoice, error) {
	id, ok := uuid(org)
	if !ok {
		return []Invoice{}, nil
	}
	rows, err := r.q(ctx).InvoicesByOrganization(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Invoice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInvoice(row))
	}
	return out, nil
}

func (r *postgresRepository) InvoiceOf(ctx context.Context, org string, periodStart time.Time) (Invoice, bool, error) {
	id, ok := uuid(org)
	if !ok {
		return Invoice{}, false, nil
	}
	row, err := r.q(ctx).InvoiceByPeriod(ctx, subscriptiondb.InvoiceByPeriodParams{OrganizationID: id, PeriodStart: day(periodStart)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Invoice{}, false, nil
	case err != nil:
		return Invoice{}, false, err
	}
	return toInvoice(row), true, nil
}

func (r *postgresRepository) NextNumber(ctx context.Context) (int64, error) {
	return r.q(ctx).NextInvoiceNumber(ctx)
}

func (r *postgresRepository) InsertInvoice(ctx context.Context, in Invoice) (Invoice, error) {
	id, ok := uuid(in.OrganizationID)
	if !ok {
		return Invoice{}, ErrNotFound
	}
	row, err := r.q(ctx).InsertInvoice(ctx, subscriptiondb.InsertInvoiceParams{
		OrganizationID: id, Number: in.Number, PeriodStart: day(in.PeriodStart), PeriodEnd: day(in.PeriodEnd), Seats: in.Seats,
		AmountMinor: in.AmountMinor, Currency: in.Currency, IssuedAt: instant(in.IssuedAt), DueAt: instant(in.DueAt),
	})
	if err != nil {
		return Invoice{}, err
	}
	return toInvoice(row), nil
}

func (r *postgresRepository) MarkPaid(ctx context.Context, number int64, at time.Time) (Invoice, error) {
	row, err := r.q(ctx).MarkInvoicePaid(ctx, subscriptiondb.MarkInvoicePaidParams{PaidAt: instant(at), Number: number})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Invoice{}, ErrNotFound
	case err != nil:
		return Invoice{}, err
	}
	return toInvoice(row), nil
}

func (r *postgresRepository) OldestPastDue(ctx context.Context, org string, now time.Time) (time.Time, error) {
	id, ok := uuid(org)
	if !ok {
		return time.Time{}, nil
	}
	due, err := r.q(ctx).OldestPastDue(ctx, subscriptiondb.OldestPastDueParams{OrganizationID: id, Now: instant(now)})
	return due.Time, err
}

func toPlan(row subscriptiondb.SubscriptionPlan) Plan {
	return Plan{
		Code: row.Code, Name: row.Name, Currency: row.Currency, Features: row.Features, PricePerSeatMinor: row.PricePerSeatMinor, Active: row.Active,
	}
}

func toSubscription(row subscriptiondb.LockSubscriptionRow) Subscription {
	return Subscription{
		TrialEndsAt: row.TrialEndsAt.Time, CurrentPeriodStart: row.CurrentPeriodStart.Time, CurrentPeriodEnd: row.CurrentPeriodEnd.Time,
		LastUpdated: row.UpdatedAt.Time, OrganizationID: row.OrganizationID.String(), PlanCode: row.PlanCode, Status: Status(row.Status),
	}
}

func toInvoice(row subscriptiondb.SubscriptionInvoice) Invoice {
	return Invoice{
		PeriodStart: row.PeriodStart.Time, PeriodEnd: row.PeriodEnd.Time, IssuedAt: row.IssuedAt.Time, DueAt: row.DueAt.Time, PaidAt: row.PaidAt.Time,
		ID: row.ID.String(), OrganizationID: row.OrganizationID.String(), Currency: row.Currency, Status: InvoiceStatus(row.Status),
		Number: row.Number, AmountMinor: row.AmountMinor, Seats: row.Seats,
	}
}

// uuid parses id, false for one that is no UUID.
func uuid(id string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(id) == nil
}

// instant is t as a nullable timestamptz, NULL for the zero time.
func instant(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }

// day is the calendar date of t as it reads in its own zone.
func day(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// one is err, or ErrNotFound when a write changed no row.
func one(n int64, err error) error {
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
