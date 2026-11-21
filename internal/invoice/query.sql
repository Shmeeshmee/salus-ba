-- Receipts and invoices (docs/plans/clinic-cms.md, package C). Step 0 holds the first query; package C adds the rest.

-- name: InvoiceByID :one
SELECT id, organization_id, seller_legal_name, seller_tax_id, seller_address, patient_id, company_name, company_tax_id,
       appointment_id, kind, reverses_id, number, issued_at, issued_by_account_id, payment_method, total_minor, currency,
       fiscal_status, fiscal_id, fiscal_code, fiscal_qr, fiscalized_at
FROM invoice
WHERE id = @id;
