-- +goose Up
-- Package M2 of docs/clinic-cms (WAVE4.md): the clinic's data export. The admin asks, a job builds one archive (the
-- price list, the staff, the appointments and the receipts as CSV), encrypted as attachments are (pkg/envelope: a data
-- key wrapped by the key-encryption key, bound to the export's id) and stored in the object store under
-- clinicexport/<id>. The admin downloads it until expires_at, 7 days after the request; the retention job then deletes
-- the row and its object. At most one export of a clinic is queued at a time (the partial unique index).
CREATE TABLE clinic_export (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    requested_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    ready_at timestamptz,
    organization_id uuid NOT NULL REFERENCES organization (id),
    requested_by_account_id uuid NOT NULL REFERENCES account (id),
    status text DEFAULT 'queued' NOT NULL CHECK (status IN ('queued', 'ready', 'failed')),
    size bigint CHECK (size >= 0), -- the archive's plaintext bytes; null until ready
    kek_id text,
    wrapped_key bytea,
    CONSTRAINT clinic_export_expiry CHECK (expires_at > requested_at),
    CONSTRAINT clinic_export_ready CHECK ((status = 'ready') = (ready_at IS NOT NULL AND size IS NOT NULL
                                                                 AND kek_id IS NOT NULL AND wrapped_key IS NOT NULL))
);
CREATE UNIQUE INDEX clinic_export_one_queued ON clinic_export (organization_id) WHERE status = 'queued';
CREATE INDEX clinic_export_requested ON clinic_export (requested_at);

ALTER TABLE clinic_export ENABLE ROW LEVEL SECURITY;
-- The clinic's scope only, asked by an account of its own.
CREATE POLICY clinic_export_clinic ON clinic_export
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = clinic_export.requested_by_account_id
                              AND a.organization_id = clinic_export.organization_id));
GRANT SELECT, INSERT ON clinic_export TO salus_app;
GRANT UPDATE (status, ready_at, size, kek_id, wrapped_key) ON clinic_export TO salus_app;
-- The retention job deletes an export 7 days after it was asked for (internal/retention, "clinic_export").
GRANT SELECT, DELETE ON clinic_export TO salus_retention;
