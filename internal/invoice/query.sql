-- Receipts and invoices (docs/plans/clinic-cms.md, package C).

-- name: InvoiceByID :one
SELECT sqlc.embed(i), r.id AS reversed_by_id
FROM invoice i
LEFT JOIN invoice r ON r.reverses_id = i.id
WHERE i.id = @id;

-- name: InvoiceLocked :one
-- The receipt, locked until the transaction ends: the fiscal job reports it once.
SELECT * FROM invoice WHERE id = @id FOR UPDATE;

-- name: InvoiceByIdempotencyKey :one
-- The clinic's receipt issued with the key (POST /clinic/receipts, Idempotency-Key).
SELECT sqlc.embed(i), r.id AS reversed_by_id
FROM invoice i
LEFT JOIN invoice r ON r.reverses_id = i.id
WHERE i.organization_id = @organization_id AND i.idempotency_key = @idempotency_key;

-- name: InvoicesIssued :many
SELECT sqlc.embed(i), r.id AS reversed_by_id
FROM invoice i
LEFT JOIN invoice r ON r.reverses_id = i.id
WHERE i.organization_id = @organization_id AND i.issued_at >= @from_time AND i.issued_at < @to_time
ORDER BY i.issued_at DESC, i.number DESC;

-- name: InvoicesOfPatient :many
SELECT sqlc.embed(i), r.id AS reversed_by_id
FROM invoice i
LEFT JOIN invoice r ON r.reverses_id = i.id
WHERE i.patient_id = @patient_id
ORDER BY i.issued_at DESC, i.number DESC;

-- name: InvoiceLines :many
-- The lines of the receipts, each receipt's in the order they were entered.
SELECT * FROM invoice_line WHERE invoice_id = ANY(@invoice_ids::uuid[]) ORDER BY invoice_id, position;

-- name: NextInvoiceNumber :one
-- The clinic's next receipt number. The row stays locked until the transaction ends, so two receipts issued at the
-- same instant get consecutive numbers, and a receipt that is rolled back gives its number back: no gap, no repeat.
INSERT INTO invoice_sequence (organization_id, last_number) VALUES (@organization_id, 1)
ON CONFLICT (organization_id) DO UPDATE SET last_number = invoice_sequence.last_number + 1
RETURNING last_number;

-- name: InsertInvoice :one
INSERT INTO invoice (organization_id, seller_legal_name, seller_tax_id, seller_address, patient_id, company_name,
                     company_tax_id, appointment_id, kind, reverses_id, number, issued_at, issued_by_account_id,
                     payment_method, total_minor, currency, idempotency_key)
VALUES (@organization_id, @seller_legal_name, @seller_tax_id, @seller_address, @patient_id, @company_name,
        @company_tax_id, @appointment_id, @kind, @reverses_id, @number, @issued_at, @issued_by_account_id,
        @payment_method, @total_minor, @currency, @idempotency_key)
RETURNING id;

-- name: InsertInvoiceLine :one
INSERT INTO invoice_line (invoice_id, position, charge_item_definition_id, name, quantity, unit_price_minor,
                          discount_minor, vat_rate_bp, total_minor)
VALUES (@invoice_id, @position, @charge_item_definition_id, @name, @quantity, @unit_price_minor, @discount_minor,
        @vat_rate_bp, @total_minor)
RETURNING id;

-- name: SetInvoiceFiscalized :execrows
UPDATE invoice
SET fiscal_status = 'fiscalized', fiscal_id = @fiscal_id, fiscal_code = @fiscal_code, fiscal_qr = @fiscal_qr,
    fiscalized_at = @fiscalized_at
WHERE id = @id AND fiscal_status IN ('pending', 'failed');

-- name: SetInvoiceFailed :execrows
UPDATE invoice SET fiscal_status = 'failed' WHERE id = @id AND fiscal_status = 'pending';
