-- +goose Up
-- The questionnaire library (docs/questionnaire, package 1; SPEC.md §2 "Tables", "Versions and lifecycle"):
-- a clinic's questionnaires (FHIR Questionnaire), one row per version, and the default questionnaire of a
-- practitioner role. No personal data: clinic documents and a role setting, with no author (who changed what
-- is in the AuditEvents).
--   - A series is one questionnaire's versions (series_id, its canonical url); version 1, 2, …. A version is
--     a draft, then active (published, never changed again), then retired. At most one draft and one active
--     version per series.
--   - items is the FHIR item tree as the service validates it (internal/questionnaire).
-- Row-level secured in the clinic scope (app.organization_id, neither patient setting set): staff read and
-- write their own clinic's rows. The patient scope's policy (the versions a patient's own responses
-- reference) comes with questionnaire_response (package 3). NULLIF: an unset setting reads as '' on a pooled
-- connection and must match nothing.

CREATE TABLE questionnaire (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    published_at timestamptz,
    retired_at timestamptz,
    series_id uuid NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    version integer NOT NULL CHECK (version >= 1),
    version_id integer DEFAULT 1 NOT NULL,
    status text NOT NULL CHECK (status IN ('draft', 'active', 'retired')),
    title text NOT NULL CHECK (btrim(title) <> ''),
    description text DEFAULT '' NOT NULL,
    items jsonb DEFAULT '[]' NOT NULL CHECK (jsonb_typeof(items) = 'array'),
    CONSTRAINT questionnaire_series_version_key UNIQUE (series_id, version),
    -- For questionnaire_response's reference (package 3), so a response is always in its version's clinic.
    CONSTRAINT questionnaire_id_organization_key UNIQUE (id, organization_id),
    CONSTRAINT questionnaire_published CHECK ((status = 'draft') = (published_at IS NULL)),
    CONSTRAINT questionnaire_retired CHECK ((status = 'retired') = (retired_at IS NOT NULL))
);

CREATE UNIQUE INDEX questionnaire_one_draft ON questionnaire (series_id) WHERE status = 'draft';
CREATE UNIQUE INDEX questionnaire_one_active ON questionnaire (series_id) WHERE status = 'active';
-- The library page and Questionnaire?status=…: a clinic's versions.
CREATE INDEX questionnaire_organization ON questionnaire (organization_id, status);

ALTER TABLE questionnaire ENABLE ROW LEVEL SECURITY;
CREATE POLICY questionnaire_clinic ON questionnaire
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- Whatever a query asks, only a draft is deleted: a published version is kept while anything references it.
CREATE POLICY questionnaire_delete_draft_only ON questionnaire AS RESTRICTIVE FOR DELETE
    USING (status = 'draft');

-- Which clinic, series and version a row is never change; a draft's content and the lifecycle do.
GRANT SELECT, INSERT, DELETE ON questionnaire TO salus_app;
GRANT UPDATE (status, title, description, items, published_at, retired_at, updated_at, version_id) ON questionnaire TO salus_app;

CREATE TABLE questionnaire_default (
    practitioner_role_id uuid NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL,
    -- The series, not a version: a newly published version applies to the next booking.
    series_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    CONSTRAINT questionnaire_default_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id)
);

ALTER TABLE questionnaire_default ENABLE ROW LEVEL SECURITY;
CREATE POLICY questionnaire_default_clinic ON questionnaire_default
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT, DELETE ON questionnaire_default TO salus_app;
GRANT UPDATE (series_id, updated_at, version_id) ON questionnaire_default TO salus_app;
