-- +goose Up
-- Package D of docs/plans/clinic-cms.md (the clinic's membership), on step 0's tables (00059). Nothing here is
-- personal data.
--   - subscription_invoice_sequence: Salus's own invoice numbers, gapless: one row, locked by the monthly job's
--     UPDATE ... RETURNING in the invoice's transaction (as invoice_sequence does for a clinic's receipts).
--   - clinic_subscription_run: the monthly job lists the clinics that have a membership (their ids, plans and statuses:
--     not personal data) in the scope app.subscription_run = 'all', then works on each in that clinic's own scope.

CREATE TABLE IF NOT EXISTS subscription_invoice_sequence (
    id boolean DEFAULT true NOT NULL PRIMARY KEY CHECK (id),
    last_number bigint DEFAULT 0 NOT NULL CHECK (last_number >= 0)
);
INSERT INTO subscription_invoice_sequence (id) VALUES (true) ON CONFLICT DO NOTHING;
GRANT SELECT ON subscription_invoice_sequence TO salus_app;
GRANT UPDATE (last_number) ON subscription_invoice_sequence TO salus_app;

DROP POLICY IF EXISTS clinic_subscription_run ON clinic_subscription;
CREATE POLICY clinic_subscription_run ON clinic_subscription FOR SELECT
    USING (current_setting('app.subscription_run', true) = 'all');
