-- +goose Up
-- Package D2 of docs/clinic-cms (WAVE5.md S4): the patient's own export, which fulfils an access or portability
-- request they filed in the app. A job builds one archive (FHIR Bundles of what the patient can read in the app, and
-- their account data for the account target), encrypted as attachments are (pkg/envelope: a data key wrapped by the
-- key-encryption key, bound to the request's id) and stored in the object store under datarequest/<request id>. The
-- patient downloads it until expires_at, 7 days after it was ready; the retention job then deletes the row and its
-- object. At most one export of a patient is queued at a time (the partial unique index).
CREATE TABLE data_request_export (
    data_request_id uuid NOT NULL PRIMARY KEY REFERENCES data_request (id),
    requested_at timestamptz NOT NULL,
    ready_at timestamptz,
    expires_at timestamptz,
    account_id uuid NOT NULL REFERENCES account (id),
    status text DEFAULT 'queued' NOT NULL CHECK (status IN ('queued', 'ready', 'failed')),
    size bigint CHECK (size >= 0), -- the archive's plaintext bytes; null until ready
    kek_id text,
    wrapped_key bytea,
    CONSTRAINT data_request_export_ready CHECK ((status = 'ready') = (ready_at IS NOT NULL AND expires_at IS NOT NULL
                                                                      AND size IS NOT NULL AND kek_id IS NOT NULL
                                                                      AND wrapped_key IS NOT NULL)),
    CONSTRAINT data_request_export_expiry CHECK (expires_at IS NULL OR expires_at > ready_at)
);
CREATE UNIQUE INDEX data_request_export_one_queued ON data_request_export (account_id) WHERE status = 'queued';
CREATE INDEX data_request_export_requested ON data_request_export (requested_at);

ALTER TABLE data_request_export ENABLE ROW LEVEL SECURITY;
-- The patient's scope only: their own export, of an access or portability request of their own.
CREATE POLICY data_request_export_own ON data_request_export
    USING (account_id = NULLIF(current_setting('app.account_id', true), '')::uuid)
    WITH CHECK (account_id = NULLIF(current_setting('app.account_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM data_request r
                            WHERE r.id = data_request_export.data_request_id
                              AND r.account_id = data_request_export.account_id
                              AND r.kind IN ('access', 'portability')));
GRANT SELECT, INSERT ON data_request_export TO salus_app;
GRANT UPDATE (status, ready_at, expires_at, size, kek_id, wrapped_key) ON data_request_export TO salus_app;
-- The retention job deletes an export 7 days after it was ready, or a failed one (internal/retention,
-- "data_request_export").
GRANT SELECT, DELETE ON data_request_export TO salus_retention;

-- Fulfils the patient's open access or portability request once its export is ready: status fulfilled, answered
-- done, with its history entry by the job (null). The patient's scope may only withdraw a request (00071's guard), so
-- the build job calls this function in that scope; it answers only a request of the scope's account whose export is
-- ready, and returns whether it did.
-- +goose StatementBegin
CREATE FUNCTION data_request_export_fulfil(request_id uuid) RETURNS boolean
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
DECLARE
    account uuid := NULLIF(current_setting('app.account_id', true), '')::uuid;
    done boolean;
BEGIN
    IF account IS NULL OR NOT EXISTS (SELECT 1 FROM public.data_request_export x
                                      WHERE x.data_request_id = request_id AND x.account_id = account
                                        AND x.status = 'ready') THEN
        RETURN false;
    END IF;
    PERFORM set_config('app.account_id', '', true);
    UPDATE public.data_request r SET status = 'fulfilled', answered_at = now(), answer_code = 'done',
                                     version = r.version + 1
    WHERE r.id = request_id AND r.account_id = account AND r.status IN ('received', 'in_progress')
      AND r.kind IN ('access', 'portability');
    done := FOUND;
    IF done THEN
        INSERT INTO public.data_request_change (changed_at, data_request_id, changed_by_account_id, status)
        VALUES (now(), request_id, NULL, 'fulfilled');
    END IF;
    PERFORM set_config('app.account_id', account::text, true);
    RETURN done;
END
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION data_request_export_fulfil(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION data_request_export_fulfil(uuid) TO salus_app;
