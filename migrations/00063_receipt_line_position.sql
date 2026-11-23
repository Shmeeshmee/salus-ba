-- +goose Up
-- Package H of docs/plans/clinic-cms.md (receipts follow-ups), on package C's tables (00059).
--   - invoice_line.position: a receipt's lines come back in the order they were entered (0, 1, ...), on the screen,
--     the PDF and the report. Lines issued before this migration are numbered by name, the order they were read in.
--   - invoice.idempotency_key: the key the receipt dialog sends with POST /clinic/receipts (a UUID), so a double click
--     or a retried request issues one receipt. Unique per clinic; NULL for a receipt issued without one and for a
--     reversal. Not personal data (a random value), kept with its receipt.

ALTER TABLE invoice_line ADD COLUMN IF NOT EXISTS position integer;

-- invoice_line refuses every UPDATE (invoice_line_no_change); the numbering of the existing lines is the one update.
ALTER TABLE invoice_line DISABLE TRIGGER invoice_line_no_change;
UPDATE invoice_line l SET position = n.position
FROM (SELECT id, row_number() OVER (PARTITION BY invoice_id ORDER BY name, id) - 1 AS position FROM invoice_line) n
WHERE l.id = n.id AND l.position IS NULL;
ALTER TABLE invoice_line ENABLE TRIGGER invoice_line_no_change;

ALTER TABLE invoice_line ALTER COLUMN position SET NOT NULL;
ALTER TABLE invoice_line DROP CONSTRAINT IF EXISTS invoice_line_position_check;
ALTER TABLE invoice_line ADD CONSTRAINT invoice_line_position_check CHECK (position >= 0);
CREATE UNIQUE INDEX IF NOT EXISTS invoice_line_position_key ON invoice_line (invoice_id, position);
DROP INDEX IF EXISTS invoice_line_invoice_idx; -- the unique index leads with invoice_id

ALTER TABLE invoice ADD COLUMN IF NOT EXISTS idempotency_key uuid;
CREATE UNIQUE INDEX IF NOT EXISTS invoice_idempotency_key ON invoice (organization_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
