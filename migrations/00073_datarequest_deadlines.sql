-- +goose Up
-- The deadline warnings of the data subject's requests (docs/clinic-cms/WAVE5.md, D1, S3): the answerer is warned 7
-- days and 1 day before a request's deadline (its due date, or the extension's). A scheduled job finds the open requests
-- of every target, the account's included, in its own scope (app.data_request_deadlines = 'all'), queues each warning
-- (sent by D5) and marks when it queued the last one, so each is queued once per deadline. The scope shows no closed
-- request and changes nothing but that mark. No personal data: an instant.
-- NULLIF and the comparison with 'all': an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE data_request ADD COLUMN deadline_warned_at timestamptz;
GRANT UPDATE (deadline_warned_at) ON data_request TO salus_app;

CREATE POLICY data_request_deadlines ON data_request FOR SELECT
    USING (current_setting('app.data_request_deadlines', true) = 'all' AND status IN ('received', 'in_progress'));
CREATE POLICY data_request_deadlines_update ON data_request FOR UPDATE
    USING (current_setting('app.data_request_deadlines', true) = 'all' AND status IN ('received', 'in_progress'))
    WITH CHECK (current_setting('app.data_request_deadlines', true) = 'all' AND status IN ('received', 'in_progress'));

-- In the job's scope only the mark changes, for every role.
-- +goose StatementBegin
CREATE FUNCTION data_request_deadlines_guard() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF current_setting('app.data_request_deadlines', true) = 'all'
       AND (NEW.status <> OLD.status OR NEW.version <> OLD.version
            OR NEW.extended_until IS DISTINCT FROM OLD.extended_until
            OR NEW.extension_reason IS DISTINCT FROM OLD.extension_reason
            OR NEW.answered_at IS DISTINCT FROM OLD.answered_at OR NEW.answer_code IS DISTINCT FROM OLD.answer_code) THEN
        RAISE EXCEPTION 'data_request: the deadline job only marks its warning' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER data_request_deadlines_guard BEFORE UPDATE ON data_request
    FOR EACH ROW EXECUTE FUNCTION data_request_deadlines_guard();
