package subscription

import (
	"context"
	"time"
)

// Repository stores memberships and their invoices. Every read sees only the clinic set by SetOrganization
// (row-level security); plans are read by everyone. The owner's connection (salusctl) sees every clinic.
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetRun scopes the rest of the transaction in ctx to the monthly run: Subscribed sees every clinic.
	SetRun(ctx context.Context) error
	// Plans returns the active plans, by code.
	Plans(ctx context.Context) ([]Plan, error)
	// Plan returns the plan code, or ErrNoPlan.
	Plan(ctx context.Context, code string) (Plan, error)
	// PutPlan stores or replaces the plan p.Code. Only the owner may.
	PutPlan(ctx context.Context, p Plan) error
	// Subscription returns the clinic's membership, or ErrNotFound.
	Subscription(ctx context.Context, org string) (Subscription, error)
	// Lock is Subscription, locking the row until the transaction ends.
	Lock(ctx context.Context, org string) (Subscription, error)
	// Insert stores the clinic's membership as a trial of plan that ends at trialEndsAt, and reports whether it did:
	// false when the clinic has one already.
	Insert(ctx context.Context, org, plan string, at, trialEndsAt time.Time) (bool, error)
	// SetStatus moves the clinic's membership to status; ErrNotFound without one.
	SetStatus(ctx context.Context, org string, status Status, at time.Time) error
	// SetPeriod sets the clinic's current period; ErrNotFound without a membership.
	SetPeriod(ctx context.Context, org string, start, end, at time.Time) error
	// Subscribed returns the ids of the clinics with a membership that is not cancelled, after SetRun.
	Subscribed(ctx context.Context) ([]string, error)
	// Invoices returns the clinic's invoices, newest first.
	Invoices(ctx context.Context, org string) ([]Invoice, error)
	// InvoiceOf returns the clinic's invoice for the month that starts at periodStart, and false when it has none.
	InvoiceOf(ctx context.Context, org string, periodStart time.Time) (Invoice, bool, error)
	// NextNumber takes the next number of Salus's sequence, locking it until the transaction ends (no gap on a
	// rollback).
	NextNumber(ctx context.Context) (int64, error)
	// InsertInvoice stores the open invoice in and returns it with its id.
	InsertInvoice(ctx context.Context, in Invoice) (Invoice, error)
	// MarkPaid marks the open invoice number paid at at and returns it; ErrNotFound when no open invoice has that
	// number. Only the owner may.
	MarkPaid(ctx context.Context, number int64, at time.Time) (Invoice, error)
	// OldestPastDue returns the due date of the clinic's oldest open invoice past due at now, zero for none.
	OldestPastDue(ctx context.Context, org string, now time.Time) (time.Time, error)
}

// Jobs enqueues the invoice's e-mail in the transaction in the context (River's InsertTx).
type Jobs interface {
	EnqueueInvoiceEmail(ctx context.Context, a InvoiceEmailArgs) error
}
