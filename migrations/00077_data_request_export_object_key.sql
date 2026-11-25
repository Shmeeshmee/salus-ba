-- +goose Up
-- Package D7 of docs/clinic-cms (WAVE5.md): each build attempt of a patient's export writes its archive under a key of
-- its own (datarequest/<request id>/<attempt>), claimed in the export's row before it builds. An attempt marks the
-- export ready only while its claim is still the row's, so of two overlapping attempts the later claim wins; the
-- earlier one deletes only its own object, and the later deletes the earlier's when it claims. The retention job
-- deletes the object the row names. Rows from before this migration keep the key D2 wrote (datarequest/<request id>).
ALTER TABLE data_request_export ADD COLUMN object_key text;
UPDATE data_request_export SET object_key = 'datarequest/' || data_request_id::text;
ALTER TABLE data_request_export
    ADD CONSTRAINT data_request_export_object_key CHECK (status <> 'ready' OR object_key IS NOT NULL);
GRANT UPDATE (object_key) ON data_request_export TO salus_app;
