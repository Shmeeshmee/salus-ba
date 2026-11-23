-- The clinic's membership (docs/plans/clinic-cms.md, package D).

-- name: ActivePlans :many
SELECT code, name, price_per_seat_minor, currency, features, active
FROM subscription_plan
WHERE active
ORDER BY code;

-- name: PlanByCode :one
SELECT code, name, price_per_seat_minor, currency, features, active
FROM subscription_plan
WHERE code = @code;

-- name: PutPlan :exec
-- The owner's, in salusctl: the app has no write grant on the plans.
INSERT INTO subscription_plan (code, name, price_per_seat_minor, currency, features, active)
VALUES (@code, @name, @price_per_seat_minor, @currency, @features, @active)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name, price_per_seat_minor = EXCLUDED.price_per_seat_minor, currency = EXCLUDED.currency,
    features = EXCLUDED.features, active = EXCLUDED.active;

-- name: SubscriptionByOrganization :one
SELECT organization_id, updated_at, plan_code, status, trial_ends_at, current_period_start, current_period_end
FROM clinic_subscription
WHERE organization_id = @organization_id;

-- name: LockSubscription :one
-- The row is locked until the transaction ends: one status move or monthly invoice of a clinic at a time.
SELECT organization_id, updated_at, plan_code, status, trial_ends_at, current_period_start, current_period_end
FROM clinic_subscription
WHERE organization_id = @organization_id
FOR UPDATE;

-- name: InsertSubscription :one
INSERT INTO clinic_subscription (organization_id, created_at, updated_at, plan_code, status, trial_ends_at)
VALUES (@organization_id, @at, @at, @plan_code, 'trial', @trial_ends_at)
ON CONFLICT (organization_id) DO NOTHING
RETURNING organization_id;

-- name: SetSubscriptionStatus :execrows
UPDATE clinic_subscription
SET status = @status, updated_at = @at
WHERE organization_id = @organization_id;

-- name: SetSubscriptionPeriod :execrows
UPDATE clinic_subscription
SET current_period_start = @period_start, current_period_end = @period_end, updated_at = @at
WHERE organization_id = @organization_id;

-- name: SubscribedOrganizations :many
-- Every clinic with a membership that is not cancelled, in the scope app.subscription_run = 'all' (00061).
SELECT organization_id
FROM clinic_subscription
WHERE status <> 'cancelled'
ORDER BY organization_id;

-- name: InvoicesByOrganization :many
SELECT id, organization_id, number, period_start, period_end, seats, amount_minor, currency, issued_at, due_at, paid_at, status
FROM subscription_invoice
WHERE organization_id = @organization_id
ORDER BY period_start DESC, number DESC;

-- name: InvoiceByPeriod :one
SELECT id, organization_id, number, period_start, period_end, seats, amount_minor, currency, issued_at, due_at, paid_at, status
FROM subscription_invoice
WHERE organization_id = @organization_id AND period_start = @period_start;

-- name: NextInvoiceNumber :one
-- Locks Salus's sequence until the transaction ends: a rolled-back invoice leaves no gap.
UPDATE subscription_invoice_sequence
SET last_number = last_number + 1
RETURNING last_number;

-- name: InsertInvoice :one
INSERT INTO subscription_invoice (organization_id, number, period_start, period_end, seats, amount_minor, currency, issued_at, due_at)
VALUES (@organization_id, @number, @period_start, @period_end, @seats, @amount_minor, @currency, @issued_at, @due_at)
RETURNING id, organization_id, number, period_start, period_end, seats, amount_minor, currency, issued_at, due_at, paid_at, status;

-- name: MarkInvoicePaid :one
-- The owner's, in salusctl: the app has no UPDATE grant on the invoices.
UPDATE subscription_invoice
SET status = 'paid', paid_at = @paid_at
WHERE number = @number AND status = 'open'
RETURNING id, organization_id, number, period_start, period_end, seats, amount_minor, currency, issued_at, due_at, paid_at, status;

-- name: OldestPastDue :one
-- The due date of the clinic's oldest open invoice past due at @now; NULL for none.
SELECT min(due_at)::timestamptz AS due_at
FROM subscription_invoice
WHERE organization_id = @organization_id AND status = 'open' AND due_at < @now;
